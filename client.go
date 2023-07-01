package dtls_tunnel

import (
	"context"
	"net"
	"sync"
)

type Clienter interface {
	Run() error      // block
	Shutdown() error // block
}

type Client struct {
	config   *ClientConfig
	listener *net.UDPConn
	mappers  Mappers

	payloadPool PayloadPooler
	readQueue   chan *Package

	cancelFunc context.CancelFunc
	ctx        context.Context
	wg         *sync.WaitGroup
	mappersWg  *sync.WaitGroup
}

func NewClient(config *ClientConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		config:      config,
		mappers:     NewMappers(),
		readQueue:   make(chan *Package, config.PackageBufferCount),
		payloadPool: NewPayloadPool(config.PackageBufferSize),
		cancelFunc:  cancel,
		ctx:         ctx,
		wg:          &sync.WaitGroup{},
		mappersWg:   &sync.WaitGroup{},
	}

	return client
}

func (c *Client) Run() error {
	if err := c.init(); err != nil {
		return MakeErrorWithErrMsg("Failed to run client: %s", err.Error())
	}

	logger.Info(FormatString("The client is running on %s", c.config.ListenAddress.String()))

	c.wg.Add(2)
	go c.writeWorker()
	go c.readWorker()

	logger.Info(FormatString("The client is started"))

	c.mappersWg.Wait()
	c.wg.Wait()

	logger.Info(FormatString("The client is shutdown"))

	return nil
}

func (c *Client) Shutdown() {
	c.cancelFunc()
}

func (c *Client) clean() error {
	c.mappersWg.Wait()
	c.wg.Wait()

	if err := c.unInit(); err != nil {
		return MakeErrorWithErrMsg("Failed to shutdown client: %s", err.Error())
	}

	return nil
}

func (c *Client) init() error {
	if err := c.InitListener(); err != nil {
		return MakeErrorWithErrMsg("Failed to init Client: %s", err.Error())
	}

	return nil
}

func (c *Client) unInit() error {
	if err := c.closeListener(); err != nil {
		return MakeErrorWithErrMsg("Failed to unInit client: %s", err.Error())
	}
	return nil
}

func (c *Client) InitListener() error {
	listener, err := net.ListenUDP(
		"udp",
		c.config.ListenAddress,
	)

	if err != nil {
		return MakeErrorWithErrMsg("Failed to init listener: %s", err.Error())
	}

	c.listener = listener

	return nil
}

func (c *Client) closeListener() error {
	err := c.listener.Close()
	if err != nil {
		return MakeErrorWithErrMsg("Failed to uninit listener: %s", err.Error())
	}
	return nil
}

func (c *Client) HandleRead(pack *Package) {
	c.readQueue <- pack
}

func (c *Client) readWorker() {
	defer c.wg.Done()

	var pack *Package = nil
	var err error = nil
	//var n int = 0
	for {
		select {
		case <-c.ctx.Done():

		case pack = <-c.readQueue:
			_, err = c.listener.WriteToUDP(
				pack.Payload.container[:pack.Payload.payloadLength],
				pack.SrcAddress,
			)
			if err != nil {
				logger.Warn(FormatString("Failed to write to %s: %s", pack.SrcAddress.String(), err.Error()))

				if err := c.payloadPool.Put(pack.Payload); err != nil {
					logger.Warn(FormatString("Failed to put payload to pool: %s", err.Error()))
				}

				continue
			}

			if err := c.payloadPool.Put(pack.Payload); err != nil {
				logger.Warn(FormatString("Failed to put payload to pool: %s", err.Error()))
			}
		}
	}

}

func (c *Client) writeWorker() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return

		default:
			payload, err := c.payloadPool.Get()
			if err != nil {
				logger.Warn(FormatString("Failed to get payload on pool: %s", payload))
				continue
			}

			n, srcAddr, err := c.listener.ReadFromUDP(payload.container)
			if err != nil {
				logger.Warn(FormatString("Failed to read on listener: %s", err.Error()))

				RecoveryPayload(payload, c.payloadPool)
				continue
			}

			payload.payloadLength = n
			srcAddrStr := srcAddr.String()

			isMapperExist := c.mappers.Exist(srcAddrStr)
			if isMapperExist {
				mapper := c.mappers.Get(srcAddrStr)
				mapper.Write(payload)
			} else {
				mapper := NewClientMapper(
					c,
					srcAddr,
					c.ctx,
				)

				c.mappers.Set(srcAddrStr, mapper)

				c.mappersWg.Add(1)
				go mapper.Run(c.mappersWg)

				mapper.Write(payload)
			}
		}
	}
}
