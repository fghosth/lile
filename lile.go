// Package lile provides helper methods to quickly create RPC based services
// that have metrics, tracing and pub/sub support
package lile

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"github.com/gorilla/handlers"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/lileio/fromenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

var (
	service = NewService("lile")
)

// RegisterImplementation allows you to register your gRPC server
type RegisterImplementation func(s *grpc.Server)

type RegisterGrpcGatewayImpl func(ctx context.Context,mux *runtime.ServeMux,endpoint string,opt []grpc.DialOption)


// ServerConfig is a generic server configuration
type ServerConfig struct {
	Port int
	Host string
}

// Address Gets a logical addr for a ServerConfig
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Service is a gRPC based server with extra features
type Service struct {
	ID   string
	Name string

	//tls
	ServerCert string //server文件路径
	ServerKey string //server路文件径
	ClientKey string //文件路径
	ClientCert string //文件路径
	CACert string //文件路径
	ServerName string //tls 对应name

	//cors
	Cors bool //是否跨域true 支持跨域，false: 不支持跨域
	CORSOption []handlers.CORSOption

	// Interceptors
	UnaryInts  []grpc.UnaryServerInterceptor
	StreamInts []grpc.StreamServerInterceptor

	// The RPC server implementation
	GRPCImplementation RegisterImplementation
	GRPCOptions        []grpc.ServerOption

	//grpc-gateway implementation
	GRPCGatewayOption []grpc.DialOption
	GRPCGatewayImpl RegisterGrpcGatewayImpl
	GRPCGatewayHeader map[string]string

	// gRPC and Prometheus endpoints
	Config           ServerConfig
	PrometheusConfig ServerConfig

	// Registry allows Lile to work with external registeries like
	// consul, zookeeper or similar
	Registry Registry

	// Private utils, exposed so they can be useful if needed
	ServiceListener  net.Listener
	GRPCServer       *grpc.Server
	PrometheusServer *http.Server
}

// NewService creates a new service with a given name
func NewService(n string) *Service {
	return &Service{
		ID:                 generateID(n),
		Name:               n,
		Config:             ServerConfig{Host: "0.0.0.0", Port: 8000},
		PrometheusConfig:   ServerConfig{Host: "0.0.0.0", Port: 9000},
		GRPCImplementation: func(s *grpc.Server) {},
		UnaryInts: []grpc.UnaryServerInterceptor{
			grpc_prometheus.UnaryServerInterceptor,
			grpc_recovery.UnaryServerInterceptor(),
		},
		StreamInts: []grpc.StreamServerInterceptor{
			grpc_prometheus.StreamServerInterceptor,
			grpc_recovery.StreamServerInterceptor(),
		},
	}
}

// GlobalService returns the global service
func GlobalService() *Service {
	return service
}

//修改host,port
func SetAddress(host string,port int){
	service.Config.Host = host
	service.Config.Port = port
}

// Name sets the name for the service
func Name(n string) {
	service.ID = generateID(n)
	service.Name = n
	AddUnaryInterceptor(otgrpc.OpenTracingServerInterceptor(
		fromenv.Tracer(n)))
}

// Server attaches the gRPC implementation to the service
func Server(r func(s *grpc.Server)) {
	service.GRPCImplementation = r
}

//grpc gateway server
func GrpcGatewayServer(r func(ctx context.Context,mux *runtime.ServeMux,endpoint string,dopts []grpc.DialOption)){
	service.GRPCGatewayImpl =r
}
//enable cors
func EnableCORS(opt ...handlers.CORSOption){
	service.Cors = true
	service.CORSOption = opt
}
// Set TLS
func SetTLS(serverCert,serverKey,clientKey,clientCert,caCert,serverName string){
	service.ServerCert= serverCert
	service.ServerKey = serverKey
	service.ClientKey= clientKey
	service.ClientCert= clientCert
	service.CACert = caCert
	service.ServerName = serverName
}
// add GRPCGatewayHeader
func AddGRPCGatewayHeader(header map[string]string){
	service.GRPCGatewayHeader = header
}
// add grpcGatewayOption
func AddGRPCGatewayOption(opt grpc.DialOption){
	service.GRPCGatewayOption = append(service.GRPCGatewayOption,opt)
}
// add grpcOption
func AddGRPCOption(opt grpc.ServerOption){
	service.GRPCOptions = append(service.GRPCOptions,opt)
}
// AddUnaryInterceptor adds a unary interceptor to the RPC server
func AddUnaryInterceptor(unint grpc.UnaryServerInterceptor) {
	service.UnaryInts = append(service.UnaryInts, unint)
}

// AddStreamInterceptor adds a stream interceptor to the RPC server
func AddStreamInterceptor(sint grpc.StreamServerInterceptor) {
	service.StreamInts = append(service.StreamInts, sint)
}

// URLForService returns a service URL via a registry or a simple DNS name
// if not available via the registry
func URLForService(name string) string {
	if service.Registry != nil {
		url, err := service.Registry.Get(name)
		if err != nil {
			fmt.Printf("lile: error contacting registry for service %s. err: %s \n", name, err.Error())
		}
		return url
	}

	return fmt.Sprintf("%s%s", name, ":80")

}

// ContextClientInterceptor passes around headers for tracing and linkerd
func ContextClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, resp interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		pairs := make([]string, 0)

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			for key, values := range md {
				if strings.HasPrefix(strings.ToLower(key), "x-") {
					for _, value := range values {
						pairs = append(pairs, key, value)
					}
				}
			}
		}

		ctx = metadata.AppendToOutgoingContext(ctx, pairs...)
		return invoker(ctx, method, req, resp, cc, opts...)
	}
}
