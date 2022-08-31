package hashrate

import (
	"math"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func ConvertMerkleBranchesToRoot(merkle_branches []string) (*chainhash.Hash, error) {
	if len(merkle_branches) == 0 {
		var zeroReturn *chainhash.Hash
		return zeroReturn, nil
	}

	nextPOT := nextPowerOfTwo(len(merkle_branches))
	arraySize := nextPOT*2 - 1
	merkle_branch_hashes := make([]*chainhash.Hash, arraySize)
	for i, br := range merkle_branches {
		//iterate through list of strings and convert each item to a chainhash.Hash
		c_hash, err := chainhash.NewHashFromStr(br)
		if err != nil {
			return nil, err
		}
		merkle_branch_hashes[i] = c_hash
	}

	offset := nextPOT
	for i := 0; i < arraySize-1; i += 2 {
		switch {
		// When there is no left child node, the parent is nil too.
		case merkle_branch_hashes[i] == nil:
			merkle_branch_hashes[offset] = nil

		// When there is no right child, the parent is generated by
		// hashing the concatenation of the left child with itself.
		case merkle_branch_hashes[i+1] == nil:
			newHash := blockchain.HashMerkleBranches(merkle_branch_hashes[i], merkle_branch_hashes[i])
			merkle_branch_hashes[offset] = newHash

		// The normal case sets the parent node to the double sha256
		// of the concatenation of the left and right children.
		default:
			newHash := blockchain.HashMerkleBranches(merkle_branch_hashes[i], merkle_branch_hashes[i+1])
			merkle_branch_hashes[offset] = newHash
		}
		offset++
	}

	//convert each chainhash.Hash to a wire.MsgTx

	return merkle_branch_hashes[len(merkle_branch_hashes)-1], nil
}

func nextPowerOfTwo(n int) int {
	// Return the number if it's already a power of 2.
	if n&(n-1) == 0 {
		return n
	}

	// Figure out and return the next power of two.
	exponent := uint(math.Log2(float64(n))) + 1
	return 1 << exponent // 2^exponent
}
