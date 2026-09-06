package tcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"time"

	core_command "github.com/moond0wner/chat/internal/core/command"
	core_config "github.com/moond0wner/chat/internal/core/config"
	core_logger "github.com/moond0wner/chat/internal/core/logger"
	client_service "github.com/moond0wner/chat/internal/features/services/client"
	room_service "github.com/moond0wner/chat/internal/features/services/room"
	"go.uber.org/zap"
)

type Server struct {
	roomService   *room_service.RoomService
	clientService *client_service.ClientService
	Manager       *core_command.Manager
	log           *core_logger.Logger
	config        *core_config.Config
	mtx           sync.RWMutex
	wg            sync.WaitGroup
	listener      net.Listener
}

func NewServer(
	logger *core_logger.Logger,
	rs *room_service.RoomService,
	cs *client_service.ClientService,
	cfg *core_config.Config,
	mng *core_command.Manager,
) *Server {
	srv := &Server{
		roomService:   rs,
		clientService: cs,
		Manager:       mng,
		config:        cfg,
		log:           logger,
	}
	return srv
}

func (s *Server) Start(ctx context.Context) error {
	var err error
	s.listener, err = net.Listen("tcp", s.config.ServerPort)
	if err != nil {
		return fmt.Errorf("error listening: %w", err)
	}
	s.log.Info("TCP server started", zap.String("port", s.config.ServerPort))

	errCh := make(chan error, 1)
	done := make(chan struct{})

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer close(done)

		for {
			select {
			case <-ctx.Done():
				s.log.Debug("Stopping accepting connections")
				return
			default:
			}

			conn, err := s.listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
				s.log.Error("Error accepting connection", zap.Error(err))
				select {
				case errCh <- fmt.Errorf("accept error: %w", err):
				default:
					s.log.Warn("Error channel full, dropping error")
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}
			go s.RegisterInServer(ctx, conn)
		}
	}()

	select {
	case <-ctx.Done():
		s.log.Warn("Shutting down TCP server...")
		return s.gracefulShutdown()
	case err = <-errCh:
		if err != nil {
			s.log.Error("Server error", zap.Error(err))
			if shutdownErr := s.gracefulShutdown(); shutdownErr != nil {
				return fmt.Errorf("server error: %w, shutdown error: %v", err, shutdownErr)
			}
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	case <-done:
		return nil
	}
}

func (s *Server) gracefulShutdown() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown TCP server: %w", err)
	}

	s.wg.Wait()

	s.log.Warn("TCP server stopped")
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.listener != nil {
		s.log.Debug("Closed listener")
		if err := s.listener.Close(); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				s.log.Warn("Error close listener", zap.Error(err))
			}
		}
	}

	clients := s.clientService.GetAllClients()
	s.log.Debug("Send message clients", zap.Int("count", s.clientService.Count()))
	for _, client := range clients {
		if client.Conn != nil {
			fmt.Fprintf(client.Conn, "Сервер останавливается...\n")
		}
	}

	for _, client := range clients {
		if client.Conn != nil {
			client.Conn.Close()
			s.log.Debug("Client connection closed", zap.Int("client", client.ID))
		}
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.log.Info("All goroutines stopped")
		return nil
	case <-ctx.Done():
		s.log.Warn("Server shutdown timeout, forced termination")
		s.mtx.Lock()
		for _, client := range clients {
			if client.Conn != nil {
				client.Conn.Close()
			}
		}
		s.mtx.Unlock()
		return ctx.Err()
	}
}
