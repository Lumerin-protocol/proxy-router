package msgbus

import (
	"testing"

	"gitlab.com/TitanInd/lumerin/cmd/log"
)

var (
	l                = log.New()
	ps       *PubSub = New(1, l)
	host             = "127.0.0.1"
	port             = "3334"
	username         = "someusername"
	testurl          = "stratum+tcp://" + username + ":@" + host + ":" + port + "/"
)

func TestDestPubWait(t *testing.T) {

	var dest Dest
	dest.ID = DestID(GetRandomIDString())
	dest.NetUrl = DestNetUrl(testurl)

	d, err := ps.DestPubWait(dest)

	if err != nil {
		t.Errorf("DestPubWait returned error: %s\n", err)
	}

	if d.ID != dest.ID {
		t.Errorf("DestPubWait returned wrong ID: %s != %s\n", d.ID, dest.ID)
	}

}

// func (ps *PubSub) DestGetWait(id DestID) (dest *Dest, err error) {
func TestDestGetWait(t *testing.T) {

	var dest Dest
	dest.ID = DestID(GetRandomIDString())
	dest.NetUrl = DestNetUrl(testurl)

	_, err := ps.DestPubWait(dest)

	if err != nil {
		t.Errorf("DestPubWait returned error: %s\n", err)
	}

	destget, err := ps.DestGetWait(dest.ID)

	if err != nil {
		t.Errorf("DestPubWait returned error: %s\n", err)
	}

	if destget.ID != dest.ID {
		t.Errorf("DestGetWait returned wrong ID: %s != %s\n", destget.ID, dest.ID)
	}

}

func TestHost(t *testing.T) {

	var dest Dest

	dest.ID = DestID(GetRandomIDString())
	dest.NetUrl = DestNetUrl(testurl)
	h := dest.Host()

	if host != h {
		t.Errorf("Got %s, wanted %s", h, host)
	}

}

func TestPort(t *testing.T) {

	var dest Dest

	dest.ID = DestID(GetRandomIDString())
	dest.NetUrl = DestNetUrl(testurl)
	p := dest.Port()

	if port != p {
		t.Errorf("Got %s, wanted %s", p, port)
	}

}

func TestUsername(t *testing.T) {

	var dest Dest

	dest.ID = DestID(GetRandomIDString())
	dest.NetUrl = DestNetUrl(testurl)
	u := dest.Username()

	if username != u {
		t.Errorf("Got %s, wanted %s", u, username)
	}

}
