package miner

import (
	"fmt"
	"sync"
)

type DestSplit struct {
	split []split // array of the percentages of splitted hashpower, total should be less than 100
	mutex sync.RWMutex
}

type split struct {
	percentage   uint8  // percentage of total miner power, value in range from 1 to 100
	destAddr     string // the address of destination pool
	destUser     string
	destPassword string
}

func (d *DestSplit) Allocate(percentage uint8, destAddr, destUser, destPassword string) error {
	if percentage > 100 {
		panic("percentage should be withing range 1..100")
	}

	if percentage > d.GetUnallocated() {
		return fmt.Errorf("total allocated value will exceed 100 percent")
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()
	// TODO: check if already allocated to this destination
	d.split = append(d.split, split{
		percentage,
		destAddr,
		destUser,
		destPassword,
	})

	return nil
}

func (d *DestSplit) Deallocate(destAddr, destUser string) (ok bool) {
	for i, spl := range d.split {
		if spl.destAddr == destAddr && spl.destUser == destUser {
			newLength := len(d.split) - 1
			d.split[i] = d.split[newLength] // puts last element in place of the deleted one
			d.split = d.split[:newLength]   // pops last element
			return true
		}
	}
	return false
}

func (d *DestSplit) AllocateRemaining(destAddr, destUser, destPassword string) {
	remaining := d.GetAllocated()
	if remaining == 0 {
		return
	}
	d.Allocate(remaining, destAddr, destUser, destPassword)
}

func (d *DestSplit) GetAllocated() uint8 {
	var total uint8 = 0

	d.mutex.RLock()
	defer d.mutex.RUnlock()
	for _, spl := range d.split {
		total += spl.percentage
	}
	return total
}

func (d *DestSplit) GetUnallocated() uint8 {
	return 100 - d.GetAllocated()
}
