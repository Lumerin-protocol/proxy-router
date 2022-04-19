package connections

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

type ConnectionsController struct {
	interfaces.IConnectionsController
	interfaces.Subscriber

	miningRequestProcessor interfaces.IMiningRequestProcessor
	poolConnection         net.Conn
	poolAddr               string
	connections            []*ConnectionInfo
}

func (c *ConnectionsController) Run() {
	log.Printf("Running main...")

	link, err := net.ListenTCP("tcp", &net.TCPAddr{Port: 3333})

	if err != nil {
		log.Fatalf("Error listening to port 3333 - %v", err)
	}

	fmt.Println("proxy : listening on port 3333")

	c.connectToPool()

	for {
		minerConnection, minerConnectionError := link.Accept()

		if minerConnectionError != nil {
			log.Fatalf("miner connection accept error: %v", minerConnectionError)
		}

		log.Println("accepted miner connection")

		go func(minerConnection net.Conn) {

			minerReader := bufio.NewReader(minerConnection)

			for {

				minerBuffer, minerReadError := minerReader.ReadBytes('\n')

				if minerReadError != nil {

					log.Printf("miner connection read error: %v;  with miner buffer: %v", minerReadError, string(minerBuffer))

					defer minerConnection.Close()

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

							continue
						}

						log.Printf("miner < pool: %v", string(poolMessage))

						c.updateConnectionStatus(minerConnection)
					}
				}()
			}
		}(minerConnection)
	}
}

func (c *ConnectionsController) connectToPool() {
	poolConnection, poolConnectionError := net.DialTimeout("tcp", c.poolAddr, 30*time.Second)

	c.poolConnection = poolConnection

	if poolConnectionError != nil {
		log.Fatalf("pool connection dial error: %v", poolConnectionError)
	}

	log.Printf("connected to pool %v", c.poolAddr)
	c.createConnection(poolConnection)
}

func (c *ConnectionsController) updateConnectionStatus(workerConnection net.Conn) {
	connection := c.connections[0]
	connection.IpAddress = workerConnection.RemoteAddr().String()
	connection.Status = "Running"
}

func (c *ConnectionsController) createConnection(poolConnection net.Conn) {
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

	<-time.After(1 * time.Minute)

	log.Printf("Switching back to old pool address: %v", oldPoolAddr)

	c.poolAddr = oldPoolAddr

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

func NewConnectionsController(poolAddr string, miningRequestProcessor interfaces.IMiningRequestProcessor) *ConnectionsController {
	return &ConnectionsController{poolAddr: poolAddr, miningRequestProcessor: miningRequestProcessor, connections: []*ConnectionInfo{}}
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
