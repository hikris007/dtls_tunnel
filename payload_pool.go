package dtls_tunnel

import "sync"

type PayloadPooler interface {
	Get() (*Payload, error)
	Put(payload *Payload) error
}

type PayloadPool struct {
	payloadPool *sync.Pool
}

func NewPayloadPool(payloadCapacity int) PayloadPooler {
	pool := &PayloadPool{
		payloadPool: &sync.Pool{
			New: func() any {
				return NewPayload(payloadCapacity)
			},
		},
	}
	return pool
}

func (p *PayloadPool) Get() (*Payload, error) {
	payload := p.payloadPool.Get().(*Payload)
	return payload, nil
}

func (p *PayloadPool) Put(payload *Payload) error {
	p.payloadPool.Put(p)
	return nil
}
