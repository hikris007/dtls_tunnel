package dtls_tunnel

import (
	"crypto/tls"
	"crypto/x509"
	"net"
)

type CommonConfig struct {
	RunMethod string
	
	// 在 Read 数据的时候传入的缓冲区大小
	PackageBufferSize int

	// Client 在读取完后并非直接转发而是写入队列
	// 而这个就是设定队列的缓存数量
	PackageBufferCount int

	// Client: 监听UDP的地址
	// Server: DTLS Listener 的监听地址
	ListenAddress *net.UDPAddr

	// Client: 远程 DTLS Listener 的监听地址
	// Server: 收到数据后转发的UDP地址
	RemoteAddress *net.UDPAddr

	Cert      tls.Certificate
	RootCerts *x509.CertPool
}

type ServerConfig struct {
	CommonConfig
}

type ClientConfig struct {
	CommonConfig
}
