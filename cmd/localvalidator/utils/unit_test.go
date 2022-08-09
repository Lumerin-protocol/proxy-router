package utils

import "testing"
import "bytes"

func TestParseIncomingHash(t *testing.T) {
	myMessage := "12345|0x1212122"
	expectedNonce := uint(12345)
	expectedHash := []byte("0x1212122")
	nonce, hash := ParseIncomingHash(myMessage)

	if nonce != expectedNonce {
		t.Error("nonce is incorrect")
	}

	if !bytes.Equal(hash, expectedHash) {
		t.Error("hash is incorrect")
	}
}

//need to test ParseValidatorCreationMessage
//1. 4 different messages; create, validate, update, and getInfo
//1. each test should have a predefined mapping, and compare each field to the output
