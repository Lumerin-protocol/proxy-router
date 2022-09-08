package lumerinlib

type NetworkType string

const TCP NetworkType = "tcp"
const TCP4 NetworkType = "tcp4"
const TCP6 NetworkType = "tcp6"
const UDP NetworkType = "udp"
const UDP4 NetworkType = "udp4"
const UDP6 NetworkType = "udp6"
const TRUNK NetworkType = "trunk"
const TCPTRUNK NetworkType = "tcptrunk"
const UDPTRUNK NetworkType = "udptrunk"
const ANYAVAILABLE NetworkType = "anyavailable"

type NetAddr struct {
	ipaddr  string
	network NetworkType
}

func (na *NetAddr) Network() string {
	return string(na.network)
}

func (na *NetAddr) String() string {
	return na.ipaddr
}

func NewNetAddr(net NetworkType, ip string) (na *NetAddr) {

	// Error checking here

	na = &NetAddr{
		ipaddr:  ip,
		network: net,
	}

	return na
}
