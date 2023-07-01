package dtls_tunnel

import "net"

type Payload struct {
	payloadCapacity int
	container       []byte
	payloadLength   int
}

func NewPayload(payloadCapacity int) *Payload {
	payload := &Payload{
		payloadCapacity: payloadCapacity,
		container:       make([]byte, payloadCapacity),
		payloadLength:   0,
	}

	return payload

}

func (p *Payload) Empty() {
	p.payloadLength = 0
}

func (p *Payload) Data() []byte {
	return p.container[:p.payloadLength]
}

type Package struct {
	SrcAddress  *net.UDPAddr
	DestAddress *net.UDPAddr
	Payload     *Payload
}

func NewPackage(srcAddr, destAddr *net.UDPAddr, payload *Payload) *Package {
	return &Package{
		SrcAddress:  srcAddr,
		DestAddress: destAddr,
		Payload:     payload,
	}
}
