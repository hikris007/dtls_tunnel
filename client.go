package dtls_tunnel

import (
	"context"
	"net"
	"os"
	"sync"
	"time"
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

	c.wg.Add(3)
	go c.mapperGarbageCollector()
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

	for {
		select {
		case <-c.ctx.Done():
			return

		case <-time.After(READ_TIMEOUT):
			continue

		case pack = <-c.readQueue:
			if err := c.listener.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT)); err != nil {
				logger.Error(FormatString("Failed to set write deadline: %s", err.Error()))
				c.Shutdown()
				return
			}
			_, err = c.listener.WriteToUDP(
				pack.Payload.container[:pack.Payload.payloadLength],
				pack.SrcAddress,
			)

			if os.IsTimeout(err) {
				continue
			}

			if err != nil {
				logger.Warn(FormatString("Failed to write to %s: %s", pack.SrcAddress.String(), err.Error()))
				RecoveryPayload(pack.Payload, c.payloadPool)
				continue
			}

			RecoveryPayload(pack.Payload, c.payloadPool)
		}
	}

}

func (c *Client) mapperGarbageCollector() {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	handler := func(key string, mapper *ClientMapper) bool {
		if mapper.activeRecorder.IsTimeout(time.Minute * 30) {
			mapper.Stop()
			logger.Info(FormatString("Clean mapper: %s", key))
		}
		return true
	}

	for {
		select {
		case <-c.ctx.Done():
			return

		case <-ticker.C:
			c.mappers.Range(handler)
		}
	}
}

func (c *Client) handleMapperDestroy(srcAddress *net.UDPAddr) {
	if srcAddress == nil {
		return
	}

	srcAddressStr := srcAddress.String()

	if !c.mappers.Exist(srcAddressStr) {
		return
	}

	c.mappers.Delete(srcAddressStr)
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

			if err := c.listener.SetReadDeadline(time.Now().Add(READ_TIMEOUT)); err != nil {
				logger.Error(FormatString("Failed to set read deadline: %s", err.Error()))
				c.Shutdown()
				RecoveryPayload(payload, c.payloadPool)
				return
			}
			n, srcAddr, err := c.listener.ReadFromUDP(payload.container)

			if os.IsTimeout(err) {
				RecoveryPayload(payload, c.payloadPool)
				continue
			}

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
				logger.Info(FormatString("New mapper: %s", srcAddrStr))
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
