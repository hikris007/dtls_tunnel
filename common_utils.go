package dtls_tunnel

import (
	"errors"
	"fmt"
	"net"
)

func FormatString(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

func MakeErrorWithErrMsg(format string, args ...any) error {
	return errors.New(FormatString(format, args...))
}

func RecoveryPayload(payload *Payload, payloadPool PayloadPooler) {
	payload.Empty()
	if err := payloadPool.Put(payload); err != nil {
		logger.Warn(FormatString("Failed to put payload to pool: %s", err.Error()))
	}
}

func CopyUdpAddr(destAddr *net.UDPAddr, srcAddr *net.UDPAddr) error {
	if destAddr == nil || srcAddr == nil {
		return MakeErrorWithErrMsg("Failed to copy addresses: destAddr or srcAddr are nil")
	}

	destAddr.Port = srcAddr.Port
	destAddr.Zone = srcAddr.Zone
	copy(destAddr.IP, srcAddr.IP)

	return nil
}

func CloneUdpAddr(srcAddr *net.UDPAddr) *net.UDPAddr {
	if srcAddr == nil {
		return nil
	}

	ip := make(net.IP, len(srcAddr.IP))
	copy(ip, srcAddr.IP)

	addr := &net.UDPAddr{
		Port: srcAddr.Port,
		Zone: srcAddr.Zone,
		IP:   ip,
	}

	return addr
}
