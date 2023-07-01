package dtls_tunnel

import (
	"context"
	"crypto/tls"
	"github.com/pion/dtls/v2"
	"net"
	"sync"
	"time"
)

type Server struct {
	config     *ServerConfig
	listener   net.Listener
	mappersWg  *sync.WaitGroup
	wg         *sync.WaitGroup
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewServer(config *ServerConfig) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config:     config,
		mappersWg:  &sync.WaitGroup{},
		wg:         &sync.WaitGroup{},
		ctx:        ctx,
		cancelFunc: cancel,
	}
	return server
}

func (s *Server) initListener() error {
	config := &dtls.Config{
		Certificates:         []tls.Certificate{s.config.Cert},
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
		ClientAuth:           dtls.RequireAndVerifyClientCert,
		ClientCAs:            s.config.RootCerts,
		ConnectContextMaker: func() (context.Context, func()) {
			// TODO:等待时间动态设置
			return context.WithTimeout(s.ctx, 30*time.Second)
		},
	}

	listener, err := dtls.Listen(
		"udp",
		s.config.ListenAddress,
		config,
	)
	if err != nil {
		return MakeErrorWithErrMsg("Failed to init listener: %s", err.Error())
	}

	s.listener = listener

	return nil
}

func (s *Server) closeListener() error {
	if err := s.listener.Close(); err != nil {
		return MakeErrorWithErrMsg("Failed to close listener: %s", err.Error())
	}

	return nil
}

func (s *Server) init() error {
	if err := s.initListener(); err != nil {
		return MakeErrorWithErrMsg("Failed to init server: %s", err.Error())
	}

	return nil
}

func (s *Server) unInit() error {
	if err := s.closeListener(); err != nil {
		return MakeErrorWithErrMsg("Failed to clean listener: %s", err.Error())
	}

	return nil
}

func (s *Server) handleConnection() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return

		default:
			conn, err := s.listener.Accept()
			if err != nil {
				logger.Error(FormatString("Failed to accept connection: %s", err.Error()))
				continue
			}

			mapper := NewServerMapper(
				s,
				conn.(*dtls.Conn),
				s.ctx,
			)

			go func() {
				s.mappersWg.Add(1)
				if err := mapper.Run(s.mappersWg); err != nil {
					logger.Error(FormatString("Failed to run mapper: %s", err.Error()))
				}
			}()
		}
	}
}

func (s *Server) clean() error {
	s.mappersWg.Wait()
	s.wg.Wait()

	if err := s.unInit(); err != nil {
		return MakeErrorWithErrMsg("Failed to un init server: %s", err.Error())
	}

	return nil
}

func (s *Server) Run() error {
	if err := s.init(); err != nil {
		return MakeErrorWithErrMsg("Failed to run mapper", err.Error())
	}

	s.wg.Add(1)

	go s.handleConnection()

	logger.Info(FormatString("The server is running on %s", s.config.ListenAddress.String()))

	s.wg.Wait()
	s.mappersWg.Wait()

	if err := s.clean(); err != nil {
		return err
	}

	logger.Info(FormatString("The client is shutdown"))

	return nil
}

func (s *Server) Shutdown() {
	s.cancelFunc()
}
