package contract

import (
	hr "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/hashrate"
	"go.uber.org/atomic"
)

type stats struct {
	jobFullMiners          *atomic.Uint64
	jobPartialMiners       *atomic.Uint64
	sharesFullMiners       *atomic.Uint64
	sharesPartialMiners    *atomic.Uint64
	globalUnderDeliveryGHS *atomic.Int64
	fullMiners             []string
	partialMiners          []string
	deliveryTargetGHS      float64
	actualHRGHS            *hr.Hashrate
}

func (s *stats) onFullMinerShare(diff float64, ID string) {
	s.jobFullMiners.Add(uint64(diff))
	s.actualHRGHS.OnSubmit(diff)
	s.sharesFullMiners.Add(1)
}

func (s *stats) onPartialMinerShare(diff float64, ID string) {
	s.jobPartialMiners.Add(uint64(diff))
	s.actualHRGHS.OnSubmit(diff)
	s.sharesPartialMiners.Add(1)
}

func (s *stats) addFullMiners(IDs ...string) {
	s.fullMiners = append(s.fullMiners, IDs...)
}

func (s *stats) removeFullMiner(ID string) (ok bool) {
	if len(s.fullMiners) == 0 {
		return
	}
	newFullMiners := make([]string, 0, len(s.fullMiners)-1)
	for _, minerID := range s.fullMiners {
		if minerID == ID {
			ok = true
			continue
		}
		newFullMiners = append(newFullMiners, minerID)
	}
	s.fullMiners = newFullMiners
	return
}

func (s *stats) addPartialMiners(IDs ...string) {
	s.partialMiners = append(s.partialMiners, IDs...)
}

func (s *stats) removePartialMiner(ID string) (ok bool) {
	if len(s.partialMiners) == 0 {
		return
	}
	newPartialMiners := make([]string, 0, len(s.partialMiners)-1)
	for _, minerID := range s.partialMiners {
		if minerID == ID {
			ok = true
			continue
		}
		newPartialMiners = append(newPartialMiners, minerID)
	}
	s.partialMiners = newPartialMiners
	return
}

func (c *stats) totalJob() float64 {
	return float64(c.jobFullMiners.Load()) + float64(c.jobPartialMiners.Load())
}
