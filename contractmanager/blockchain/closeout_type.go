package blockchain

type CloseoutType uint8

const (
	CloseoutTypeCancel       CloseoutType = 0 // to be triggered by the buyer or validator if a contract needs to be canceled early for any reason
	CloseoutTypeOnlyWithdraw              = 1 // to be triggered by the seller to withdraw funds at any time during the smart contracts lifecycle (contract is not closing)
	CloseoutTypeWithoutClaim              = 2 // closeout without claiming funds
	CloseoutTypeWithClaim                 = 3 // closeout with claiming funds
)
