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

type SourceStats struct {
	WeAcceptedShares       uint64 // shares that passed our validator (incl AcceptedUsRejectedThem)
	WeRejectedShares       uint64 // shares that failed during validation (incl RejectedUsAcceptedThem)
	WeAcceptedTheyRejected uint64 // shares that passed our validator, but rejected by the destination
	WeRejectedTheyAccepted uint64 // shares that failed our validator, but accepted by the destination
}

func (s *SourceStats) Copy() *SourceStats {
	return &SourceStats{
		WeAcceptedShares:       s.WeAcceptedShares,
		WeRejectedShares:       s.WeRejectedShares,
		WeAcceptedTheyRejected: s.WeAcceptedTheyRejected,
		WeRejectedTheyAccepted: s.WeRejectedTheyAccepted,
	}
}
