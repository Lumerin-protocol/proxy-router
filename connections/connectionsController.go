package connections

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
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

	log.Printf("Running main...")

	link, err := net.ListenTCP("tcp", &net.TCPAddr{Port: 3333})

	if err != nil {
		log.Fatalf("Error listening to port 3333 - %v", err)
	}

	fmt.Println("proxy : listening on port 3333")

	c.connectToPool()

	for {
		minerConnection, minerConnectionError := link.Accept()
		c.minerConnections = append(c.minerConnections, minerConnection)

		if minerConnectionError != nil {
			log.Fatalf("miner connection accept error: %v", minerConnectionError)
		}

		log.Println("accepted miner connection")

		go func(minerConnection net.Conn) {

			minerReader := bufio.NewReader(minerConnection)

			for {

				minerBuffer, minerReadError := minerReader.ReadBytes('\n')

				if minerReadError != nil {

					log.Printf("miner connection read error: %v;  with miner buffer: %v; address: %v", minerReadError, string(minerBuffer), minerConnection.RemoteAddr().String())

					defer minerConnection.Close()
					c.minerConnections = removeIt(minerConnection, c.minerConnections)

					break
				}

				if len(minerBuffer) <= 0 {
					log.Printf("empty message, continue...")
					continue
				}

				miningMessage := c.miningRequestProcessor.ProcessMiningMessage(minerBuffer)

				_, poolWriteError := c.poolConnection.Write(miningMessage)

				if poolWriteError != nil {
					log.Printf("pool connection write error: %v", poolWriteError)
					c.poolConnection.Close()
					c.connectToPool()
					break
				}

				log.Printf("miner > pool: %v", string(miningMessage))

				go func() {

					poolReader := bufio.NewReader(c.poolConnection)

					for {
						poolBuffer, poolReadError := poolReader.ReadBytes('\n')

						if poolReadError != nil {

							log.Printf("pool connection read error: %v", poolReadError)
							defer c.poolConnection.Close()
							c.connectToPool()
							break
						}

						if len(poolBuffer) <= 0 {
							log.Printf("empty message, continue...")
							continue
						}

						poolMessage := c.miningRequestProcessor.ProcessPoolMessage(poolBuffer)
						_, minerConnectionWriteError := minerConnection.Write(poolMessage)

						if minerConnectionWriteError != nil {
							log.Printf("miner connection write error: %v", minerConnectionWriteError)

							defer minerConnection.Close()
							c.minerConnections = removeIt(minerConnection, c.minerConnections)

							break
						}

						log.Printf("miner < pool: %v", string(poolMessage))

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
		log.Fatalf("pool connect error;  failed to parse url: % v; %v", uri, err)
	}

	log.Printf("Dialing pool at %v", uri)
	poolConnection, poolConnectionError := net.DialTimeout("tcp", c.poolAddr, 30*time.Second)

	c.poolConnection = poolConnection

	if poolConnectionError != nil {
		log.Fatalf("pool connection dial error: %v", poolConnectionError)
	}

	log.Printf("connected to pool %v", c.poolAddr)
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

	log.Printf("Switching to new pool address: %v", destinationMessage.NetUrl)

	c.connectToPool()

	<-time.After(1 * time.Minute)

	log.Printf("Switching back to old pool address: %v", oldPoolAddr)

	c.poolAddr = oldPoolAddr

	c.connectToPool()
}

func (c *ConnectionsController) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	connectionsResponse, err := json.Marshal(c.connections)

	if err != nil {
		log.Printf("API /connections error: Failed to marshal connections to json byte array; %v", "", err)
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
