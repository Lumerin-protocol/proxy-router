/*
Stratum-proxy with external manage.
*/

package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"

	"github.com/joho/godotenv"
	"gitlab.com/TitanInd/hashrouter/connections"
	"gitlab.com/TitanInd/hashrouter/contractmanager"
	"gitlab.com/TitanInd/hashrouter/events"
	"gitlab.com/TitanInd/hashrouter/interfaces"
	"gitlab.com/TitanInd/hashrouter/mining"
)

/*
VERSION - proxy version.
*/
const VERSION = "0.01"

var (

	// Db of users credentials.
	// db Db
	// Stratum endpoint.
	stratumAddr = "0.0.0.0:9332"
	// API endpoint.
	webAddr = "0.0.0.0:8081"
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
	// Metrics proxy tag.
	tag = ""
	// HashrateContract Address
	hashrateContract string
	// Eth node Address
	ethNodeAddr string
	logToFile   bool
)

func init() {
	godotenv.Load(".env")
	flag.StringVar(&stratumAddr, "stratum.addr", "0.0.0.0:3333", "Address and port for stratum")
	flag.StringVar(&webAddr, "web.addr", "0.0.0.0:8080", "Address and port for web server and metrics")
	flag.StringVar(&poolAddr, "pool.addr", "mining.staging.pool.titan.io:4242", "Address and port for mining pool")
	flag.BoolVar(&syslog, "syslog", false, "On true adapt log to out in syslog, hide date and colors")
	// flag.StringVar(&tag, "metrics.tag", stratumAddr, "Prometheus metrics proxy tag")
	flag.StringVar(&hashrateContract, "contract.addr", os.Getenv("DEFAULT_CONTRACT_ADDRESS"), "Address of smart contract that node is servicing")
	flag.StringVar(&ethNodeAddr, "ethNode.addr", os.Getenv("DEFAULT_EHTHEREUM_NODE_ADDRESS"), "Address of Ethereum RPC node to connect to via websocket")
	flag.BoolVar(&logToFile, "file.log", true, "true or false - whether to output logs to a file named 'logs'")
}

/*
Main function.
*/
func main() {
	flag.Parse()

	if syslog {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}
	manageLogOutput := InitLogs()
	defer manageLogOutput()

	eventManager := events.NewEventManager()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go InitControllers(eventManager)
	go InitContractManager(eventManager)

	wg.Wait()
}

func InitControllers(eventManager interfaces.IEventManager) {
	miningController := &mining.MiningController{}
	connectionsController := connections.NewConnectionsController(os.Getenv("DEFAULT_POOL_ADDRESS"), miningController)

	miningController.SetAuth(os.Getenv("DEFAULT_POOL_USER"), os.Getenv("DEFAULT_POOL_PASSWORD"))
	eventManager.Attach(contractmanager.DestMsg, connectionsController)
	eventManager.Attach(contractmanager.DestMsg, miningController)

	go connectionsController.Run()

	http.Handle("/connections", connectionsController)

	go func() {
		log.Printf("Listening for api requests at %v", webAddr)
		if err := http.ListenAndServe(webAddr, nil); err != nil {
			log.Fatalf("Web address listening at %v has suffered a fatal error: %v", webAddr, err)
		}
	}()
}

func InitContractManager(eventManager interfaces.IEventManager) {

	log.Println("initalizing contract manager...")
	ctx := context.Background()

	sellerManager := &contractmanager.SellerContractManager{}
	sellerManager.SetLogger(log.Default())

	contractmanager.Run(&ctx, sellerManager, eventManager, hashrateContract, ethNodeAddr)
}

func InitLogs() func() {

	if logToFile {

		logfile := `logfile`
		// open file read/write | create if not exist | clear file at open if exists
		f, _ := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)

		// save existing stdout | MultiWriter writes to saved stdout and file
		out := os.Stdout
		mw := io.MultiWriter(out, f)

		// get pipe reader and writer | writes to pipe writer come out pipe reader
		r, w, _ := os.Pipe()

		// replace stdout,stderr with pipe writer | all writes to stdout, stderr will go through pipe instead (fmt.print, log)
		os.Stdout = w
		os.Stderr = w

		// writes with log.Print should also write to mw
		log.SetOutput(mw)

		//create channel to control exit | will block until all copies are finished
		exit := make(chan bool)

		go func() {
			// copy all reads from pipe to multiwriter, which writes to stdout and file
			_, _ = io.Copy(mw, r)
			// when r or w is closed copy will finish and true will be sent to channel
			exit <- true
		}()

		// function to be deferred in main until program exits
		return func() {
			// close writer then block on exit channel | this will let mw finish writing before the program exits
			_ = w.Close()
			<-exit
			// close file after all writes have finished
			_ = f.Close()
		}
	}

	return func() {}
}
