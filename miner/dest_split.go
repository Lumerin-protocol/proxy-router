package miner

import (
	"fmt"
	"log"
	"sync"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

const MinPercentage = 0.05

type Split struct {
	ID         string
	Percentage float64 // percentage of total miner power, value in range from 0 to 1
	Dest       interfaces.IDestination
}

type DestSplit struct {
	split []Split // array of the percentages of splitted hashpower, total should be less than 1
	mutex sync.RWMutex
}

func NewDestSplit() *DestSplit {
	return &DestSplit{}
}

func (d *DestSplit) Allocate(ID string, percentage float64, dest interfaces.IDestination) (*Split, error) {
	adjustedPercentage := d.adjustPercentage(percentage)
	return d.allocate(ID, adjustedPercentage, dest)
}

func (d *DestSplit) IncreaseAllocation(ID string, percentage float64) bool {
	for i, item := range d.split {
		if item.ID == ID {
			d.split[i] = Split{
				ID:         ID,
				Percentage: item.Percentage + percentage,
				Dest:       item.Dest,
			}
			return true
		}
	}
	return false
}

// adjustPercentage prevents from setting too low percentage
// TODO: adjust it so minimum time will be larger than 2 minutes
func (d *DestSplit) adjustPercentage(percentage float64) float64 {
	if percentage < MinPercentage {
		return MinPercentage
	}
	if percentage > 1-MinPercentage {
		return 1
	}
	return percentage
}

// allocate is used adjustPercentage is called for percentage
func (d *DestSplit) allocate(ID string, percentage float64, dest interfaces.IDestination) (*Split, error) {
	if percentage > 1 || percentage == 0 {
		return nil, fmt.Errorf("percentage should be withing range 0..1")
	}

	if percentage > d.GetUnallocated() {
		return nil, fmt.Errorf("total allocated value will exceed 1")
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()
	newSplit := []Split{{
		ID:         ID,
		Percentage: percentage,
		Dest:       dest},
	}
	// TODO: check if already allocated to this destination
	d.split = append(newSplit, d.split...)

	return &(newSplit[0]), nil // returning pointer to be used for further deletion
}

func (d *DestSplit) AllocateRemaining(ID string, dest interfaces.IDestination) {
	remaining := d.GetUnallocated()
	if remaining == 0 {
		return
	}
	_, err := d.allocate(ID, remaining, dest)
	if err != nil {
		log.Println(fmt.Errorf("allocateRemaining failed: %s", err))
	}
}

func (d *DestSplit) GetByID(ID string) (Split, bool) {
	for _, item := range d.split {
		if item.ID == ID {
			return item, true
		}
	}
	return Split{}, false
}

func (d *DestSplit) SetFractionByID(ID string, fraction float64) bool {
	for i, item := range d.split {
		if item.ID == ID {
			d.split[i] = Split{
				ID:         ID,
				Percentage: fraction,
				Dest:       item.Dest,
			}
			return true
		}
	}
	return false
}

func (d *DestSplit) RemoveByID(ID string) bool {
	for i, item := range d.split {
		if item.ID == ID {
			d.split = append(d.split[:i], d.split[i+1:]...)
			return true
		}
	}
	return false
}

func (d *DestSplit) GetAllocated() float64 {
	var total float64 = 0

	d.mutex.RLock()
	defer d.mutex.RUnlock()
	for _, spl := range d.split {
		total += spl.Percentage
	}
	return total
}

func (d *DestSplit) GetUnallocated() float64 {
	return 1 - d.GetAllocated()
}

func (d *DestSplit) Iter() []Split {
	return d.split
}

func (d *DestSplit) Copy() *DestSplit {
	newSplit := make([]Split, len(d.split))
	for i, v := range d.split {
		newSplit[i] = Split{
			Percentage: v.Percentage,
			Dest:       v.Dest,
		}
	}

	return &DestSplit{
		split: newSplit,
		mutex: *new(sync.RWMutex),
	}
}
