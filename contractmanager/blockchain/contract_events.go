package blockchain

import "github.com/ethereum/go-ethereum/crypto"

const (
	contractCreatedSig     = "contractCreated(address,string)"
	contractPurchasedSig   = "contractPurchased(address)"
	contractClosedSig      = "contractClosed()"
	purchaseInfoUpdatedSig = "purchaseInfoUpdated()"
	cipherTextUpdatedSig   = "cipherTextUpdated(string)"
)

var (
	ContractCreatedHex     = crypto.Keccak256Hash([]byte(contractCreatedSig)).Hex()
	ContractPurchasedHex   = crypto.Keccak256Hash([]byte(contractPurchasedSig)).Hex()
	ContractClosedSigHex   = crypto.Keccak256Hash([]byte(contractClosedSig)).Hex()
	CurchaseInfoUpdatedHex = crypto.Keccak256Hash([]byte(purchaseInfoUpdatedSig)).Hex()
	CipherTextUpdatedHex   = crypto.Keccak256Hash([]byte(cipherTextUpdatedSig)).Hex()
)
