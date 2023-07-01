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
	logger = zap.NewExample()
	dtls_tunnel.SetLogger(logger)
}

func main() {
	config, err := dtls_tunnel.ParseServerConfig()
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
