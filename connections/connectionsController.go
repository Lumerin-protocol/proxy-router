package connections

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"go.uber.org/zap"
)

type ConnectionsController struct {
	interfaces.IConnectionsController
	interfaces.Subscriber

	miningRequestProcessor interfaces.IMiningRequestProcessor
	poolConnection         net.Conn
	minerConnections       []net.Conn
	poolAddr               string
	connections            []*ConnectionInfo
	eventManager           interfaces.IEventManager
	logger                 *zap.SugaredLogger
}

func (c *ConnectionsController) Run(ctx context.Context) error {
	c.eventManager.Attach(contractmanager.DestMsg, c)
	errCh := make(chan error, 1)
	go func() {
		errCh <- c.run(ctx)
	}()

	var err error

	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	c.logger.Info("Conroller cancelled")
	c.eventManager.DeAttach(contractmanager.DestMsg, c)
	return err
}

func (c *ConnectionsController) run(ctx context.Context) (err error) {
	port := 3333

	c.logger.Infof("Running main...")

	link, err := net.ListenTCP("tcp", &net.TCPAddr{Port: port})

	if err != nil {
		c.logger.Fatalf("Error listening to port %s - %v", port, err)
	}

	c.logger.Infof("proxy : listening on port %s", port)

	c.connectToPool()

	for {
		minerConnection, minerConnectionError := link.Accept()
		c.minerConnections = append(c.minerConnections, minerConnection)

		if minerConnectionError != nil {
			c.logger.Fatalf("miner connection accept error: %v", minerConnectionError)
		}

		c.logger.Info("accepted miner connection")

		go func(minerConnection net.Conn) {

			minerReader := bufio.NewReader(minerConnection)

			for {

				minerBuffer, minerReadError := minerReader.ReadBytes('\n')

				if minerReadError != nil {

					c.logger.Errorf("miner connection read error: %v;  with miner buffer: %v; address: %v", minerReadError, string(minerBuffer), minerConnection.RemoteAddr().String())

					defer minerConnection.Close()
					c.minerConnections = removeIt(minerConnection, c.minerConnections)

					break
				}

				if len(minerBuffer) <= 0 {
					c.logger.Warn("empty message, continue...")
					continue
				}

				miningMessage := c.miningRequestProcessor.ProcessMiningMessage(minerBuffer)

				_, poolWriteError := c.poolConnection.Write(miningMessage)

				if poolWriteError != nil {
					c.logger.Error("pool connection write error", poolWriteError)
					c.poolConnection.Close()
					c.connectToPool()
					break
				}

				c.logger.Info("miner > pool", string(miningMessage))

				go func() {

					poolReader := bufio.NewReader(c.poolConnection)

					for {
						poolBuffer, poolReadError := poolReader.ReadBytes('\n')

						if poolReadError != nil {
							c.logger.Error("pool connection read error", poolReadError)
							defer c.poolConnection.Close()
							c.connectToPool()
							break
						}

						if len(poolBuffer) <= 0 {
							c.logger.Warn("empty message, continue...")
							continue
						}

						poolMessage := c.miningRequestProcessor.ProcessPoolMessage(poolBuffer)
						_, minerConnectionWriteError := minerConnection.Write(poolMessage)

						if minerConnectionWriteError != nil {
							c.logger.Error("miner connection write error", minerConnectionWriteError)

							defer minerConnection.Close()
							c.minerConnections = removeIt(minerConnection, c.minerConnections)

							break
						}

						c.logger.Info("miner < pool", string(poolMessage))

						c.updateConnectionStatusToConnected(minerConnection)
					}
				}()
			}
		}(minerConnection)
	}
}

func (c *ConnectionsController) connectToPool() {

	uri, err := url.Parse(c.poolAddr)

	if uri.Scheme != "" && uri.Host != "" {
		c.poolAddr = fmt.Sprintf("%v%v", uri.Host, uri.Path)
	}

	if err != nil {
		c.logger.Fatal("pool connect error;  failed to parse url", uri, err)
	}

	c.logger.Infof("Dialing pool at %s", uri)
	poolConnection, poolConnectionError := net.DialTimeout("tcp", c.poolAddr, 30*time.Second)

	c.poolConnection = poolConnection

	if poolConnectionError != nil {
		c.logger.Fatal("pool connection dial error", poolConnectionError)
	}

	c.logger.Infof("connected to pool %s", c.poolAddr)
	c.setConnection(poolConnection)

	c.resetMinerConnections()
}

func (c *ConnectionsController) resetMinerConnections() {
	for _, connection := range c.minerConnections {
		connection.Close()
	}

	c.minerConnections = []net.Conn{}
}
func (c *ConnectionsController) updateConnectionStatusToConnected(workerConnection net.Conn) {
	connection := c.connections[0]
	connection.IpAddress = workerConnection.RemoteAddr().String()
	connection.Status = "Running"
}

func (c *ConnectionsController) updateConnectionStatusToDisconnected(workerConnection net.Conn) {
	connection := c.connections[0]
	connection.IpAddress = workerConnection.RemoteAddr().String()
	connection.Status = "Available"
}

func (c *ConnectionsController) setConnection(poolConnection net.Conn) {
	c.connections = []*ConnectionInfo{
		{
			Id:            "1",
			SocketAddress: poolConnection.RemoteAddr().String(),
			Status:        "Available",
		},
	}
}

func (c *ConnectionsController) Update(message interface{}) {
	destinationMessage := message.(contractmanager.Dest)

	oldPoolAddr := c.poolAddr
	c.poolAddr = destinationMessage.NetUrl

	c.logger.Infof("Switching to new pool address: %v", destinationMessage.NetUrl)

	c.connectToPool()

	<-time.After(1 * time.Minute)

	c.logger.Infof("Switching back to old pool address: %v", oldPoolAddr)

	c.poolAddr = oldPoolAddr

	c.connectToPool()
}

func (c *ConnectionsController) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	connectionsResponse, err := json.Marshal(c.connections)

	if err != nil {
		c.logger.Infof("API /connections error: Failed to marshal connections to json byte array; %v", "", err)
	}
	w.Write(connectionsResponse)
}

// func BuildConnectionInfo(worker *Worker) *ConnectionInfo {
// 	status := "Running"

// 	if worker.pool.client == nil {
// 		status = "Available"
// 	}

// 	return &ConnectionInfo{
// 		Id:            worker.id,
// 		IpAddress:     worker.addr,
// 		Status:        status,
// 		SocketAddress: worker.pool.addr,
// 	}
// }

func NewConnectionsController(poolAddr string, miningRequestProcessor interfaces.IMiningRequestProcessor, eventManager interfaces.IEventManager, logger *zap.SugaredLogger) *ConnectionsController {
	return &ConnectionsController{
		poolAddr:               poolAddr,
		miningRequestProcessor: miningRequestProcessor,
		connections:            []*ConnectionInfo{},
		minerConnections:       []net.Conn{},
		eventManager:           eventManager,
		logger:                 logger,
	}
}

type ConnectionInfo struct {
	Id            string
	IpAddress     string `json:"ipAddress"`
	Status        string `json:"status"`
	SocketAddress string `json:"socketAddress"`
	Total         string `json:"total"`
	Accepted      string `json:"accepted"`
	Rejected      string `json:"rejected"`
}

func removeIt(ss net.Conn, ssSlice []net.Conn) []net.Conn {
	for idx, v := range ssSlice {
		if v == ss {
			return append(ssSlice[0:idx], ssSlice[idx+1:]...)
		}
	}
	return ssSlice
}
