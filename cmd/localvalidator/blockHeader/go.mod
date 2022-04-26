module example.com/block_header

go 1.16

replace example.com/wire => ../wire

require (
	example.com/chainhash v0.0.0-00010101000000-000000000000
	example.com/wire v0.0.0-00010101000000-000000000000
	github.com/btcsuite/btcd v0.22.0-beta
	golang.org/x/crypto v0.0.0-20220112180741-5e0467b6c7ce // indirect
)

replace example.com/chainhash => ../chainhash

replace example.com/hashing => ../hashing

replace example.com/blockchain => ../blockchain
