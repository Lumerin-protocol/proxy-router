package proxy

import (
	"context"
	"time"

	globalInterfaces "gitlab.com/TitanInd/proxy/proxy-router-v3/internal/interfaces"
	"gitlab.com/TitanInd/proxy/proxy-router-v3/internal/resources/hashrate/proxy/interfaces"
)

// ConnSource is a miner connection, a wrapper around StratumConnection
// that adds miner specific state variables
type ConnSource struct {
	// state
	workerName string

	extraNonce     string // last relevant extraNonce (from subscribe or set_extranonce)
	extraNonceSize int

	versionRollingMask        string // original supported rolling mask from the miner
	versionRollingMinBitCount int    // originally sent from the miner
	currentVersionRollingMask string // current rolling mask after negotiation with server

	// deps
	log  globalInterfaces.ILogger
	conn *StratumConnection
}

func NewSourceConn(conn *StratumConnection, log globalInterfaces.ILogger) *ConnSource {
	return &ConnSource{
		conn: conn,
		log:  log,
	}
}

func (c *ConnSource) Read(ctx context.Context) (interfaces.MiningMessageGeneric, error) {
	//TODO: message validation
	return c.conn.Read(ctx)
}

func (c *ConnSource) Write(ctx context.Context, msg interfaces.MiningMessageGeneric) error {
	//TODO: message validation
	return c.conn.Write(ctx, msg)
}

func (c *ConnSource) GetExtraNonce() (extraNonce string, extraNonceSize int) {
	return c.extraNonce, c.extraNonceSize
}

func (c *ConnSource) SetExtraNonce(extraNonce string, extraNonceSize int) {
	c.extraNonce, c.extraNonceSize = extraNonce, extraNonceSize
}

func (c *ConnSource) SetVersionRolling(mask string, minBitCount int) {
	c.versionRollingMask, c.versionRollingMinBitCount = mask, minBitCount
}

func (c *ConnSource) GetVersionRolling() (mask string, minBitCount int) {
	return c.versionRollingMask, c.versionRollingMinBitCount
}

// GetNegotiatedVersionRollingMask returns actual version rolling mask after negotiation with server
func (c *ConnSource) GetNegotiatedVersionRollingMask() string {
	return c.versionRollingMask
}

// SetNegotiatedVersionRollingMask sets actual version rolling mask after negotiation with server
func (c *ConnSource) SetNegotiatedVersionRollingMask(mask string) {
	c.versionRollingMask = mask
}

func (c *ConnSource) SetWorkerName(workerName string) {
	c.workerName = workerName

	c.log = c.log.Named(workerName)
	c.conn.log = c.log
}

func (c *ConnSource) GetWorkerName() string {
	return c.workerName
}

func (c *ConnSource) GetConnectedAt() time.Time {
	return c.conn.GetConnectedAt()
}
