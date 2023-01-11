package lile

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

// Run is a blocking cmd to run the gRPC and metrics server.
// You should listen to os signals and call Shutdown() if you
// want a graceful shutdown or want to handle other goroutines
func Run() error {
	if service.Registry != nil {
		service.Registry.Register(service)
	}

	// Start a metrics server in the background
	startPrometheusServer()

	// Create and then server a gRPC server
	err := ServeGRPC()
	if service.Registry != nil {
		service.Registry.DeRegister(service)
	}
	return err
}

// ServeGRPC creates and runs a blocking gRPC server
func ServeGRPC() error {
	var err error
	endpoint:=service.Config.Address()
	conn, err := net.Listen("tcp", endpoint)
	//如果有tls信息
	if service.ServerKey!="" && service.ServerCert!="" && service.ClientKey!="" && service.ClientCert!="" && service.CACert!="" && service.ServerName!=""{//有tls信息 grpc http同端口
		// gw server
		ctx := context.Background()
		gwmux := runtime.NewServeMux()
		//加载客户端证书
		cert, _ := tls.LoadX509KeyPair(service.ClientCert, service.ClientKey)
		certPool := x509.NewCertPool()
		ca, _ := ioutil.ReadFile(service.CACert)
		certPool.AppendCertsFromPEM(ca)

		creds := credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
			ServerName:   service.ServerName,
			RootCAs:      certPool,
		})
		dopt := grpc.WithTransportCredentials(creds)
		service.GRPCGatewayOption = append(service.GRPCGatewayOption,dopt)

		service.GRPCGatewayImpl(ctx,gwmux,endpoint,service.GRPCGatewayOption)

		// http服务
		mux := http.NewServeMux()
		var h http.Handler
		//cors
		//if service.Cors {
		//	h = handlers.CORS(service.CORSOption...)(headerHandler(gwmux))
		//}else{
		//	h = headerHandler(gwmux)
		//}
		h = headerHandler(wsproxy.WebsocketProxy(gwmux))
		mux.Handle("/", h)
		grpcServer:= createGrpcServer()
		srv := &http.Server{
			Addr:      endpoint,
			Handler:   grpcHandlerFunc(grpcServer, mux),
			TLSConfig: getTLSConfig(),
		}
		logrus.Infof("gRPC and https listen on: %s\n", endpoint)
		if err = srv.Serve(tls.NewListener(conn, srv.TLSConfig)); err != nil {
			logrus.Errorln("ListenAndServe: ", err)
		}
	}else{//没有tls
		ctx := context.Background()
		gwmux := runtime.NewServeMux()
		//dcreds, err := credentials.NewClientTLSFromFile(service.Cert, service.ServerName)
		//if err!=nil {
		//	return err
		//}
		//dopt := grpc.WithTransportCredentials(dcreds)
		//service.GRPCGatewayOption = append(service.GRPCGatewayOption,dopt)


		// Create a cmux.
		m := cmux.New(conn)
		grpcL := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
		//err =createGrpcServer().Serve(grpcL)
		go createGrpcServer().Serve(grpcL)
		httpL := m.Match(cmux.HTTP1Fast())
		var h http.Handler
		//cors
		//if service.Cors {
		//	h = handlers.CORS(service.CORSOption...)(headerHandler(gwmux))
		//}else{
		//	h = headerHandler(gwmux)
		//}
		h = headerHandler(wsproxy.WebsocketProxy(gwmux))
		httpS := &http.Server{
			Handler: h,
		}
		dopt := grpc.WithInsecure()
		service.GRPCGatewayOption = append(service.GRPCGatewayOption,dopt)
		service.GRPCGatewayImpl(ctx,gwmux,endpoint,service.GRPCGatewayOption)
		go httpS.Serve(httpL)
		// Start serving!
		m.Serve()
	}


	return err
}
type exampleHTTPHandler struct{}

func (h *exampleHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "example http response")
}
// Shutdown gracefully shuts down the gRPC and metrics servers
func Shutdown() {
	logrus.Infof("lile: Gracefully shutting down gRPC and Prometheus")

	if service.Registry != nil {
		service.Registry.DeRegister(service)
	}

	service.GRPCServer.GracefulStop()

	// 30 seconds is the default grace period in Kubernetes
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	if err := service.PrometheusServer.Shutdown(ctx); err != nil {
		logrus.Infof("Timeout during shutdown of metrics server. Error: %v", err)
	}
}

func createGrpcServer() *grpc.Server {
	service.GRPCOptions = append(service.GRPCOptions, grpc.UnaryInterceptor(
		grpc_middleware.ChainUnaryServer(service.UnaryInts...)))

	service.GRPCOptions = append(service.GRPCOptions, grpc.StreamInterceptor(
		grpc_middleware.ChainStreamServer(service.StreamInts...)))

	//如果有tls信息
	if service.ServerCert!="" && service.ServerKey!="" && service.ServerName!="" {
		cert, _ := tls.LoadX509KeyPair(service.ServerCert, service.ServerKey)
		certPool := x509.NewCertPool()
		ca, _ := ioutil.ReadFile(service.CACert)
		certPool.AppendCertsFromPEM(ca)

		creds := credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    certPool,
		})
		AddGRPCOption(grpc.Creds(creds))
	}

	service.GRPCServer = grpc.NewServer(
		service.GRPCOptions...,
	)

	service.GRPCImplementation(service.GRPCServer)

	grpc_prometheus.EnableHandlingTimeHistogram(
		func(opt *prometheus.HistogramOpts) {
			opt.Buckets = prometheus.ExponentialBuckets(0.005, 1.4, 20)
		},
	)

	grpc_prometheus.Register(service.GRPCServer)
	return service.GRPCServer
}

func startPrometheusServer() {
	service.PrometheusServer = &http.Server{Addr: service.PrometheusConfig.Address()}

	http.Handle("/metrics", promhttp.Handler())
	logrus.Infof("Prometheus metrics at http://%s/metrics", service.PrometheusConfig.Address())

	go func() {
		if err := service.PrometheusServer.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			logrus.Errorf("Prometheus http server: ListenAndServe() error: %s", err)
		}
	}()
}

func getTLSConfig() *tls.Config {
	cert, _ := ioutil.ReadFile(service.ServerCert)
	key, _ := ioutil.ReadFile(service.ServerKey)
	var KeyPair *tls.Certificate
	pair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		grpclog.Fatalf("TLS KeyPair err: %v\n", err)
	}
	KeyPair = &pair
	return &tls.Config{
		Certificates: []tls.Certificate{*KeyPair},
		NextProtos:   []string{http2.NextProtoTLS}, // HTTP2 TLS支持
	}
}

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Copied from cockroachdb.
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	if otherHandler == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			grpcServer.ServeHTTP(w, r)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}

func headerHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range service.GRPCGatewayHeader {
			w.Header().Set(k,v)
		}
		//跨域非简单方法 二次验证返回 200
		if r.Method == "OPTIONS" && service.Cors {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}