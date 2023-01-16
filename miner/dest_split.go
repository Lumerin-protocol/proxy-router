package miner

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/lib"
)

type DestSplit struct {
	split []SplitItem // array of the percentages of splitted hashpower, total should be less than 1
}

type SplitItem struct {
	ID       string  // external identifier of split item, can be ContractID
	Fraction float64 // fraction of total miner power, value in range from 0 to 1
	Dest     interfaces.IDestination
	OnSubmit interfaces.IHashrate
}

func NewDestSplit() *DestSplit {
	return &DestSplit{}
}

func (d *DestSplit) Copy() *DestSplit {
	newSplit := make([]SplitItem, len(d.split))

	for i, v := range d.split {
		newSplit[i] = SplitItem{
			ID:       v.ID,
			Fraction: v.Fraction,
			Dest:     v.Dest,
			OnSubmit: v.OnSubmit,
		}
	}

	return &DestSplit{
		split: newSplit,
	}
}

func (d *DestSplit) Allocate(ID string, percentage float64, dest interfaces.IDestination, onSubmit interfaces.IHashrate) (*DestSplit, error) {
	if percentage > 1 || percentage == 0 {
		return nil, fmt.Errorf("percentage(%.2f) should be withing range 0..1", percentage)
	}

	if percentage > d.GetUnallocated() {
		return nil, fmt.Errorf("total allocated value will exceed 1")
	}

	newDestSplit := d.Copy()

	sp := SplitItem{
		ID:       ID,
		Fraction: percentage,
		Dest:     dest,
		OnSubmit: onSubmit,
	}

	newDestSplit.split = append(newDestSplit.split, sp)

	return newDestSplit, nil
}

func (d *DestSplit) UpsertFractionByID(ID string, fraction float64, dest interfaces.IDestination, onSubmit interfaces.IHashrate) (*DestSplit, error) {
	destSplit, ok := d.SetFractionByID(ID, fraction, dest, onSubmit)
	if ok {
		return destSplit, nil
	}
	return d.Allocate(ID, fraction, dest, onSubmit)
}

func (d *DestSplit) SetFractionByID(ID string, fraction float64, dest interfaces.IDestination, onSubmit interfaces.IHashrate) (*DestSplit, bool) {
	newDestSplit := d.Copy()

	for i, item := range newDestSplit.split {
		if item.ID == ID {
			newDestSplit.split[i] = SplitItem{
				ID:       ID,
				Fraction: fraction,
				Dest:     dest,
				OnSubmit: onSubmit,
			}
			return newDestSplit, true
		}
	}
	return newDestSplit, false
}

func (d *DestSplit) AllocateRemaining(ID string, dest interfaces.IDestination, onSubmit interfaces.IHashrate) *DestSplit {
	newDestSplit := d.Copy()
	remaining := newDestSplit.GetUnallocated()
	if remaining == 0 {
		return newDestSplit
	}

	// skipping error check because Allocate validates fraction value, which is always correct in this case
	destSplit, _ := newDestSplit.Allocate(ID, remaining, dest, onSubmit)

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
		total += spl.Fraction
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
	w := tabwriter.NewWriter(b, 1, 1, 1, ' ', 0)
	fmt.Fprintf(w, "\nN\tContractID\tFraction\tDestination")

	for i, item := range d.split {
		fmt.Fprintf(w, "\n%d\t%s\t%.2f\t%s", i, lib.AddrShort(item.ID), item.Fraction, item.Dest.String())
	}
	_ = w.Flush()
	return b.String()
}
