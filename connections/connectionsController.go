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

	poolConnection net.Conn
	poolAddr       string
}

func (c *ConnectionsController) Run() {
	log.Printf("Running main...")

	link, err := net.ListenTCP("tcp", &net.TCPAddr{Port: 3333})

	if err != nil {
		log.Fatalf("Error listening to port 3333 - %v", err)
	}

	fmt.Println("proxy : listening on port 3333")

	poolConnection, poolConnectionError := net.DialTimeout("tcp", c.poolAddr, 30*time.Second)

	c.poolConnection = poolConnection

	if poolConnectionError != nil {
		log.Fatalf("pool connection dial error: %v", poolConnectionError)
	}

	log.Printf("connected to pool %v", c.poolAddr)

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
					if minerReadError.Error() == "EOF" {
						continue
					}

					log.Printf("miner connection read error: %v;  with miner buffer: %v", minerReadError, string(minerBuffer))

					// minerConnection.Close()

					break
				}

				_, poolWriteError := c.poolConnection.Write(minerBuffer)

				if poolWriteError != nil && poolWriteError.Error() != "EOF" {
					log.Fatalf("pool connection write error: %v", poolWriteError)
				}

				log.Printf("miner > pool: %v", string(minerBuffer))

				go func() {

					poolReader := bufio.NewReader(c.poolConnection)

					for {
						poolBuffer, poolReadError := poolReader.ReadBytes('\n')

						if poolReadError != nil {
							if poolReadError.Error() == "EOF" {
								continue
							}

							log.Printf("pool connection read error: %v", poolReadError)

							break
						}

						if len(poolBuffer) > 0 {
							_, minerConnectionWriteError := minerConnection.Write(poolBuffer)

							if minerConnectionWriteError != nil {
								log.Fatalf("miner connection write error: %v", minerConnectionWriteError)
							}

							log.Printf("miner < pool: %v", string(poolBuffer))
						}
					}
				}()
			}
		}(minerConnection)
	}
}

func (c *ConnectionsController) Update(message interface{}) {
	destinationMessage := message.(contractmanager.Dest)

	oldPoolAddr := c.poolAddr
	c.poolAddr = destinationMessage.NetUrl

	log.Printf("Switching to new pool address: %v", destinationMessage.NetUrl)

	<-time.After(2 * time.Minute)

	log.Printf("Switching back to old pool address: %v", oldPoolAddr)

	c.poolAddr = oldPoolAddr

}

func NewConnectionsController(poolAddr string) *ConnectionsController {
	return &ConnectionsController{poolAddr: poolAddr}
}
