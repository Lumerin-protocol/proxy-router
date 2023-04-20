package contractmanager

import "time"

type ContractState = uint8

const CYCLE_DURATION_DEFAULT = 30 * time.Second

const (
	ContractStateAvailable ContractState = iota // contract was created and the system is following its updates
	ContractStatePurchased                      // contract was purchased but not yet picked up by miners
	ContractStateRunning                        // contract is fulfilling
)
