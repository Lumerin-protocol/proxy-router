package blockchain

import "github.com/ethereum/go-ethereum/crypto"

const (
	contractCreatedSig             = "contractCreated(address,string)"
	ClonefactoryContractPurchased  = "clonefactoryContractPurchased(address)"
	contractPurchasedSig           = "contractPurchased(address)"
	contractClosedSig              = "contractClosed()"
	contractPurchaseInfoUpdatedSig = "purchaseInfoUpdated()"
	contractCipherTextUpdatedSig   = "cipherTextUpdated(string)" // purchased contract was edited by the buyer
)

var (
	ContractCreatedHash               = crypto.Keccak256Hash([]byte(contractCreatedSig))
	ClonefactoryContractPurchasedHash = crypto.Keccak256Hash([]byte(ClonefactoryContractPurchased))
	ContractPurchasedHash             = crypto.Keccak256Hash([]byte(contractPurchasedSig))
	ContractClosedSigHash             = crypto.Keccak256Hash([]byte(contractClosedSig))
	ContractPurchaseInfoUpdatedHash   = crypto.Keccak256Hash([]byte(contractPurchaseInfoUpdatedSig))
	ContractCipherTextUpdatedHash     = crypto.Keccak256Hash([]byte(contractCipherTextUpdatedSig))

	ContractCreatedHex               = ContractCreatedHash.Hex()               // 0x1b08e1646439b7521399d47f93ab6b1ebc92803e155d0b2f2ad2d4702a050804
	ClonefactoryContractPurchasedHex = ClonefactoryContractPurchasedHash.Hex() // 0xbf1df41b401a1bb8d9bd03fb6fe59b6ced0e61a76cdd3d3d511b4d06eb2cdebe
	ContractPurchasedHex             = ContractPurchasedHash.Hex()             // 0x0c00d1d6cea0bd55f7d3b6e92ef60237b117b050185fc2816c708fd45f45e5bb
	ContractClosedSigHex             = ContractClosedSigHash.Hex()             // 0xf5e1a452bb76d7335225182a97ad694be2c7b4b5d75dcffb67ddf15db95f4844
	ContractPurchaseInfoUpdatedHex   = ContractPurchaseInfoUpdatedHash.Hex()   // 0x03e052767f275c0c51cc93a76255d42498341feb7a5beef7cc11fd57c5b66818
	ContractCipherTextUpdatedHex     = ContractCipherTextUpdatedHash.Hex()     // 0x2301ef7d9f42b857543faf9e285b5807e028d4ae99810ea7fe0aadda3a717e9d
)
