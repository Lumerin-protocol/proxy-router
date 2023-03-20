package contractdata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/TitanInd/hashrouter/lib"
)

func TestDestUrlDecryption(t *testing.T) {
	// On buyer side on purchase you get public key, encrypt destination.
	// On create contract Seller set public key, then buyer get it.
	data, _ := decryptDestination(EncryptedDestSample, PrivateKeySample)
	t.Log(data)
	assert.Equal(t, DecryptedDestSample, data, "Incorrect decoded url")
}

func TestContractDataDecryption(t *testing.T) {
	contractAddr := lib.GetRandomAddr()
	contractData := NewContractData(
		contractAddr,
		lib.GetRandomAddr(),
		lib.GetRandomAddr(),
		0,
		100,
		0,
		100,
		3600,
		time.Now().Unix(),
		EncryptedDestSample,
	)

	contractDataDecr, err := DecryptContractData(contractData, PrivateKeySample)
	assert.NoError(t, err)

	dest := lib.MustParseDest(DecryptedDestSample)
	expectedDest := lib.NewDest(contractAddr.String(), "", dest.GetHost(), nil)

	assert.Equal(t, expectedDest.String(), contractDataDecr.GetDest().String())
	assert.Equal(t, contractData.Addr.Hex(), contractDataDecr.Addr.Hex())
}
