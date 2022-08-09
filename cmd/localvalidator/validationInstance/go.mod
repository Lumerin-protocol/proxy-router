module example.com/validationInstance

go 1.16

replace example.com/blockHeader => ../blockHeader

replace example.com/hashing => ../hashing

require (
	example.com/blockHeader v0.0.0-00010101000000-000000000000
	example.com/message v0.0.0-00010101000000-000000000000
	github.com/btcsuite/btcd v0.22.0-beta // indirect
)

replace example.com/chainhash => ../chainhash

replace example.com/wire => ../wire

replace example.com/message => ../message
