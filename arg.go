package dtls_tunnel

import (
	"crypto/x509"
	"flag"
	"github.com/pion/dtls/v2/examples/util"
	"net"
)

func ParseCommonConfig() (*CommonConfig, error) {
	config := &CommonConfig{}

	var listenAddressWithPort string
	var remoteAddressWithPort string

	var keyPath string
	var certPath string
	var rootCertPath string

	flag.IntVar(&config.PackageBufferSize, "pbs", 1500, "")
	flag.IntVar(&config.PackageBufferCount, "pbc", 1500, "")

	flag.StringVar(&listenAddressWithPort, "l", "0.0.0.0:10000", "")
	flag.StringVar(&remoteAddressWithPort, "r", "127.0.0.1:10000", "")

	flag.StringVar(&keyPath, "key", "", "")
	flag.StringVar(&certPath, "cert", "", "")
	flag.StringVar(&rootCertPath, "rc", "", "root cert")

	flag.Parse()

	address, err := net.ResolveUDPAddr("udp", listenAddressWithPort)
	if err != nil {
		return nil, MakeErrorWithErrMsg("Failed to parse listen address of client: %s", err.Error())
	}
	config.ListenAddress = address

	address, err = net.ResolveUDPAddr("udp", remoteAddressWithPort)
	if err != nil {
		return nil, MakeErrorWithErrMsg("Failed to parse listen address of server: %s", err.Error())
	}
	config.RemoteAddress = address

	cert, err := util.LoadKeyAndCertificate(keyPath, certPath)
	if err != nil {
		return nil, MakeErrorWithErrMsg("Failed to load key or cert: %s", err.Error())
	}
	config.Cert = cert

	rootCert, err := util.LoadCertificate(rootCertPath)
	if err != nil {
		return nil, MakeErrorWithErrMsg("Failed to load root cert: %s", err.Error())
	}

	rootCertParsed, err := x509.ParseCertificate(rootCert.Certificate[0])
	if err != nil {
		return nil, MakeErrorWithErrMsg("Failed to parse root cert: %s", err.Error())
	}

	rootCertPool := x509.NewCertPool()
	rootCertPool.AddCert(rootCertParsed)
	config.RootCerts = rootCertPool

	return config, nil
}

func ParseClientConfig() (*ClientConfig, error) {
	commonConfig, err := ParseCommonConfig()
	if err != nil {
		return nil, MakeErrorWithErrMsg("Failed to parse common config: %s", err.Error())
	}

	config := &ClientConfig{}
	config.PackageBufferSize = commonConfig.PackageBufferCount
	config.PackageBufferCount = commonConfig.PackageBufferCount

	config.ListenAddress = commonConfig.ListenAddress
	config.RemoteAddress = commonConfig.RemoteAddress

	config.Cert = commonConfig.Cert
	config.RootCerts = commonConfig.RootCerts

	return config, nil
}

func ParseServerConfig() (*ServerConfig, error) {
	commonConfig, err := ParseCommonConfig()
	if err != nil {
		return nil, MakeErrorWithErrMsg("Failed to parse common config: %s", err.Error())
	}

	config := &ServerConfig{}
	config.PackageBufferSize = commonConfig.PackageBufferCount
	config.PackageBufferCount = commonConfig.PackageBufferCount

	config.ListenAddress = commonConfig.ListenAddress
	config.RemoteAddress = commonConfig.RemoteAddress

	config.Cert = commonConfig.Cert
	config.RootCerts = commonConfig.RootCerts

	return config, nil
}
