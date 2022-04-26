package integrationtestspass

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/protocol/stratumv1"
	"gitlab.com/TitanInd/lumerin/lumerinlib"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

type LocalConfig struct {
	BuyerNode           bool
	ListenIP            string
	ListenPort          string
	DefaultPoolAddr     string
	SchedulePassthrough bool
	LogFilePath         string
}

func TestConnMgr(t *testing.T) {

	var sleepTime time.Duration = 60 * time.Second

	//
	// Load configuration
	//
	configPath := "../../lumerinconfig-connmgr-msgbus-test.json"
	configs, err := LoadTestConfiguration(configPath)
	if err != nil {
		panic(fmt.Sprintf("Loading Config Failed: %s", err))
	}

	//
	// Setup logging
	//
	l := log.New()
	l.SetLevel(log.LevelTrace)
	// l.SetLevel(log.LevelDebug)

	// logFile, err := os.OpenFile("/tmp/logfile", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	// logFile, err := os.OpenFile("/dev/stdout", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	// logFile, err := os.OpenFile(configs.LogFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	// logFile, err := os.OpenFile(configs.LogFilePath, os.O_WRONLY, 0666)
	if err != nil {
		l.Logf(log.LevelFatal, "error opening log file: %v", err)
	}
	// defer logFile.Close()
	// l.SetFormat(log.FormatJSON).SetOutput(logFile)
	// l.SetFormat(log.FormatJSON).SetOutput(logFile)
	// l.SetFormat(log.FormatJSON).SetOutput(os.Stdout)
	l.SetOutput(os.Stdout)
	// l.SetOutput(os.Stdout)

	l.Logf(log.LevelTrace, "Starting Logfile")

	//
	// Setup MsgBus
	//
	ps := msgbus.New(10, l)
	// ps := msgbus.New(10, nil)

	mainContext := context.Background()

	defaultdst := "stratum+tcp://seanmcadam.switcher0:@mining.pool.titan.io:4242"
	seconddst := "stratum+tcp://seanmcadam.switcher1:@mining.pool.titan.io:4242/"

	//defaultdst := "stratum+tcp://seanmcadam.switcher0:@pooltesta1.sbx.lumerin.io:4242/"
	//seconddst := "stratum+tcp://seanmcadam.switcher1:@pooltesta1.sbx.lumerin.io:4242/"

	src := lumerinlib.NewNetAddr(lumerinlib.TCP, configs.ListenIP+":"+configs.ListenPort)
	dst := lumerinlib.NewNetAddr(lumerinlib.TCP, defaultdst)

	cs := contextlib.NewContextStruct(nil, ps, l, src, dst)

	mainContext = context.WithValue(mainContext, contextlib.ContextKey, cs)

	defaultDest := &msgbus.Dest{
		ID:     msgbus.DestID(msgbus.DEFAULT_DEST_ID),
		NetUrl: msgbus.DestNetUrl(defaultdst),
	}

	//
	// Publish Default Dest record
	//
	event, err := ps.PubWait(msgbus.DestMsg, msgbus.IDString(msgbus.DEFAULT_DEST_ID), defaultDest)
	if err != nil {
		panic(fmt.Sprintf("Adding Default Dest Failed: %s", err))
	}
	if event.Err != nil {
		panic(fmt.Sprintf("Adding Default Dest Failed: %s", event.Err))
	}

	//
	// Publish alternate pool destination
	//
	newTargetDest := msgbus.Dest{
		ID:     msgbus.DestID(msgbus.GetRandomIDString()),
		NetUrl: msgbus.DestNetUrl(seconddst),
	}
	ps.PubWait(msgbus.DestMsg, msgbus.IDString(newTargetDest.ID), newTargetDest)

	srcStrat, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", configs.ListenIP, configs.ListenPort))
	if err != nil {
		lumerinlib.PanicHere("")
	}

	stratum, err := stratumv1.NewListener(mainContext, srcStrat, defaultDest)
	if err != nil {
		panic(fmt.Sprintf("Stratum Protocol New() failed:%s", err))
	}

	stratum.RunOnce()
	// stratum.Run()

	//
	// Sleep while the system stablizes
	//

	time.Sleep(sleepTime)

	for {
		//
		// Point miners to new Dest
		//

		miners, _ := ps.MinerGetAllWait()
		for _, v := range miners {
			minerptr, _ := ps.MinerGetWait(msgbus.MinerID(v))
			miner := *minerptr
			if minerptr != nil {
				miner.Dest = newTargetDest.ID
				e := ps.MinerSetWait(miner)
				if e != nil {
					t.Errorf("MinerSetWait() error:%s on %s", e, miner.ID)
				}
			}
		}

		time.Sleep(sleepTime)

		fmt.Printf("\n***********\nPoint to Alternate DST\n************\n")
		//
		// Toggle back to the default dest
		//
		miners, _ = ps.MinerGetAllWait()
		for _, v := range miners {
			minerptr, _ := ps.MinerGetWait(msgbus.MinerID(v))
			if minerptr != nil {
				miner := *minerptr
				miner.Dest = defaultDest.ID
				e := ps.MinerSetWait(miner)
				if e != nil {
					t.Errorf("MinerSetWait() error:%s on %s", e, miner.ID)
				}
			}
		}

		time.Sleep(sleepTime)
		fmt.Printf("\n***********\nPoint to Primary DST\n************\n")
	}
}

//
//
//
func LoadTestConfiguration(filePath string) (configs LocalConfig, err error) {
	var data map[string]interface{}
	currDir, _ := os.Getwd()
	defer os.Chdir(currDir)

	if err != nil {
		panic(fmt.Errorf("error retrieving config file variable: %s", err))
	}
	file := filepath.Base(filePath)
	filePath = filepath.Dir(filePath)
	os.Chdir(filePath)

	configFile, err := os.Open(file)
	if err != nil {
		return configs, err
	}
	defer configFile.Close()
	byteValue, _ := ioutil.ReadAll(configFile)

	err = json.Unmarshal(byteValue, &data)

	configData := data["config"].(map[string]interface{})
	configs.BuyerNode = configData["buyerNode"].(bool)

	connConfigData := data["connection"].(map[string]interface{})
	configs.ListenIP = connConfigData["listenIP"].(string)
	configs.ListenPort = connConfigData["listenPort"].(string)
	configs.DefaultPoolAddr = connConfigData["defaultPoolAddr"].(string)

	schedConfigData := data["schedule"].(map[string]interface{})
	configs.SchedulePassthrough = schedConfigData["passthrough"].(bool)

	logConfigData := data["logging"].(map[string]interface{})
	configs.LogFilePath = logConfigData["filePath"].(string)

	return configs, err
}
