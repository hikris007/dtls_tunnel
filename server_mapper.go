package dtls_tunnel

import (
	"context"
	"github.com/pion/dtls/v2"
	"net"
	"sync"
)

type ServerMapper struct {
	server         *Server
	srcConnection  *dtls.Conn
	destConnection *net.UDPConn
	ctx            context.Context
	cancelFunc     context.CancelFunc
	wg             *sync.WaitGroup // 转发携程的同步等待组
}

func NewServerMapper(server *Server, src *dtls.Conn, parentCtx context.Context) *ServerMapper {
	ctx, cancel := context.WithCancel(parentCtx)

	serverMapper := &ServerMapper{
		server:        server,
		srcConnection: src,
		ctx:           ctx,
		cancelFunc:    cancel,
		wg:            &sync.WaitGroup{},
	}

	return serverMapper
}

func (sm *ServerMapper) Run(wg *sync.WaitGroup) error {
	if err := sm.init(); err != nil {
		return MakeErrorWithErrMsg("Failed to run server mapper: %s", err.Error())
	}

	sm.runInLoop(wg)

	if err := sm.clean(); err != nil {
		return err
	}

	return nil
}

func (sm *ServerMapper) Stop() {
	sm.cancelFunc()
}

func (sm *ServerMapper) clean() error {

	sm.wg.Wait()

	if err := sm.unInit(); err != nil {
		return MakeErrorWithErrMsg("Failed to stop server mapper: %s", err.Error())
	}

	return nil
}

func (sm *ServerMapper) init() error {
	if err := sm.initDestConnection(); err != nil {
		return MakeErrorWithErrMsg("Failed to init server mapper: %s", err.Error())
	}

	return nil
}

func (sm *ServerMapper) unInit() error {
	if err := sm.closeSrcConnection(); err != nil {
		return MakeErrorWithErrMsg("Failed to un init server mapper: %s", err.Error())
	}

	if err := sm.closeDestConnection(); err != nil {
		return MakeErrorWithErrMsg("Failed to un init server mapper: %s", err.Error())
	}

	return nil
}

func (sm *ServerMapper) initDestConnection() error {
	destConnection, err := net.DialUDP(
		"udp",
		nil,
		sm.server.config.RemoteAddress,
	)

	if err != nil {
		return MakeErrorWithErrMsg("Failed to init dest connection: %s", err.Error())
	}

	sm.destConnection = destConnection

	return nil
}

func (sm *ServerMapper) closeDestConnection() error {

	if err := sm.destConnection.Close(); err != nil {
		return MakeErrorWithErrMsg("Failed to close dest connection: %s", err.Error())
	}

	return nil
}

func (sm *ServerMapper) closeSrcConnection() error {
	if err := sm.srcConnection.Close(); err != nil {
		return MakeErrorWithErrMsg("Failed to close src connection: %s", err.Error())
	}
	return nil
}

func (sm *ServerMapper) runInLoop(wg *sync.WaitGroup) {
	defer wg.Done()

	sm.wg.Add(2)

	go sm.handleWrite()
	go sm.handleRead()

	sm.wg.Wait()
}

func (sm *ServerMapper) handleRead() {
	defer sm.wg.Done()

	var buffer []byte = make([]byte, sm.server.config.PackageBufferSize)
	var n int = 0
	var err error = nil

	for {
		select {
		case <-sm.ctx.Done():
			return

		default:
			n, err = sm.destConnection.Read(buffer)
			if err != nil {
				logger.Error(FormatString("Failed to read from dest conn: %s", err.Error()))
				sm.Stop()
				return
			}

			n, err = sm.srcConnection.Write(buffer[:n])
			if err != nil {
				logger.Error(FormatString("Failed to write to src conn: %s", err.Error()))
				sm.Stop()
				return
			}
		}
	}
}

func (sm *ServerMapper) handleWrite() {
	defer sm.wg.Done()

	var buffer []byte = make([]byte, sm.server.config.PackageBufferSize)
	var n int = 0
	var err error = nil

	for {
		select {
		case <-sm.ctx.Done():
			return

		default:
			n, err = sm.srcConnection.Read(buffer)
			if err != nil {
				logger.Error(FormatString("Failed to read from src conn: %s", err.Error()))
				sm.Stop()
				return
			}

			n, err = sm.destConnection.Write(buffer[:n])
			if err != nil {
				logger.Error(FormatString("Failed to write to dest conn: %s", err.Error()))
				sm.Stop()
				return
			}
		}
	}
}
