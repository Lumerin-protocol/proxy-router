package blockHeader

import "testing"
import "fmt"

//each test serializes the information in the blockheader with the provided nonce
//then the result is double hashed Sha256(Sha256(serialization))
//then the result is compared to the blockHash result

func TestBlock1(t *testing.T) {
	blockHash := "06e533fd1ada86391f3f6c343204b0d278d4aaec1c0b20aa27ba030000000000"

	nonce := "274148111"
	time := "4d1b2237"

	blockHeader := BlockHeader{
		Version:           "00000001",
		MerkleRoot:        "f3e94742aca4b5ef85488dc37c06c3282295ffec960994b2c0d5ac2a25a95766",
		Time:              "4d1b2237",
		Difficulty:        "1b04864c",
		PreviousBlockHash: "000000000002d01c1fccc21636b607dfd930d31d01c3a62104612a1719011250",
	}

	resultHash := blockHeader.HashInput(nonce, time)

	if fmt.Sprintf("%x", resultHash) != blockHash {
		t.Error(fmt.Printf("expected blockhash: %s\nCalculated blockhash: %x\n", blockHash, resultHash))
	}
}

func TestBlock2(t *testing.T) {
	blockHash := "bf0e2e13fce62f3a5f15903a177ad6a258a01f164aefed7d4a03000000000000"
	nonce := "4158183488"
	time := "505d96e7"
	blockHeader := BlockHeader{
		Version:           "00000002",
		MerkleRoot:        "a08f8101f50fd9c9b3e5252aff4c1c1bd668f878fffaf3d0dbddeb029c307e88",
		Time:              "1348310759",
		Difficulty:        "1a05db8b",
		PreviousBlockHash: "00000000000003a20def7a05a77361b9657ff954b2f2080e135ea6f5970da215",
	}

	resultHash := blockHeader.HashInput(nonce, time)

	if fmt.Sprintf("%x", resultHash) != blockHash {
		t.Error(fmt.Printf("expected blockhash: %s\nCalculated blockhash: %s\n", blockHash, resultHash))
	}
}

func TestBlock3(t *testing.T) {
	blockHash := "5472ac8b1187bfcf91d6d218bbda1eb2405d7c55f1f8cc820000000000000000"
	nonce := "222771801"
	time := "536dc802"
	blockHeader := BlockHeader{
		Version:           "00000002",                                                         //big-endian
		MerkleRoot:        "915c887a2d9ec3f566a648bedcf4ed30d0988e22268cfe43ab5b0cf8638999d3", //big-endian
		Time:              "1399703554",                                                       //regular time
		Difficulty:        "1900896c",                                                         //big-endian
		PreviousBlockHash: "000000000000000067ecc744b5ae34eebbde14d21ca4db51652e4d67e155f07e", //big-endian
	}

	resultHash := blockHeader.HashInput(nonce, time)

	if fmt.Sprintf("%x", resultHash) != blockHash {
		t.Error(fmt.Printf("expected blockhash: %s\nCalculated blockhash: %x\n", blockHash, resultHash))
	}
}

func TestBlock4(t *testing.T) {
	blockHash := "6f3f6e272452ff8f324bedc2dd1cedf1e62f73e46c46ec040000000000000000"
	nonce := "657220870"
	time := "56cf2acc"
	blockHeader := BlockHeader{
		Version:           "00000004",
		MerkleRoot:        "b0e8f88d4fb7cbc49ab49a3a43c368550e22a8e9e3e04b15e34240306a53aeec",
		Time:              "1456417484",
		Difficulty:        "1806b99f",
		PreviousBlockHash: "0000000000000000030034b661aed920a9bdf6bbfa6d2e7a021f78481882fa39",
	}

	resultHash := blockHeader.HashInput(nonce, time)

	if fmt.Sprintf("%x", resultHash) != blockHash {
		t.Error(fmt.Printf("expected blockhash: %s\nCalculated blockhash: %x\n", blockHash, resultHash))
	}
}

func TestBlock300_000(t *testing.T) {
	/*
	   block 300000 data obtained from a raspibolt lnd node using unix command bitcoin-cli getBlockInfo $(bitcoin-cli getBlockHash 300000)
	   hashes are represented in big-endian format but converted to little-endian format for testing reasons
	   "hash": "000000000000000082ccf8f1557c5d40b21edabb18d2d691cfbf87118bac7254",
	   "confirmations": 421369,
	   "strippedsize": 128810,
	   "size": 128810,
	   "weight": 515240,
	   "height": 300000,
	   "version": 2,
	   "versionHex": "00000002",
	   "merkleroot": "915c887a2d9ec3f566a648bedcf4ed30d0988e22268cfe43ab5b0cf8638999d3",
	   "tx": [
	   ],
	   "time": 1399703554,
	   "mediantime": 1399701278,
	   "nonce": 222771801,
	   "bits": "1900896c",
	   "difficulty": 8000872135.968163,
	   "chainwork": "000000000000000000000000000000000000000000005a7b3c42ea8b844374e9",
	   "nTx": 237,
	   "previousblockhash": "000000000000000067ecc744b5ae34eebbde14d21ca4db51652e4d67e155f07e",
	   "nextblockhash": "000000000000000049a0914d83df36982c77ac1f65ade6a52bdced2ce312aba9"
	*/
	blockHash := "5472ac8b1187bfcf91d6d218bbda1eb2405d7c55f1f8cc820000000000000000"
	nonce := "222771801"
	time := "536dc802"
	blockHeader := BlockHeader{
		Version:           "00000002",
		MerkleRoot:        "915c887a2d9ec3f566a648bedcf4ed30d0988e22268cfe43ab5b0cf8638999d3",
		Time:              "1399703554",
		Difficulty:        "1900896c",
		PreviousBlockHash: "000000000000000067ecc744b5ae34eebbde14d21ca4db51652e4d67e155f07e",
	}

	resultHash := blockHeader.HashInput(nonce, time)

	if fmt.Sprintf("%x", resultHash) != blockHash {
		t.Error(fmt.Printf("expected blockhash: %s\nCalculated blockhash: %x\n", blockHash, resultHash))
	}
}
