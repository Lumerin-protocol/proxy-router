package contract

import (
	"fmt"
	"math"
	"net/url"
	"sync"
	"time"

	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/lib"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate"
	hashrateContract "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/allocator"
	hr "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"go.uber.org/atomic"
	"golang.org/x/exp/slices"
)

var (
	ErrNotRunningBlockchain = fmt.Errorf("contract is not running in blockchain")
	ErrStopped              = fmt.Errorf("contract is stopped")
	ErrAlreadyRunning       = fmt.Errorf("contract is already running")
)

var (
	AdjustmentThresholdGHS = 100.0
)

type ContractWatcherSellerV2 struct {
	// config
	contractCycleDuration time.Duration

	// state
	stats             *stats
	startCh           chan struct{}
	stopCh            chan struct{}
	doneCh            chan struct{}
	cycleEndsAt       time.Time
	minerConnectCh    *lib.ChanRecvStop[allocator.MinerItem]
	minerDisconnectCh *lib.ChanRecvStop[allocator.MinerItem]
	deliveryLog       *DeliveryLog

	// shared state
	fulfillmentStartedAt atomic.Value // time.Time
	starvingGHS          atomic.Uint64
	err                  *atomic.Error

	isRunning      bool
	isRunningMutex sync.RWMutex

	// deps
	*hashrate.Terms
	allocator *allocator.Allocator
	hrFactory func() *hr.Hashrate
	log       interfaces.ILogger
}

func NewContractWatcherSellerV2(terms *hashrateContract.Terms, cycleDuration time.Duration, hashrateFactory func() *hr.Hashrate, allocator *allocator.Allocator, log interfaces.ILogger) *ContractWatcherSellerV2 {
	return &ContractWatcherSellerV2{
		contractCycleDuration: cycleDuration,
		stats: &stats{
			actualHRGHS: hashrateFactory(),
		},
		isRunning:   false,
		startCh:     make(chan struct{}),
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
		err:         atomic.NewError(nil),
		deliveryLog: NewDeliveryLog(),
		Terms:       terms,
		allocator:   allocator,
		hrFactory:   hashrateFactory,
		log:         log,
	}
}

func (p *ContractWatcherSellerV2) StartFulfilling() error {
	p.isRunningMutex.Lock()
	defer p.isRunningMutex.Unlock()
	p.Reset()

	p.isRunning = true

	go func() {
		p.log.Infof("contract %s started", p.ID())

		err := p.run()
		p.err.Store(err)
		if err != nil && err != ErrStopped {
			p.log.Errorf("contract %s stopped with error: %s", p.ID(), err)
		}
		close(p.doneCh)
		p.log.Infof("contract stopped")

		p.isRunningMutex.Lock()
		p.isRunning = false
		p.isRunningMutex.Unlock()
	}()

	return nil
}

func (p *ContractWatcherSellerV2) StopFulfilling() {
	p.isRunningMutex.Lock()
	defer p.isRunningMutex.Unlock()

	if !p.isRunning {
		return
	}

	close(p.stopCh)
	p.log.Infof("contract %s stopping", p.ID())
}

func (p *ContractWatcherSellerV2) Done() <-chan struct{} {
	return p.doneCh
}

func (p *ContractWatcherSellerV2) Err() error {
	return p.err.Load()
}

// Reset resets the contract state
func (p *ContractWatcherSellerV2) Reset() {
	p.stats = &stats{
		actualHRGHS: p.hrFactory(),
	}
	p.isRunning = false
	p.startCh = make(chan struct{})
	p.stopCh = make(chan struct{})
	p.doneCh = make(chan struct{})
	p.err = atomic.NewError(nil)
}

func (p *ContractWatcherSellerV2) run() error {
	p.stats = &stats{
		jobFullMiners:          atomic.NewUint64(0),
		jobPartialMiners:       atomic.NewUint64(0),
		sharesFullMiners:       atomic.NewUint64(0),
		sharesPartialMiners:    atomic.NewUint64(0),
		globalUnderDeliveryGHS: atomic.NewInt64(0),
		fullMiners:             make([]string, 0),
		partialMiners:          make([]string, 0),
		actualHRGHS:            p.hrFactory(),
		deliveryTargetGHS:      0,
	}

	p.minerConnectCh = lib.NewChanRecvStop[allocator.MinerItem]()
	p.minerDisconnectCh = lib.NewChanRecvStop[allocator.MinerItem]()
	defer p.minerConnectCh.Stop()
	defer p.minerDisconnectCh.Stop()

	minerListener := p.allocator.AddVettedListener(func(minerID string) {
		p.minerConnectCh.Send(allocator.MinerItem{
			ID: minerID,
		})
	})
	defer p.allocator.RemoveVettedListener(minerListener)

	p.stats.actualHRGHS.Reset()
	p.stats.actualHRGHS.Start()
	now := time.Now()
	p.fulfillmentStartedAt.Store(&now)
	p.stats.deliveryTargetGHS = p.HashrateGHS()

CONTRACT_CYCLE:
	for {
		p.log.Debugf("new contract cycle started")
		if !p.isRunningBlockchain() {
			return ErrNotRunningBlockchain
		}
		if p.isTimeExpired() {
			return nil
		}

		p.stats.partialMiners = p.stats.partialMiners[:0]
		p.stats.jobFullMiners.Store(0)
		p.stats.jobPartialMiners.Store(0)
		p.stats.sharesFullMiners.Store(0)
		p.stats.sharesPartialMiners.Store(0)

		p.cycleEndsAt = time.Now().Add(p.contractCycleDuration)

		p.logDeliveryTarget()
		p.stats.deliveryTargetGHS -= p.adjustHashrate(p.stats.deliveryTargetGHS)
		p.logDeliveryTarget()

	EVENTS_CONTROLLER:
		for {
			select {
			// new miner connected
			case miner := <-p.minerConnectCh.Receive():
				p.log.Infof("got miner connect event: %s", miner.ID)

				p.logDeliveryTarget()
				p.stats.deliveryTargetGHS -= p.adjustHashrate(p.stats.deliveryTargetGHS)
				p.logDeliveryTarget()

				continue EVENTS_CONTROLLER

			// contract miner disconnected
			case minerItem := <-p.minerDisconnectCh.Receive():
				p.log.Infof("got miner disconnect event: %s", minerItem.ID)

				p.logDeliveryTarget()
				p.stats.deliveryTargetGHS -= p.replaceMiner(minerItem)
				p.logDeliveryTarget()

				continue EVENTS_CONTROLLER

			// shorter loop if not enough hashrate
			case <-time.After(10 * time.Second):
				if int(p.stats.deliveryTargetGHS) > 0 {
					p.log.Debugf("not enough hashrate: trying to allocate more")

					p.logDeliveryTarget()
					p.stats.deliveryTargetGHS -= p.adjustHashrate(p.stats.deliveryTargetGHS)
					p.logDeliveryTarget()

				}
				continue EVENTS_CONTROLLER

			// contract ended
			case <-time.After(p.getEndsAfter()):
				elapsedCycleDuration := p.contractCycleDuration - p.remainingCycleDuration()
				p.onCycleEnd(elapsedCycleDuration) // to log the last cycle
				p.removeAllMiners()
				p.reportTotalStats()
				return nil

			// contract stopped from outside
			case <-p.stopCh:
				elapsedCycleDuration := p.contractCycleDuration - p.remainingCycleDuration()
				p.onCycleEnd(elapsedCycleDuration) // to log the last cycle
				p.removeAllMiners()
				p.reportTotalStats()
				return ErrStopped

			// contract cycle ended
			case <-time.After(p.remainingCycleDuration()):
				p.onCycleEnd(p.contractCycleDuration)
				continue CONTRACT_CYCLE
			}
		}
	}
}

func (p *ContractWatcherSellerV2) onCycleEnd(cycleDuration time.Duration) {
	thisCycleActualGHS := hr.JobSubmittedToGHSV2(p.stats.totalJob(), cycleDuration)
	thisCycleUnderDeliveryGHS := p.HashrateGHS() - thisCycleActualGHS
	p.stats.globalUnderDeliveryGHS.Add(int64(thisCycleUnderDeliveryGHS))
	p.stats.deliveryTargetGHS = p.HashrateGHS() - p.getFullMinersHR() + float64(p.stats.globalUnderDeliveryGHS.Load())

	logEntry := DeliveryLogEntry{
		Timestamp:                         time.Now(),
		ActualGHS:                         int(thisCycleActualGHS),
		FullMinersGHS:                     int(hr.JobSubmittedToGHSV2(float64(p.stats.jobFullMiners.Load()), cycleDuration)),
		FullMiners:                        lib.CopySlice(p.stats.fullMiners),
		FullMinersShares:                  int(p.stats.sharesFullMiners.Load()),
		PartialMinersGHS:                  int(hr.JobSubmittedToGHSV2(float64(p.stats.jobPartialMiners.Load()), cycleDuration)),
		PartialMiners:                     lib.CopySlice(p.stats.partialMiners),
		PartialMinersShares:               int(p.stats.sharesPartialMiners.Load()),
		UnderDeliveryGHS:                  int(thisCycleUnderDeliveryGHS),
		GlobalHashrateGHS:                 int(p.stats.actualHRGHS.GetHashrateAvgGHSAll()["mean"]),
		GlobalUnderDeliveryGHS:            int(p.stats.globalUnderDeliveryGHS.Load()),
		GlobalError:                       1 - p.stats.actualHRGHS.GetHashrateAvgGHSAll()["mean"]/p.HashrateGHS(),
		NextCyclePartialDeliveryTargetGHS: int(p.stats.deliveryTargetGHS),
	}
	p.deliveryLog.AddEntry(logEntry)

	p.log.Infof("contract cycle ended %+v", logEntry)
}

// adjustHashrate adjusts the hashrate of the contract by adding/removing full miners and allocating partial miners
// if hashrateGHS > 0 the allocation increases, if hashrateGHS < 0 the allocation decreases
// returns the amount of hashrateGHS that was added or removed (with negative sign)
func (p *ContractWatcherSellerV2) adjustHashrate(hashrateGHS float64) (adjustedGHS float64) {
	expectedAdjustmentGHS := hashrateGHS
	fullMinerThresholdGHS := 1000.0
	partialMinersThresholdGHS := 100.0

	adjustmentRequired := math.Abs(hashrateGHS) > AdjustmentThresholdGHS
	if !adjustmentRequired {
		p.starvingGHS.Store(0)
		return 0
	}

	if hashrateGHS < -fullMinerThresholdGHS {
		hashrateGHS += p.removeFullMiners(hashrateGHS)
	}

	if hashrateGHS > fullMinerThresholdGHS {
		hashrateGHS -= p.addFullMiners(hashrateGHS)
	}

	remainingCycleDuration := p.remainingCycleDuration()

	if hashrateGHS > partialMinersThresholdGHS {
		job := hr.GHSToJobSubmittedV2(hashrateGHS, remainingCycleDuration)
		addedJob := p.addPartialMiners(job, remainingCycleDuration)
		addedGHS := hr.JobSubmittedToGHSV2(addedJob, remainingCycleDuration)
		hashrateGHS -= addedGHS
		p.log.Debugf("added %.f GHS of partial miners", addedGHS)
	}

	starvingGHS := uint64(hashrateGHS)
	p.starvingGHS.Store(starvingGHS)
	if starvingGHS > 0 {
		p.log.Warnf("not enough hashrate to fulfill contract (lacking %d GHS)", starvingGHS)
	}

	deltaGHS := expectedAdjustmentGHS - hashrateGHS
	p.log.Debugf("adjustment delta %.f GHS", deltaGHS)
	return deltaGHS
}

// addFullMiners adds full miners, they persist for the duration of the contract
func (p *ContractWatcherSellerV2) addFullMiners(hashrateGHS float64) (addedGHS float64) {
	fullMiners, remainderGHS := p.allocator.AllocateFullMinersForHR(
		p.ID(),
		hashrateGHS,
		p.getAdjustedDest(),
		p.Duration(),
		p.stats.onFullMinerShare,
		func(ID string, hashrateGHS float64, remainingJob float64) {
			p.log.Warn("full miner disconnected", ID)
			p.minerDisconnectCh.Send(allocator.MinerItem{
				ID:           ID,
				HrGHS:        hashrateGHS,
				JobRemaining: remainingJob,
			})
		},
	)
	if len(fullMiners) > 0 {
		p.stats.addFullMiners(fullMiners...)
	}
	p.log.Infof("added %d full miners, addedGHS %.f", len(fullMiners), hashrateGHS-remainderGHS)
	p.log.Infof("full miners: %v", p.stats.fullMiners)
	return hashrateGHS - remainderGHS
}

// removeFullMiners removes full miners, cause they persist for the duration of the contract
func (p *ContractWatcherSellerV2) removeFullMiners(hrGHS float64) (removedGHS float64) {
	items := p.getFullMinersSorted()

	if len(items) == 0 {
		p.log.Warnf("no miners found to be removed")
	}

	for _, item := range items {
		minerToRemove := item.ID
		miner, ok := p.allocator.GetMiners().Load(minerToRemove)
		if ok {
			miner.RemoveTasksByID(p.ID())
			removedGHS = +miner.HashrateGHS()
		}
		p.stats.removeFullMiner(minerToRemove)
		if hrGHS-removedGHS < 0 {
			break
		}
	}

	p.log.Debugf("removed %d full miners, removedGHS %.f", len(items)-len(p.stats.fullMiners), removedGHS)
	p.log.Debugf("full miners: %v", p.stats.fullMiners)
	return removedGHS
}

// addPartialMiners adds partial miners, they allocated for one cycle
func (p *ContractWatcherSellerV2) addPartialMiners(job float64, cycleEndTimeout time.Duration) (addedJob float64) {
	miners, remainderJob := p.allocator.AllocatePartialForJob(
		p.ID(),
		job,
		p.getAdjustedDest(),
		cycleEndTimeout,
		func(diff float64, ID string) {
			p.stats.onPartialMinerShare(diff, ID)
			actualCycleGHS := hr.JobSubmittedToGHSV2(p.stats.totalJob(), p.contractCycleDuration)
			expectedCycleGHS := p.HashrateGHS() + float64(p.stats.globalUnderDeliveryGHS.Load())
			if actualCycleGHS >= expectedCycleGHS {
				p.log.Infof("this cycle reached target prematurely actualGHS %.f expectedGHS %.f", actualCycleGHS, expectedCycleGHS)
				// TODO: potential race if new partial miner is added when removePartialMiners is reading
				p.removeAllPartialMiners()
			}
		},
		func(ID string, hrGHS float64, remainingJob float64) {
			p.log.Warn("partial miner disconnected", ID)
			p.minerDisconnectCh.Send(allocator.MinerItem{
				ID:           ID,
				HrGHS:        hrGHS,
				JobRemaining: remainingJob,
			})
		},
	)

	if len(miners) > 0 {
		for minerID := range miners {
			p.stats.addPartialMiners(minerID)
		}
	}

	p.log.Debugf("added %d partial miners", len(miners))
	p.log.Debugf("partial miners: %v", p.stats.partialMiners)

	return job - remainderJob
}

func (p *ContractWatcherSellerV2) removeAllMiners() {
	p.removeAllFullMiners()
	p.removeAllPartialMiners()
}

func (p *ContractWatcherSellerV2) removeAllFullMiners() {
	for _, minerID := range p.getAvailableFullMiners() {
		miner, ok := p.allocator.GetMiners().Load(minerID)
		if !ok {
			continue
		}
		miner.RemoveTasksByID(p.ID())
		p.log.Debugf("full miner %s was removed from this contract", miner.ID())
	}
	return
}

func (p *ContractWatcherSellerV2) removeAllPartialMiners() {
	for _, minerID := range p.stats.partialMiners {
		miner, ok := p.allocator.GetMiners().Load(minerID)
		if !ok {
			continue
		}
		miner.RemoveTasksByID(p.ID())
		p.log.Debugf("partial miner %s was removed from this contract", miner.ID())
	}
}

func (p *ContractWatcherSellerV2) replaceMiner(minerItem allocator.MinerItem) (adjustedGHS float64) {
	p.log.Debugf("replacing miner %s, %.f GHS", minerItem.ID, minerItem.HrGHS)

	isFullMiner := p.stats.removeFullMiner(minerItem.ID)
	isPartialMiner := p.stats.removePartialMiner(minerItem.ID)

	if isFullMiner {
		p.stats.deliveryTargetGHS += minerItem.HrGHS
		p.log.Debugf("miner %s is full miner", minerItem.ID)
	}

	if isPartialMiner {
		p.log.Debugf("miner %s is partial miner", minerItem.ID)
		remainingGHS := hr.JobSubmittedToGHSV2(minerItem.JobRemaining, p.remainingCycleDuration())
		p.stats.deliveryTargetGHS += remainingGHS
	}

	return p.adjustHashrate(p.stats.deliveryTargetGHS)
}

func (p *ContractWatcherSellerV2) reportTotalStats() {
	expectedJob := hr.GHSToJobSubmittedV2(p.HashrateGHS(), p.Duration())
	actualJob := p.stats.actualHRGHS.GetTotalWork()
	undeliveredJob := expectedJob - actualJob
	undeliveredFraction := undeliveredJob / expectedJob

	p.log.Infof("contract ended, undelivered work %d, undelivered fraction %.2f",
		int(undeliveredJob), undeliveredFraction)
}

func (p *ContractWatcherSellerV2) isRunningBlockchain() bool {
	return p.BlockchainState() == hashrateContract.BlockchainStateRunning
}

func (p *ContractWatcherSellerV2) isTimeExpired() bool {
	return p.EndTime().Before(time.Now())
}

// getAdjustedDest returns the destination url with the username set to the contractID
// this is required for the buyer to distinguish incoming hashrate between different contracts
func (p *ContractWatcherSellerV2) getAdjustedDest() *url.URL {
	if p.Terms.Dest() == nil {
		return nil
	}
	dest := lib.CopyURL(p.Terms.Dest())
	lib.SetUserName(dest, p.Terms.ID())
	return dest
}

func (p *ContractWatcherSellerV2) getFullMinersSorted() []*allocator.MinerItem {
	items := make([]*allocator.MinerItem, 0, len(p.stats.fullMiners))

	for _, minerID := range p.stats.fullMiners {
		miner, ok := p.allocator.GetMiners().Load(minerID)
		if !ok {
			continue
		}
		items = append(items, &allocator.MinerItem{
			ID:    miner.ID(),
			HrGHS: miner.HashrateGHS(),
		})
	}

	slices.SortStableFunc(items, func(a, b *allocator.MinerItem) bool {
		return a.HrGHS < b.HrGHS
	})

	if len(items) < len(p.stats.fullMiners) {
		var minerIDs []string
		for _, miner := range items {
			minerIDs = append(minerIDs, miner.ID)
		}
		p.stats.fullMiners = minerIDs
	}

	return items
}

func (p *ContractWatcherSellerV2) getAvailableFullMiners() []string {
	newFullMiners := make([]string, 0, len(p.stats.fullMiners))
	for _, minerID := range p.stats.fullMiners {
		_, ok := p.allocator.GetMiners().Load(minerID)
		if !ok {
			continue
		}
		newFullMiners = append(newFullMiners, minerID)
	}
	if len(newFullMiners) != len(p.stats.fullMiners) {
		p.stats.fullMiners = newFullMiners
	}
	return p.stats.fullMiners
}

func (p *ContractWatcherSellerV2) getFullMinersHR() float64 {
	miners := p.getFullMinersSorted()
	totalGHS := 0.0
	for _, miner := range miners {
		totalGHS += miner.HrGHS
	}
	return totalGHS
}

func (p *ContractWatcherSellerV2) getEndsAfter() time.Duration {
	endTime := p.EndTime()
	if endTime.IsZero() {
		return 0
	}
	return endTime.Sub(time.Now())
}

func (p *ContractWatcherSellerV2) remainingCycleDuration() time.Duration {
	return p.cycleEndsAt.Sub(time.Now())
}

//
// Public getters
//

// constants
func (p *ContractWatcherSellerV2) Role() resources.ContractRole {
	return resources.ContractRoleSeller
}

func (p *ContractWatcherSellerV2) ResourceType() string {
	return ResourceTypeHashrate
}

func (p *ContractWatcherSellerV2) ValidationStage() hashrateContract.ValidationStage {
	return hashrateContract.ValidationStageNotApplicable // only for buyer
}

// state getters
func (p *ContractWatcherSellerV2) FulfillmentStartTime() time.Time {
	return p.fulfillmentStartedAt.Load().(time.Time)
}
func (p *ContractWatcherSellerV2) ResourceEstimatesActual() map[string]float64 {
	return p.stats.actualHRGHS.GetHashrateAvgGHSAll()
}
func (p *ContractWatcherSellerV2) GetDeliveryLogs() ([]DeliveryLogEntry, error) {
	return p.deliveryLog.GetEntries()
}
func (p *ContractWatcherSellerV2) State() resources.ContractState {
	p.isRunningMutex.RLock()
	defer p.isRunningMutex.RUnlock()

	if p.isRunning {
		return resources.ContractStateRunning
	}
	return resources.ContractStatePending
}
func (p *ContractWatcherSellerV2) IsRunning() bool {
	p.isRunningMutex.RLock()
	defer p.isRunningMutex.RUnlock()
	return p.isRunning
}
func (p *ContractWatcherSellerV2) StarvingGHS() int {
	return int(p.starvingGHS.Load())
}

// terms getters
func (p *ContractWatcherSellerV2) Dest() string {
	if dest := p.getAdjustedDest(); dest != nil {
		return dest.String()
	}
	return ""
}
func (p *ContractWatcherSellerV2) ResourceEstimates() map[string]float64 {
	return map[string]float64{
		ResourceEstimateHashrateGHS: p.Terms.HashrateGHS(),
	}
}
func (p *ContractWatcherSellerV2) ShouldBeRunning() bool {
	return p.Terms.BlockchainState() == hashrate.BlockchainStateRunning
}

// terms setters
func (p *ContractWatcherSellerV2) SetTerms(terms *hashrate.Terms) {
	p.isRunningMutex.RLock()
	defer p.isRunningMutex.RUnlock()

	if p.isRunning {
		p.log.Warnf("cannot update contract terms while running, terms will apply after closeout")
		return
	}

	p.Terms = terms
	p.log.Infof("contract terms updated: price %.f LMR, hashrate %.f GHS, duration %s, state %s", terms.PriceLMR(), terms.HashrateGHS(), terms.Duration().Round(time.Second), terms.BlockchainState().String())
}

func (p *ContractWatcherSellerV2) logDeliveryTarget() {
	p.log.Debugf("deliveryTarget %.0f GHS", p.stats.deliveryTargetGHS)
}