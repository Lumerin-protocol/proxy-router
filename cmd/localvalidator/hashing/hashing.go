package hashing

//this package me be subject to removal since all hashing is now taken care of in the chainhash and wire packages

import (
	"crypto/sha256"
	"encoding/hex"
)

// function to hash the block header and return a hash
func hashBlockHeader(BlockHeader []byte) [32]byte {
	//need to modify this to be hashed twice
	hash := sha256.Sum256(BlockHeader) //returns a [32]byte array
	//hash the result
	return sha256.Sum256(hash[:])
}

// function to compare 2 hashes and determine if they're equal or not
// todo, find the most efficient means of comparing the two lists
func compareTwoHashes(hash1 [32]byte, hash2 [32]byte) bool {
	return hash1 == hash2
}

//intakes the block header and a submitted hash, output is whether the block header hashes to the submitted hash
func ValidateBlock(BlockHeader []byte, submittedHash [32]byte) bool {
	calculatedHash := hashBlockHeader(BlockHeader)
	isValid := compareTwoHashes(calculatedHash, submittedHash)
	return isValid
}

func HashBlockHeader(BlockHeader []byte) string {
	hash := hashBlockHeader(BlockHeader)
	return hex.EncodeToString(hash[:])
}
