package connections

import (
	"net"

	"go.uber.org/zap"
)

type minerConn struct {
	minerConn net.Conn
	log       *zap.SugaredLogger
}
