package lumerintrunk

////
//// Table of connections to other Lumerin Hosts by IP
////
//type LumerinTrunks struct {
//	trunk [ip]socket.SocketStruct
//}
//
////
//// Structue to manage Trunks to other nodes
////
//// Needs lots of work here
//type LumerinProtocolStruct struct {
//	connections LumerinTrunks
//	shutdown    chan bool
//}
//
////
//// Attached to either SocketStruct, or LumerinSocketStruct
//// to implement common functions
////
//type LumerinMessageStruct struct {
//	msgType
//	msgLenth
//	msgData
//}
//
//var ErrLumProtoClosed = errors.New("lumerin proto: socket closed")
//var ErrLumProtoTargetNotResponding = errors.New("lumerin proto: Target not responding")
//var ErrLumProtoTargetRejecting = errors.New("lumerin proto: Target Rejecting")
//var ErrLumProtoListenAddrBusy = errors.New("lumerin proto: Listen Address Busy")
//
////
//// Sets up local Lumerin Trunk Handling
//// Listening for New trunk connections
////
//func New() (lps LumerinProtocolStruct, e error) {
//
//}
//
//func (*LumerinProtocolStruct) LumerinListen(port uint16, handler NewLumerinHandler) (e error) {
//
//}
//
//func (*LumerinProtocolStruct) LumerinDial(port uint16, handler NewLumerinHandler) (e error) {
//
//}
//
