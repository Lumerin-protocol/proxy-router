package externalapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"gitlab.com/TitanInd/lumerin/cmd/externalapi/handlers"
	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus/msgdata"
)

// api holds dependencies for an external API.
type api struct {
	*gin.Engine

	Config                *msgdata.ConfigInfoRepo
	ContractManagerConfig *msgdata.ContractManagerConfigRepo
	Connection            *msgdata.ConnectionRepo
	Contract              *msgdata.ContractRepo
	Dest                  *msgdata.DestRepo
	Miner                 *msgdata.MinerRepo
	NodeOperator          *msgdata.NodeOperatorRepo
}

// New sets up a new API to access the given message bus data.
func New(ps *msgbus.PubSub) *api {
	api := &api{
		Engine:                gin.Default(),
		Config:                msgdata.NewConfigInfo(ps),
		ContractManagerConfig: msgdata.NewContractManagerConfig(ps),
		Connection:            msgdata.NewConnection(ps),
		Contract:              msgdata.NewContract(ps),
		Dest:                  msgdata.NewDest(ps),
		Miner:                 msgdata.NewMiner(ps),
		NodeOperator:          msgdata.NewNodeOperator(ps),
	}

	return api
}

// Run will start up the API on the given port, with a given logger.
func (api *api) Run(port string, l *log.Logger) {
	go api.Config.SubscribeToConfigInfoMsgBus()
	go api.ContractManagerConfig.SubscribeToContractManagerConfigMsgBus()
	go api.Connection.SubscribeToConnectionMsgBus()
	go api.Contract.SubscribeToContractMsgBus()
	go api.Dest.SubscribeToDestMsgBus()
	go api.Miner.SubscribeToMinerMsgBus()
	go api.NodeOperator.SubscribeToNodeOperatorMsgBus()

	time.Sleep(time.Millisecond * 2000)

	configRoutes := api.Group("/config")
	{
		configRoutes.GET("/", handlers.ConfigsGET(api.Config))
		configRoutes.GET("/:id", handlers.ConfigGET(api.Config))
		configRoutes.POST("/", handlers.ConfigPOST(api.Config))
		configRoutes.PUT("/:id", handlers.ConfigPUT(api.Config))
		configRoutes.DELETE("/:id", handlers.ConfigDELETE(api.Config))
	}

	contractManagerConfigRoutes := api.Group("/contractmanagerconfig")
	{
		contractManagerConfigRoutes.GET("/", handlers.ContractManagerConfigsGET(api.ContractManagerConfig))
		contractManagerConfigRoutes.GET("/:id", handlers.ContractManagerConfigGET(api.ContractManagerConfig))
		contractManagerConfigRoutes.POST("/", handlers.ContractManagerConfigPOST(api.ContractManagerConfig))
		contractManagerConfigRoutes.PUT("/:id", handlers.ContractManagerConfigPUT(api.ContractManagerConfig))
		contractManagerConfigRoutes.DELETE("/:id", handlers.ContractManagerConfigDELETE(api.ContractManagerConfig))
	}

	connectionRoutes := api.Group("/connection")
	{
		connectionRoutes.GET("/", handlers.ConnectionsGET(api.Connection))
		connectionRoutes.GET("/:id", handlers.ConnectionGET(api.Connection))
		connectionRoutes.POST("/", handlers.ConnectionPOST(api.Connection))
		connectionRoutes.PUT("/:id", handlers.ConnectionPUT(api.Connection))
		connectionRoutes.DELETE("/:id", handlers.ConnectionDELETE(api.Connection))
	}

	streamRoute := api.Group("/ws")
	{
		streamRoute.GET("/", handlers.ConnectionSTREAM(api.Connection))
	}

	contractRoutes := api.Group("/contract")
	{
		contractRoutes.GET("/", handlers.ContractsGET(api.Contract))
		contractRoutes.GET("/:id", handlers.ContractGET(api.Contract))
		contractRoutes.POST("/", handlers.ContractPOST(api.Contract))
		contractRoutes.PUT("/:id", handlers.ContractPUT(api.Contract))
		contractRoutes.DELETE("/:id", handlers.ContractDELETE(api.Contract))
	}

	destRoutes := api.Group("/dest")
	{
		destRoutes.GET("/", handlers.DestsGET(api.Dest))
		destRoutes.GET("/:id", handlers.DestGET(api.Dest))
		destRoutes.POST("/", handlers.DestPOST(api.Dest))
		destRoutes.PUT("/:id", handlers.DestPUT(api.Dest))
		destRoutes.DELETE("/:id", handlers.DestDELETE(api.Dest))
	}

	minerRoutes := api.Group("/miner")
	{
		minerRoutes.GET("/", handlers.MinersGET(api.Miner))
		minerRoutes.GET("/:id", handlers.MinerGET(api.Miner))
		minerRoutes.POST("/", handlers.MinerPOST(api.Miner))
		minerRoutes.PUT("/:id", handlers.MinerPUT(api.Miner))
		minerRoutes.DELETE("/:id", handlers.MinerDELETE(api.Miner))
	}

	nodeOperatorRoutes := api.Group("/nodeoperator")
	{
		nodeOperatorRoutes.GET("/", handlers.NodeOperatorsGET(api.NodeOperator))
		nodeOperatorRoutes.GET("/:id", handlers.NodeOperatorGET(api.NodeOperator))
		nodeOperatorRoutes.POST("/", handlers.NodeOperatorPOST(api.NodeOperator))
		nodeOperatorRoutes.PUT("/:id", handlers.NodeOperatorPUT(api.NodeOperator))
		nodeOperatorRoutes.DELETE("/:id", handlers.NodeOperatorDELETE(api.NodeOperator))
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           api,
		IdleTimeout:       20 * time.Second,
		WriteTimeout:      60 * time.Second,
		ReadHeaderTimeout: 20 * time.Second,
		MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
	}

	fmt.Printf("REST listening on port :%v\n", port)

	if err := server.ListenAndServe(); err != nil {
		l.Logf(log.LevelError, "serving REST API: %v", err)
	}
}
