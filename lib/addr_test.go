package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddrShort(t *testing.T) {
	assert.Equal(t, "0x60E..ec2", AddrShort("0x60EbdC73d89a9f02D1cA0EbcD842650873c4dec2"), "should correctly shorten address")
	assert.Equal(t, "test", AddrShort("test"), "shorter strings should remain")
}
