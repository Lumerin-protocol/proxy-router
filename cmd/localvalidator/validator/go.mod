module example.com/validator

go 1.16

replace example.com/hashing => ../hashing

replace example.com/blockHeader => ../blockHeader

require (
	example.com/blockHeader v0.0.0-00010101000000-000000000000
	example.com/channels v0.0.0-00010101000000-000000000000
	example.com/message v0.0.0-00010101000000-000000000000
	example.com/utils v0.0.0-00010101000000-000000000000
	example.com/validationInstance v0.0.0-00010101000000-000000000000
	github.com/btcsuite/btcd v0.22.0-beta // indirect
)

replace example.com/validationInstance => ../validationInstance

replace example.com/channels => ../channels

replace example.com/message => ../message

replace example.com/utils => ../utils

replace example.com/chainhash => ../chainhash

replace example.com/wire => ../wire
