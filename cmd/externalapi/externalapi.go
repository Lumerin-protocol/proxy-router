package externalapi

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	"gitlab.com/TitanInd/lumerin/cmd/externalapi/handlers"
	"gitlab.com/TitanInd/lumerin/cmd/log"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus"
	"gitlab.com/TitanInd/lumerin/cmd/msgbus/msgdata"
	"gitlab.com/TitanInd/lumerin/interfaces"

	runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	configv1 "github.com/lsheva/lumerin-sdk-go/proto/config/v1"
	contextlib "gitlab.com/TitanInd/lumerin/lumerinlib/context"
)

const ContentTypeHeader = "Content-Type"
const ContentTypeApplicationGRPC = "application/grpc"

var restApiMatcher = regexp.MustCompile(`^\/v\d+?\/`)

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
func (api *api) Run(ctx context.Context, port string) {
	go api.configRepo.SubscribeToConfigInfoMsgBus()
	go api.contractManagerConfigRepo.SubscribeToContractManagerConfigMsgBus()
	go api.connectionRepo.SubscribeToConnectionMsgBus()
	go api.contractRepo.SubscribeToContractMsgBus()
	go api.destRepo.SubscribeToDestMsgBus()
	go api.minerRepo.SubscribeToMinerMsgBus()
	go api.nodeOperatorRepo.SubscribeToNodeOperatorMsgBus()

	time.Sleep(time.Millisecond * 2000)

	api.registerLegacyHttpHandlers()

	restMux := runtime.NewServeMux()
	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	if err := api.registerHandlers(ctx, grpcServer, restMux); err != nil {
		contextlib.Logf(ctx, log.LevelError, "Cannot register handlers: %v", err)
	}

	grpcWebServer := grpcweb.WrapServer(grpcServer, grpcweb.WithWebsockets(true))
	httpsSrv := &http.Server{
		// These interfere with websocket streams, disable for now
		// ReadTimeout: 5 * time.Second,
		// WriteTimeout: 10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		Addr:              fmt.Sprintf("0.0.0.0:%s", port),
		Handler:           h2c.NewHandler(api.universalHandler(grpcWebServer, restMux), &http2.Server{}),
	}

	err := httpsSrv.ListenAndServe()
	if err != nil {
		contextlib.Logf(ctx, log.LevelError, "Cannot start server: %v", err)
	}
}

func (api *api) universalHandler(grpcSrv http.Handler, restSrv http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isGRPCRequest(r) || websocket.IsWebSocketUpgrade(r) {
			// grpc and grpc-web api
			grpcSrv.ServeHTTP(w, r)
		} else if restApiMatcher.Match([]byte(r.URL.Path)) {
			// new rest api
			restSrv.ServeHTTP(w, r)
		} else {
			// old rest api
			api.ServeHTTP(w, r)
		}
	})
}

func (api *api) registerHandlers(ctx context.Context, grpc grpc.ServiceRegistrar, rest *runtime.ServeMux) error {
	configv1Mux := handlers.NewConfigHandlers(api.configRepo)
	configv1.RegisterConfigsServiceServer(grpc, configv1Mux)
	return configv1.RegisterConfigsServiceHandlerServer(ctx, rest, configv1Mux)
}

func (api *api) registerLegacyHttpHandlers() {

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
}

func isGRPCRequest(r *http.Request) bool {
	return strings.Contains(r.Header.Get(ContentTypeHeader), ContentTypeApplicationGRPC)
}
