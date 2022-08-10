package interfaces

import "gitlab.com/TitanInd/hashrouter/interop"

type IBlockchainWallet interface {
	GetPrivateKey() string
	GetAddress() (interop.BlockchainAddress, error)
}
