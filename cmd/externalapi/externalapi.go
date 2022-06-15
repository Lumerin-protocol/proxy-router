package externalapi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"gitlab.com/TitanInd/lumerin/cmd/externalapi/handlers"
	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus/msgdata"
	"gitlab.com/TitanInd/lumerin/interfaces"

	runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	configv1 "github.com/lsheva/lumerin-sdk-go/config/v1"
)

// api holds dependencies for an external API.
type api struct {
	*gin.Engine

	configRepo                *msgdata.ConfigInfoRepo
	contractManagerConfigRepo *msgdata.ContractManagerConfigRepo
	connectionRepo            *msgdata.ConnectionRepo
	contractRepo              *msgdata.ContractRepo
	destRepo                  *msgdata.DestRepo
	minerRepo                 *msgdata.MinerRepo
	nodeOperatorRepo          *msgdata.NodeOperatorRepo
}

// New sets up a new API to access the given message bus data.
func New(ps *msgbus.PubSub, connectionCollection interfaces.IConnectionController) *api {
	api := &api{
		Engine:                    gin.Default(),
		configRepo:                msgdata.NewConfigInfo(ps),
		contractManagerConfigRepo: msgdata.NewContractManagerConfig(ps),
		connectionRepo:            msgdata.NewConnection(ps),
		contractRepo:              msgdata.NewContract(ps),
		destRepo:                  msgdata.NewDest(ps),
		minerRepo:                 msgdata.NewMiner(ps),
		nodeOperatorRepo:          msgdata.NewNodeOperator(ps),
	}

	handlers.SetConnectionCollection(connectionCollection)

	return api
}

// Run will start up the API on the given port, with a given logger.
func (api *api) Run(ctx context.Context, port string, grpcAddress string, grpcWebPort string, l *log.Logger) {
	go api.configRepo.SubscribeToConfigInfoMsgBus()
	go api.contractManagerConfigRepo.SubscribeToContractManagerConfigMsgBus()
	go api.connectionRepo.SubscribeToConnectionMsgBus()
	go api.contractRepo.SubscribeToContractMsgBus()
	go api.destRepo.SubscribeToDestMsgBus()
	go api.minerRepo.SubscribeToMinerMsgBus()
	go api.nodeOperatorRepo.SubscribeToNodeOperatorMsgBus()

	time.Sleep(time.Millisecond * 2000)

	configRoutes := api.Group("/config")
	{
		configRoutes.GET("/", handlers.ConfigsGET(api.configRepo))
		configRoutes.GET("/:id", handlers.ConfigGET(api.configRepo))
		configRoutes.POST("/", handlers.ConfigPOST(api.configRepo))
		configRoutes.PUT("/:id", handlers.ConfigPUT(api.configRepo))
		configRoutes.DELETE("/:id", handlers.ConfigDELETE(api.configRepo))
	}

	contractManagerConfigRoutes := api.Group("/contractmanagerconfig")
	{
		contractManagerConfigRoutes.GET("/", handlers.ContractManagerConfigsGET(api.contractManagerConfigRepo))
		contractManagerConfigRoutes.GET("/:id", handlers.ContractManagerConfigGET(api.contractManagerConfigRepo))
		contractManagerConfigRoutes.POST("/", handlers.ContractManagerConfigPOST(api.contractManagerConfigRepo))
		contractManagerConfigRoutes.PUT("/:id", handlers.ContractManagerConfigPUT(api.contractManagerConfigRepo))
		contractManagerConfigRoutes.DELETE("/:id", handlers.ContractManagerConfigDELETE(api.contractManagerConfigRepo))
	}

	connectionRoutes := api.Group("/connections")
	{
		connectionRoutes.GET("/", handlers.ConnectionsGET(api.connectionRepo))
		connectionRoutes.GET("/:id", handlers.ConnectionGET(api.connectionRepo))
		connectionRoutes.POST("/", handlers.ConnectionPOST(api.connectionRepo))
		connectionRoutes.PUT("/:id", handlers.ConnectionPUT(api.connectionRepo))
		connectionRoutes.DELETE("/:id", handlers.ConnectionDELETE(api.connectionRepo))
	}

	streamRoute := api.Group("/ws")
	{
		streamRoute.GET("/", handlers.ConnectionSTREAM(api.connectionRepo))
	}

	contractRoutes := api.Group("/contract")
	{
		contractRoutes.GET("/", handlers.ContractsGET(api.contractRepo))
		contractRoutes.GET("/:id", handlers.ContractGET(api.contractRepo))
		contractRoutes.POST("/", handlers.ContractPOST(api.contractRepo))
		contractRoutes.PUT("/:id", handlers.ContractPUT(api.contractRepo))
		contractRoutes.DELETE("/:id", handlers.ContractDELETE(api.contractRepo))
	}

	destRoutes := api.Group("/dest")
	{
		destRoutes.GET("/", handlers.DestsGET(api.destRepo))
		destRoutes.GET("/:id", handlers.DestGET(api.destRepo))
		destRoutes.POST("/", handlers.DestPOST(api.destRepo))
		destRoutes.PUT("/:id", handlers.DestPUT(api.destRepo))
		destRoutes.DELETE("/:id", handlers.DestDELETE(api.destRepo))
	}

	minerRoutes := api.Group("/miner")
	{
		minerRoutes.GET("/", handlers.MinersGET(api.minerRepo))
		minerRoutes.GET("/:id", handlers.MinerGET(api.minerRepo))
		minerRoutes.POST("/", handlers.MinerPOST(api.minerRepo))
		minerRoutes.PUT("/:id", handlers.MinerPUT(api.minerRepo))
		minerRoutes.DELETE("/:id", handlers.MinerDELETE(api.minerRepo))
	}

	nodeOperatorRoutes := api.Group("/nodeoperator")
	{
		nodeOperatorRoutes.GET("/", handlers.NodeOperatorsGET(api.nodeOperatorRepo))
		nodeOperatorRoutes.GET("/:id", handlers.NodeOperatorGET(api.nodeOperatorRepo))
		nodeOperatorRoutes.POST("/", handlers.NodeOperatorPOST(api.nodeOperatorRepo))
		nodeOperatorRoutes.PUT("/:id", handlers.NodeOperatorPUT(api.nodeOperatorRepo))
		nodeOperatorRoutes.DELETE("/:id", handlers.NodeOperatorDELETE(api.nodeOperatorRepo))
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           api,
		IdleTimeout:       20 * time.Second,
		WriteTimeout:      60 * time.Second,
		ReadHeaderTimeout: 20 * time.Second,
		MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
	}

	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	httpMux := runtime.NewServeMux()

	api.registerHandlers(context.Background(), grpcServer, httpMux)

	// legacy http server
	go func() {
		l.Logf(log.LevelInfo, "REST listening on port: %v", port)
		if err := server.ListenAndServe(); err != nil {
			l.Logf(log.LevelError, "Error serving REST API: %v", err)
		}
	}()

	// grpc server
	go func() {
		l.Logf(log.LevelInfo, "gRPC server is listening on %v", grpcAddress)
		lis, err := net.Listen("tcp", grpcAddress)
		if err != nil {
			l.Logf(log.LevelError, "Error listening gRPC: %v", err)
		}
		if err := grpcServer.Serve(lis); err != nil {
			l.Logf(log.LevelError, "Error serving gRPC: %v", err)
		}
	}()

	// grpc-web http server mux
	go func() {
		l.Logf(log.LevelInfo, "gRPC-Web server mux is listening on %v", grpcWebPort)

		if err := http.ListenAndServe(fmt.Sprintf(":%s", grpcWebPort), httpMux); err != nil {
			l.Logf(log.LevelError, "Error serving gRPC: %v", err)
		}
	}()

}

func (api *api) registerHandlers(ctx context.Context, grpcServer grpc.ServiceRegistrar, httpMux *runtime.ServeMux) error {
	configv1Server := handlers.NewConfigHandlers(api.configRepo)
	// registers grpc api
	configv1.RegisterConfigsServiceServer(grpcServer, configv1Server)

	// registers rest api mux, according to grpc-web annotations
	return configv1.RegisterConfigsServiceHandlerServer(ctx, httpMux, configv1Server)
}
