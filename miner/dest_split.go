package miner

import (
	"fmt"
	"math"
	"sync"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

// TODO: consider storing percentage in float64 to simplify the code
var AllocationPrecision uint8 = 10 // the precision for percentages, should be a divisor of 100

type DestSplit struct {
	split []Split // array of the percentages of splitted hashpower, total should be less than 100
	mutex sync.RWMutex
}

type Split struct {
	Percentage uint8 // percentage of total miner power, value in range from 1 to 100
	Dest       interfaces.IDestination
	Parent     *DestSplit
}

func (s *Split) Deallocate() bool {
	return s.Parent.Deallocate(s)
}

func NewDestSplit() *DestSplit {
	return &DestSplit{}
}

func (d *DestSplit) Allocate(percentage float64, dest interfaces.IDestination) (*Split, error) {
	adjustedPercentage := d.adjustPercentage(percentage)
	return d.allocate(adjustedPercentage, dest)
}

// Deallocate accepts pointer to the allocated split and deallocates it from the miner
func (d *DestSplit) Deallocate(split *Split) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for i, item := range d.split {
		if &item == split {
			d.split = append(d.split[:i], d.split[i+1:]...)
			return true
		}
	}
	return false
}

// adjustPercentage reduces precision of percentage according to AllocationPrecision
// to avoid changing destination for short periods of time. it always rounds up
func (d *DestSplit) adjustPercentage(percentage float64) uint8 {
	return uint8(math.Ceil(percentage*100/float64(AllocationPrecision))) * AllocationPrecision
}

// allocate is used adjustPercentage is called for percentage
func (d *DestSplit) allocate(percentage uint8, dest interfaces.IDestination) (*Split, error) {
	if percentage > 100 || percentage == 0 {
		panic("percentage should be withing range 1..100")
	}

	if percentage > d.GetUnallocated() {
		return nil, fmt.Errorf("total allocated value will exceed 100 percent")
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()
	newSplit := Split{
		percentage,
		dest,
		d,
	}
	// TODO: check if already allocated to this destination
	d.split = append(d.split, newSplit)

	return &newSplit, nil // returning pointer to be used for further deletion
}

func (d *DestSplit) AllocateRemaining(dest interfaces.IDestination) {
	remaining := d.GetUnallocated()
	if remaining == 0 {
		return
	}
	d.allocate(remaining, dest)
}

func (d *DestSplit) GetAllocated() uint8 {
	var total uint8 = 0

	d.mutex.RLock()
	defer d.mutex.RUnlock()
	for _, spl := range d.split {
		total += spl.Percentage
	}
	return total
}

func (d *DestSplit) GetUnallocated() uint8 {
	return 100 - d.GetAllocated()
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
