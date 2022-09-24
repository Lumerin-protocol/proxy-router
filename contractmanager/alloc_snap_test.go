package contractmanager

import "testing"

func TestAllocSnap(t *testing.T) {
	snap := NewAllocSnap()
	snap.Set("miner1", "contract1", 0.5, 10000)
	snap.Set("miner1", "contract2", 0.4, 20000)
	snap.Set("miner2", "contract1", 0.3, 30000)

	allocSnap, ok := snap.Get("miner1", "contract2")
	if !ok {
		t.Fatalf("allocSnap not found")
	}
	if allocSnap.Fraction != 0.4 {
		t.Fatalf("invalid fraction")
	}

	allocCollection, ok := snap.Contract("contract1")
	if !ok {
		t.Fatalf("allocCollection not found")
	}
	if len(allocCollection) != 2 {
		t.Fatalf("invalid allocCollection length")
	}

	allocCollection2, ok := snap.Miner("miner1")
	if !ok {
		t.Fatalf("allocCollection not found")
	}
	if len(allocCollection2) != 2 {
		t.Fatalf("invalid allocCollection length")
	}
}
