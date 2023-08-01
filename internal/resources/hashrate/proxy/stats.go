package proxy

type DestStats struct {
	WeAcceptedTheyAccepted uint64 // our validator accepted and dest accepted
	WeAcceptedTheyRejected uint64 // our validator accepted and dest rejected
	WeRejectedTheyAccepted uint64 // our validator rejected, but dest accepted
}

func (s *DestStats) Copy() *DestStats {
	return &DestStats{
		WeAcceptedTheyAccepted: s.WeAcceptedTheyAccepted,
		WeAcceptedTheyRejected: s.WeAcceptedTheyRejected,
		WeRejectedTheyAccepted: s.WeRejectedTheyAccepted,
	}
}
