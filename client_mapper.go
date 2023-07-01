package dtls_tunnel

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/pion/dtls/v2"
)

/*
 * client -> forward_client -> forward_server -> server : write
 * server -> forward_server -> forward_client -> client : read
 */

type ClientMapper struct {
	client     *Client       // Client 的指针
	srcAddress *net.UDPAddr  // 源地址
	tunnel     *dtls.Conn    // DTLS 连接
	readQueue  chan *Payload // 从 DTLS 连接返回的数据的队列
	writeQueue chan *Payload // 往 DTLS 连接写入的队列

	// 本地的 context 是独立的 基于创建时传入的父 context
	ctx context.Context

	// 本地 context 的关闭函数
	cancelFunc context.CancelFunc

	// 处理 readQueue 的队列和处理 writeQueue的队列的携程会用到
	wg *sync.WaitGroup
}

func NewClientMapper(client *Client, srcAddress *net.UDPAddr, parentCtx context.Context) *ClientMapper {
	ctx, cancel := context.WithCancel(parentCtx)

	clientMapper := &ClientMapper{
		client:     client,
		srcAddress: CloneUdpAddr(srcAddress),
		readQueue:  make(chan *Payload, client.config.PackageBufferCount),
		writeQueue: make(chan *Payload, client.config.PackageBufferCount),
		ctx:        ctx,
		cancelFunc: cancel,
		wg:         &sync.WaitGroup{},
	}

	return clientMapper
}

func (cm *ClientMapper) Run(wg *sync.WaitGroup) error {
	if err := cm.init(); err != nil {
		return MakeErrorWithErrMsg("Failed to run client mapper: %s", err.Error())
	}

	cm.runInLoop(wg)

	if err := cm.clean(); err != nil {
		return err
	}

	return nil
}

func (cm *ClientMapper) Stop() {
	cm.cancelFunc()
}

func (cm *ClientMapper) clean() error {
	// 等待转发携程关闭
	// Mark:是否必要
	cm.wg.Wait()

	// 取消初始化
	if err := cm.unInit(); err != nil {
		return MakeErrorWithErrMsg("Failed to close client mapper: %s", err.Error())
	}
	return nil
}

func (cm *ClientMapper) runInLoop(wg *sync.WaitGroup) {
	defer wg.Done()

	cm.wg.Add(3)

	go cm.handleWrite()
	go cm.handleRead()
	go cm.handleReadQueue()

	cm.wg.Wait()
}

func (cm *ClientMapper) Write(payload *Payload) {
	cm.writeQueue <- payload
}

func (cm *ClientMapper) handleWrite() {
	defer cm.wg.Done()

	var payload *Payload = nil
	var n int = 0
	var err error = nil

	for {
		select {
		// 等待本地的context关闭
		case <-cm.ctx.Done():
			return

		case payload = <-cm.writeQueue:
			n, err = cm.tunnel.Write(payload.Data())

			if err != nil {
				logger.Error(FormatString("Failed to write to tunnel: %s", err.Error()))
				cm.Stop()

				RecoveryPayload(payload, cm.client.payloadPool)
				return
			}

			if n != payload.payloadLength {
				logger.Error(FormatString("Write to tunnel with an error, len of written != payload's len"))
				cm.Stop()

				RecoveryPayload(payload, cm.client.payloadPool)
				return
			}

			RecoveryPayload(payload, cm.client.payloadPool)
		}
	}
}

func (cm *ClientMapper) handleRead() {
	defer cm.wg.Done()

	for {
		select {
		case <-cm.ctx.Done():
			return

		default:
			payload, err := cm.client.payloadPool.Get()
			if err != nil {
				logger.Warn(FormatString("Failed to get payload on pool: %s", err.Error()))
				continue
			}

			payload.payloadLength, err = cm.tunnel.Read(payload.container)
			if err != nil {
				logger.Error(FormatString("Failed to read from tunnel: %s", err.Error()))
				cm.Stop()

				RecoveryPayload(payload, cm.client.payloadPool)
				return
			}

			cm.readQueue <- payload
		}
	}
}

func (cm *ClientMapper) handleReadQueue() {
	defer cm.wg.Done()

	for {
		select {
		case <-cm.ctx.Done():
			return

		case payload := <-cm.readQueue:
			cm.client.HandleRead(NewPackage(CloneUdpAddr(cm.srcAddress), nil, payload))
		}
	}
}

func (cm *ClientMapper) init() error {

	if err := cm.initTunnel(); err != nil {
		return MakeErrorWithErrMsg("Failed to init: %s", err.Error())
	}

	return nil
}

func (cm *ClientMapper) unInit() error {
	if err := cm.closeTunnel(); err != nil {
		return MakeErrorWithErrMsg("Failed to un init client mapper: %s", err.Error())
	}

	return nil
}

func (cm *ClientMapper) initTunnel() error {
	ctx, cancel := context.WithTimeout(cm.ctx, time.Second*10)
	defer cancel()

	config := &dtls.Config{
		Certificates:         []tls.Certificate{cm.client.config.Cert},
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
		RootCAs:              cm.client.config.RootCerts,
	}

	tunnel, err := dtls.DialWithContext(ctx, "udp", cm.client.config.RemoteAddress, config)
	if err != nil {
		return MakeErrorWithErrMsg("Failed to dial remote server: %s", err.Error())
	}

	cm.tunnel = tunnel

	return nil
}

func (cm *ClientMapper) closeTunnel() error {
	if err := cm.tunnel.Close(); err != nil {
		return MakeErrorWithErrMsg("Failed to close tunnel: %s", err.Error())
	}

	return nil
}
