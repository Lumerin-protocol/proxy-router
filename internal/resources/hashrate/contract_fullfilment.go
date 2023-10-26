package hashrate

type ContractFullfilment struct {
	ContractID          string
	UndeliveredJob      float64
	UndeliveredFraction float64
	ActualJob           float64
	ExpectedJob         float64
	Terms               Terms
}

func NewContractFullfillment(contractID string, UndeliveredJob float64,
	undeliveredFraction float64,
	actualJob float64,
	expectedJob float64, terms Terms) *ContractFullfilment {
	return &ContractFullfilment{
		ContractID:          contractID,
		UndeliveredJob:      UndeliveredJob,
		UndeliveredFraction: undeliveredFraction,
		ActualJob:           actualJob,
		ExpectedJob:         expectedJob,
		Terms:               terms,
	}
}
