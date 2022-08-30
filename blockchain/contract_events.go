package blockchain

import "github.com/ethereum/go-ethereum/crypto"

const (
	contractCreatedSig             = "contractCreated(address,string)"
	contractPurchasedSig           = "contractPurchased(address)"
	contractClosedSig              = "contractClosed()"
	contractPurchaseInfoUpdatedSig = "purchaseInfoUpdated()"
	contractCipherTextUpdatedSig   = "cipherTextUpdated(string)"
)

var (
	ContractCreatedHex             = crypto.Keccak256Hash([]byte(contractCreatedSig)).Hex()
	ContractPurchasedHex           = crypto.Keccak256Hash([]byte(contractPurchasedSig)).Hex()
	ContractClosedSigHex           = crypto.Keccak256Hash([]byte(contractClosedSig)).Hex()
	ContractPurchaseInfoUpdatedHex = crypto.Keccak256Hash([]byte(contractPurchaseInfoUpdatedSig)).Hex()
	ContractCipherTextUpdatedHex   = crypto.Keccak256Hash([]byte(contractCipherTextUpdatedSig)).Hex()
)
