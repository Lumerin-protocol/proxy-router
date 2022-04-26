package msgbus

import (
	"fmt"
	"net"
	"net/url"

	"gitlab.com/TitanInd/lumerin/lumerinlib"
)

type DestNetUrl string

type DestID IDString

const DEFAULT_DEST_ID DestID = "DefaultDestID"

type Dest struct {
	ID     DestID
	NetUrl DestNetUrl
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) DestPubWait(dest Dest) (d Dest, err error) {

	if dest.ID == "" {
		dest.ID = DestID(GetRandomIDString())
	}

	event, err := ps.PubWait(DestMsg, IDString(dest.ID), dest)
	if err != nil || event.Err != nil {
		panic(fmt.Sprintf(lumerinlib.Funcname()+"Unable to add Record %s, %s\n", err, event.Err))
	}

	d = event.Data.(Dest)
	if err != nil || event.Err != nil {
		fmt.Printf(lumerinlib.Funcname()+" PubWait returned err: %s, %s\n", err, event.Err)
	}

	return d, err
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) DestGetWait(id DestID) (dest *Dest, err error) {

	if id == "" {
		panic(fmt.Sprintf(lumerinlib.FileLine() + " empty DestID passed in\n"))
	}

	event, err := ps.GetWait(DestMsg, IDString(id))
	if err != nil || event.Err != nil {
		fmt.Printf(lumerinlib.Funcname()+" ID not found %s, %s\n", err, event.Err)
	}

	switch event.Data.(type) {
	case Dest:
		d := event.Data.(Dest)
		dest = &d
	case *Dest:
		dest = event.Data.(*Dest)
	default:
		dest = nil
	}

	return dest, err
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (d *Dest) NetAddr() (addr net.Addr, e error) {

	// Assum TCP for the moment.

	tcp, e := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", d.Host(), d.Port()))
	addr = net.Addr(tcp)

	return addr, e
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (ps *PubSub) DestExistsWait(id DestID) bool {

	dest, _ := ps.DestGetWait(id)

	return dest != nil
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (d *Dest) Host() (host string) {

	u, err := url.Parse(string(d.NetUrl))
	if err != nil {
		panic(fmt.Sprintf(lumerinlib.FileLine()+"url: %s, err %s\n", u, err))
	}

	host, _, err = net.SplitHostPort(u.Host)
	if err != nil {
		panic(fmt.Sprintf(lumerinlib.FileLine()+"url: %s, err %s\n", u, err))
	}

	return host
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (d *Dest) Port() (port string) {

	u, err := url.Parse(string(d.NetUrl))
	if err != nil {
		panic(fmt.Sprintf(lumerinlib.FileLine()+"url: %s, err %s\n", u, err))
	}

	_, port, err = net.SplitHostPort(u.Host)
	if err != nil {
		panic(fmt.Sprintf(lumerinlib.FileLine()+"url: %s, err %s\n", u, err))
	}

	return port
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (d *Dest) Username() string {

	if d == nil {
		return ""
	}

	u, err := url.Parse(string(d.NetUrl))
	if err != nil {
		panic(fmt.Sprintf(lumerinlib.FileLine()+"url: %s, err %s\n", u, err))
	}

	return u.User.Username()
}

//---------------------------------------------------------------
//
//---------------------------------------------------------------
func (d *Dest) Password() string {

	u, err := url.Parse(string(d.NetUrl))
	if err != nil {
		panic(fmt.Sprintf(lumerinlib.FileLine()+"url: %s, err %s\n", u, err))
	}

	pass, set := u.User.Password()
	if set {
		return pass
	} else {
		return ""
	}
}
