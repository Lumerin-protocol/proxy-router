package interfaces

type IContractFactory interface {
	CreateContract(
		IsSeller bool,
		ID string,
		State string,
		Buyer string,
		Price int,
		Limit int,
		Speed int,
		Length int,
		StartingBlockTimestamp int,
		Dest string,
	) (IContractModel, error)
}
