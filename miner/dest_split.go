package miner

import (
	"fmt"
	"math"
	"sync"

	"gitlab.com/TitanInd/hashrouter/interop"
)

// TODO: consider storing percentage in float64 to simplify the code
var AllocationPrecision uint8 = 10 // the precision for percentages, should be a divisor of 100

type DestSplit struct {
	split []Split // array of the percentages of splitted hashpower, total should be less than 100
	mutex sync.RWMutex
}

type Split struct {
	Percentage uint8 // percentage of total miner power, value in range from 1 to 100
	Dest       interop.Dest
}

func NewDestSplit() *DestSplit {
	return &DestSplit{}
}

func (d *DestSplit) Allocate(percentage float64, dest interop.Dest) error {
	adjustedPercentage := d.adjustPercentage(percentage)
	return d.allocate(adjustedPercentage, dest)
}

// adjustPercentage reduces precision of percentage according to AllocationPrecision
// to avoid changing destination for short periods of time. it always rounds up
func (d *DestSplit) adjustPercentage(percentage float64) uint8 {
	return uint8(math.Ceil(percentage/float64(AllocationPrecision))) * AllocationPrecision
}

// allocate is used adjustPercentage is called for percentage
func (d *DestSplit) allocate(percentage uint8, dest interop.Dest) error {
	if percentage > 100 || percentage == 0 {
		panic("percentage should be withing range 1..100")
	}

	if percentage > d.GetUnallocated() {
		return fmt.Errorf("total allocated value will exceed 100 percent")
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()
	// TODO: check if already allocated to this destination
	d.split = append(d.split, Split{
		percentage,
		dest,
	})

	return nil
}

func (d *DestSplit) Deallocate(dest interop.Dest) (ok bool) {
	for i, spl := range d.split {
		if spl.Dest.Host == dest.Host && spl.Dest.User.Username() == dest.User.Username() {
			newLength := len(d.split) - 1
			d.split[i] = d.split[newLength] // puts last element in place of the deleted one
			d.split = d.split[:newLength]   // pops last element
			return true
		}
	}
	return false
}

func (d *DestSplit) AllocateRemaining(dest interop.Dest) {
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
