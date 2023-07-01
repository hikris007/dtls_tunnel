package dtls_tunnel

import "go.uber.org/zap"

var logger *zap.Logger

func init() {

}

func SetLogger(l *zap.Logger) {
	logger = l
}
