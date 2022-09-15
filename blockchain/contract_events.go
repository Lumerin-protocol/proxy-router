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
	ContractCreatedHash             = crypto.Keccak256Hash([]byte(contractCreatedSig))
	ContractPurchasedHash           = crypto.Keccak256Hash([]byte(contractPurchasedSig))
	ContractClosedSigHash           = crypto.Keccak256Hash([]byte(contractClosedSig))
	ContractPurchaseInfoUpdatedHash = crypto.Keccak256Hash([]byte(contractPurchaseInfoUpdatedSig))
	ContractCipherTextUpdatedHash   = crypto.Keccak256Hash([]byte(contractCipherTextUpdatedSig))
	ContractCreatedHex              = ContractCreatedHash.Hex()
	ContractPurchasedHex            = ContractPurchasedHash.Hex()
	ContractClosedSigHex            = ContractClosedSigHash.Hex()
	ContractPurchaseInfoUpdatedHex  = ContractPurchaseInfoUpdatedHash.Hex()
	ContractCipherTextUpdatedHex    = ContractCipherTextUpdatedHash.Hex()
)
