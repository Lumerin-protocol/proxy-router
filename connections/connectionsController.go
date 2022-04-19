package connections

import (
	"bufio"
	"fmt"
	"log"
	"net"
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
						// _, minerConnectionWriteError := minerConnection.Write(poolBuffer)

						if minerConnectionWriteError != nil {
							log.Printf("miner connection write error: %v", minerConnectionWriteError)

							defer minerConnection.Close()

							continue
						}

						log.Printf("miner < pool: %v", string(poolMessage))
						// log.Printf("miner < pool: %v", string(poolBuffer))
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

func NewConnectionsController(poolAddr string, miningRequestProcessor interfaces.IMiningRequestProcessor) *ConnectionsController {
	return &ConnectionsController{poolAddr: poolAddr, miningRequestProcessor: miningRequestProcessor}
}
