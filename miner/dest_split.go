package miner

import (
	"bytes"
	"fmt"

	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type DestSplit struct {
	split []SplitItem // array of the percentages of splitted hashpower, total should be less than 1
}

type SplitItem struct {
	ID         string  // external identifier of split item, can be ContractID
	Percentage float64 // percentage of total miner power, value in range from 0 to 1
	Dest       interfaces.IDestination
}

func NewDestSplit() *DestSplit {
	return &DestSplit{}
}

func (d *DestSplit) Copy() *DestSplit {
	newSplit := make([]SplitItem, len(d.split))

	for i, v := range d.split {
		newSplit[i] = SplitItem{
			ID:         v.ID,
			Percentage: v.Percentage,
			Dest:       v.Dest,
		}
	}

	return &DestSplit{
		split: newSplit,
	}
}

func (d *DestSplit) Allocate(ID string, percentage float64, dest interfaces.IDestination) (*DestSplit, error) {
	if percentage > 1 || percentage == 0 {
		return nil, fmt.Errorf("percentage should be withing range 0..1")
	}

	if percentage > d.GetUnallocated() {
		return nil, fmt.Errorf("total allocated value will exceed 1")
	}

	newDestSplit := d.Copy()

	sp := SplitItem{
		ID:         ID,
		Percentage: percentage,
		Dest:       dest,
	}

	newDestSplit.split = append(newDestSplit.split, sp)

	return newDestSplit, nil
}

func (d *DestSplit) UpsertFractionByID(ID string, fraction float64, dest interfaces.IDestination) (*DestSplit, error) {
	destSplit, ok := d.SetFractionByID(ID, fraction)
	if ok {
		return destSplit, nil
	}
	return d.Allocate(ID, fraction, dest)
}

func (d *DestSplit) SetFractionByID(ID string, fraction float64) (*DestSplit, bool) {
	newDestSplit := d.Copy()

	for i, item := range newDestSplit.split {
		if item.ID == ID {
			newDestSplit.split[i] = SplitItem{
				ID:         ID,
				Percentage: fraction,
				Dest:       item.Dest,
			}
			return newDestSplit, true
		}
	}
	return newDestSplit, false
}

func (d *DestSplit) AllocateRemaining(ID string, dest interfaces.IDestination) *DestSplit {
	newDestSplit := d.Copy()
	remaining := newDestSplit.GetUnallocated()
	if remaining == 0 {
		return newDestSplit
	}

	// skipping error check because Allocate validates fraction value, which is always correct in this case
	destSplit, _ := newDestSplit.Allocate(ID, remaining, dest)

	return destSplit
}

func (d *DestSplit) RemoveByID(ID string) (*DestSplit, bool) {
	newDestSplit := d.Copy()

	for i, item := range newDestSplit.split {
		if item.ID == ID {
			newDestSplit.split = append(newDestSplit.split[:i], newDestSplit.split[i+1:]...)
			return newDestSplit, true
		}
	}

	return newDestSplit, false
}

func (d *DestSplit) GetByID(ID string) (SplitItem, bool) {
	for _, item := range d.split {
		if item.ID == ID {
			return item, true
		}
	}
	return SplitItem{}, false
}

func (d *DestSplit) GetAllocated() float64 {
	var total float64 = 0

	for _, spl := range d.split {
		total += spl.Percentage
	}

	return total
}

func (d *DestSplit) GetUnallocated() float64 {
	return 1 - d.GetAllocated()
}

func (d *DestSplit) Iter() []SplitItem {
	return d.split
}

func (d *DestSplit) IsEmpty() bool {
	return len(d.split) == 0
}

func (d *DestSplit) String() string {
	var b = new(bytes.Buffer)

	fmt.Fprintf(b, "\nN\tContractID\nDestination\tPercentage")

	for i, item := range d.split {
		fmt.Fprintf(b, "\n%d\t%s\t%s\t%.2f", i, item.ID, item.Dest.String(), item.Percentage)
	}

	return b.String()
}
