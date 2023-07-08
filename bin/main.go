package main

import (
	"dtls_tunnel"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

var logger *zap.Logger

func init() {
	l, _ := zap.NewProduction()
	logger = l
	dtls_tunnel.SetLogger(l)
}

func server(commonConfig *dtls_tunnel.CommonConfig) {
	config, err := dtls_tunnel.ParseServerConfig(commonConfig)
	if err != nil {
		logger.Error(dtls_tunnel.FormatString("Failed to parse config: %s", err.Error()))
		os.Exit(1)
	}

	server := dtls_tunnel.NewServer(config)

	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChannel
		server.Shutdown()
	}()

	if err := server.Run(); err != nil {
		logger.Error(dtls_tunnel.FormatString("Failed to run server: %s", err.Error()))
		os.Exit(1)
	}
}

func client(commonConfig *dtls_tunnel.CommonConfig) {
	config, err := dtls_tunnel.ParseClientConfig(commonConfig)
	if err != nil {
		logger.Error(dtls_tunnel.FormatString("Failed to parse config: %s", err.Error()))
		os.Exit(1)
	}

	client := dtls_tunnel.NewClient(config)

	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChannel
		client.Shutdown()
	}()

	if err := client.Run(); err != nil {
		logger.Error(dtls_tunnel.FormatString("Failed to run client: %s", err.Error()))
		os.Exit(1)
	}
}

func main() {
	commonConfig, err := dtls_tunnel.ParseCommonConfig()
	if err != nil {
		logger.Error(dtls_tunnel.FormatString("Failed to parse common config: %s", err.Error()))
		os.Exit(1)
	}

	switch commonConfig.RunMethod {
	case "server":
		server(commonConfig)
		break

	case "client":
		client(commonConfig)
		break

	default:
		logger.Error("bad run type...")
		os.Exit(1)
	}
}
