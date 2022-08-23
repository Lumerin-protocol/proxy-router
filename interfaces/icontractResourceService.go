package interfaces

type IContractResourceService interface {
	Allocate(hashrate uint64, dest IDestination) error
	GetUnallocatedHashrate() (uint64, IRoutablestreamFullfillmentCollection)
}

type RunningContract struct {
	//*ContactModel
	//*Hashrate
	// []destinations
}

// func Allocate() {

// 	miners := c.RoutableStreamService.GetRoutableStreams()

// }
// func (c *Contract) Execute() {

// 	c.Allocate()
// 	for {
// 		hashrate := Scheduler.GetCurrentHashrate()

// 		if hashrate < c.Required*c.LossTolerance {
// 			Scheduler.Stop()
// 		}

// 		if c.ContractDurationComplete() {
// 			Scheduler.Stop()
// 		}

// 		time.Sleep(1000)
// 	}
// }

// func (c *Contract) Stop() {}
