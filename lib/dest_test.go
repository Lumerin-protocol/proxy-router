package lib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDestEmptyUserPass(t *testing.T) {
	host := "test.com"
	port := 123
	dest, err := ParseDest(fmt.Sprintf("stratum+tcp://:@%s:%d", host, port))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, dest.GetHost(), fmt.Sprintf("%s:%d", host, port))
	assert.Equal(t, dest.Username(), "")
	assert.Equal(t, dest.Password(), "")
}

func TestParseDestNoUserPass(t *testing.T) {
	host := "test.com"
	port := 123
	dest, err := ParseDest(fmt.Sprintf("stratum+tcp://%s:%d", host, port))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, dest.GetHost(), fmt.Sprintf("%s:%d", host, port))
	assert.Equal(t, dest.Username(), "")
	assert.Equal(t, dest.Password(), "")
}

func TestParseNoProto(t *testing.T) {
	host := "test.com"
	port := 123
	dest, err := ParseDest(fmt.Sprintf("//:@%s:%d", host, port))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, dest.GetHost(), fmt.Sprintf("%s:%d", host, port))
	assert.Equal(t, dest.Username(), "")
	assert.Equal(t, dest.Password(), "")
}

func TestParseDifferentProto(t *testing.T) {
	host := "test.com"
	port := 123
	dest, err := ParseDest(fmt.Sprintf("tcp://:@%s:%d", host, port))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, dest.GetHost(), fmt.Sprintf("%s:%d", host, port))
	assert.Equal(t, dest.Username(), "")
	assert.Equal(t, dest.Password(), "")
}
