/*
Stratum-proxy with external manage.
*/

package main

import (
	"context"
	"flag"
	"log"
	"regexp"
	"sync"

	"gitlab.com/TitanInd/hashrouter/connections"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/events"
	"gitlab.com/TitanInd/hashrouter/interfaces"
)

/*
VERSION - proxy version.
*/
const VERSION = "0.01"

var (

	// Db of users credentials.
	// db Db
	// Stratum endpoint.
	stratumAddr = "127.0.0.1:9332"
	// API endpoint.
	webAddr = "127.0.0.1:8081"
	// Pool target
	poolAddr = ""
	// Out to syslog.
	syslog = false
	// GitCommit - Git commit for build
	GitCommit string
	// Compiled regexp for hexademical checks.
	rHexStr = regexp.MustCompile(`^[\da-fA-F]+$`)
	// Extensions that supported by the proxy.
	sExtensions = []string{
		// "subscribe-extranonce",
		"version-rolling",
	}
	// SQLite db path.
	dbPath = "proxy.db"
	// Metrics proxy tag.
	tag = ""
	// HashrateContract Address
	hashrateContract string
	// Eth node Address
	ethNodeAddr string
	// minerReader *bufio.Reader
	// poolReader  *bufio.Reader
)

func init() {
	// godotenv.Load(".env")
	flag.StringVar(&stratumAddr, "stratum.addr", "0.0.0.0:3333", "Address and port for stratum")
	flag.StringVar(&webAddr, "web.addr", "127.0.0.1:8082", "Address and port for web server and metrics")
	flag.StringVar(&poolAddr, "pool.addr", "mining.staging.pool.titan.io:4242", "Address and port for mining pool")
	flag.BoolVar(&syslog, "syslog", false, "On true adapt log to out in syslog, hide date and colors")
	flag.StringVar(&dbPath, "db.path", "proxy.db", "Filepath for SQLite database")
	// flag.StringVar(&tag, "metrics.tag", stratumAddr, "Prometheus metrics proxy tag")
	flag.StringVar(&hashrateContract, "contract.addr", "", "Address of smart contract that node is servicing")
	flag.StringVar(&ethNodeAddr, "ethNode.addr", "", "Address of Ethereum RPC node to connect to via websocket")

	// fmt.Println("listening on  socket...", "")
}

/*
Main function.
*/
func main() {

	flag.Parse()

	if syslog {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	eventManager := events.NewEventManager()

	wg := sync.WaitGroup{}
	wg.Add(1)

	InitControllers(eventManager)
	InitContractManager(eventManager)

	wg.Wait()
}

func InitControllers(eventManager interfaces.IEventManager) {
	connectionsController := connections.NewConnectionsController("mining.staging.pool.titan.io:4242")

	eventManager.Attach(contractmanager.DestMsg, connectionsController)

	connectionsController.Run()
}

func InitContractManager(eventManager interfaces.IEventManager) {

	log.Println("initalizing contract manager...")
	ctx := context.Background()

	sellerManager := &contractmanager.SellerContractManager{}
	sellerManager.SetLogger(log.Default())

	contractmanager.Run(&ctx, sellerManager, eventManager, hashrateContract, ethNodeAddr)
}
