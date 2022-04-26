package connectionmanager

import (
	"errors"
	"net"
)

var ErrVirtSocClosed = errors.New("virtual socket: socket closed")
var ErrVirtSocTargetNotResponding = errors.New("virtual socket: Target not responding")
var ErrVirtSocTargetRejecting = errors.New("virtual socket: Target Rejecting")
var ErrVirtSocListenAddrBusy = errors.New("virtual socket: Listen Port Address Busy")
var ErrVirtSocIPAddrBusy = errors.New("virtual socket: Listen IP Address Busy")

type TargetURL string

type VirtSocketStatusStruct struct {
	bytesRead    int
	bytesWritten int
	countRead    int
	countWrite   int
}

type VirtListenerStatusStruct struct {
	connectionCount int
}

type VirtSocketStruct struct {
	socket *net.TCPConn
	status VirtSocketStatusStruct
}

type VirtSocketListenStruct struct {
	listener *net.TCPListener
	status   VirtListenerStatusStruct
}

//
// Opens a listening socket on the port and IP address of the local system
//
func Listen(port uint16) (v VirtSocketListenStruct, e error) {

	// Error checking here
	// Is the port already being listened too?

	return v, e
}

//
// Closes down a listening Socket
//
func (v *VirtSocketListenStruct) Close() (e error) {

	return e
}

//
//
//
func (v *VirtSocketListenStruct) Status() (vlss VirtListenerStatusStruct, e error) {

	return vlss, e
}

//
//
//
func Dial(target TargetURL) (vss VirtSocketStruct, e error) {

	return vss, e
}

//
//
//
func (s *VirtSocketStruct) Read(buf []byte) (count int, e error) {

	return count, e
}

//
//
//
func (s *VirtSocketStruct) Write(buf []byte) (count int, e error) {

	return count, e
}

//
//
//
func (v *VirtSocketStruct) Status() (s VirtSocketStatusStruct, e error) {

	return s, e
}

//
//
//
func (v *VirtSocketStruct) Close() (e error) {

	return e
}
