package tcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	core_command "tcp_srv/internal/core/command"
	core_domain "tcp_srv/internal/core/domain"
	core_logger "tcp_srv/internal/core/logger"
	"tcp_srv/internal/features/services"
	"time"

	"go.uber.org/zap"
)

type Server struct {
	roomService   *services.RoomService
	clientService *services.ClientService
	Manager       *core_command.Manager
	log           *core_logger.Logger
	mtx           sync.RWMutex
	wg            sync.WaitGroup
	listener      net.Listener
}

func NewServer(logger *core_logger.Logger, rs *services.RoomService, cs *services.ClientService) *Server {
	srv := &Server{
		roomService:   rs,
		clientService: cs,
		Manager:       core_command.NewManager(),
		log:           logger,
	}
	srv.registerCommands()
	return srv
}

func (s *Server) registerCommands() {
	s.Manager.Register([]core_command.CommandHandler{
		{
			Name:        "/join",
			Description: "Подключение к комнате",
			Usage:       "/join <room_name>",
			MinArgs:     1,
			Handler: func(client *core_domain.Client, args []string) error {
				return s.roomService.JoinRoom(client, args[0])
			},
		},
		// {
		// 	Name:        "/info",
		// 	Description: "Информация о сервере: комнаты и пользователи в ней",
		// 	Usage:       "/info",
		// 	MinArgs:     0,
		// 	Handler: func(client *core_domain.Client, args []string) error {
		// 		return s.roomService.GetClients(args[0])
		// 	},
		// },
		{
			Name:        "/nick",
			Description: "Сменить никнейм",
			Usage:       "/nick <new_name>",
			MinArgs:     1,
			Handler: func(client *core_domain.Client, args []string) error {
				return s.clientService.ChangeNick(client.ID, args[0])
			},
		},
		{
			Name:        "/leave",
			Description: "Покинуть текущую команту",
			Usage:       "/leave",
			MinArgs:     0,
			Handler: func(client *core_domain.Client, args []string) error {
				return s.roomService.LeaveRoom(client)
			},
		},
	})
}

func (s *Server) Start(ctx context.Context) error {
	var err error
	s.listener, err = net.Listen("tcp", ":8080")
	if err != nil {
		return fmt.Errorf("Error listening: %v", err)
	}

	s.roomService.CreateRoom("general")
	s.log.Info("Server started on :8080")

	errCh := make(chan error, 1)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-ctx.Done():
				s.log.Debug("Прекращаем принимать соединения")
				return
			default:
			}

			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					s.log.Error("Error accepting conn", zap.Error(err))
					select {
					case errCh <- err:
					default:
					}
				}
				continue
			}
			go s.HandleClient(conn, ctx)
		}
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("listen TCP server: %w", err)
		}
	case <-ctx.Done():
		s.log.Warn("Остановка TCP сервера...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("Остановка TCP сервера: %w", err)
		}
		s.log.Warn("TCP сервер остановлен!")
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Warn("Остановка сервера...")

	s.mtx.Lock()

	if s.listener != nil {
		s.log.Debug("Закрываем listener")
		if err := s.listener.Close(); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				s.log.Warn("Ошибка закрытия listener", zap.Error(err))
			}
		}
	}

	clients := s.clientService.GetAllClients()
	s.log.Debug("Отправка уведомления клиентам", zap.Int("count", s.clientService.Count()))
	for _, client := range clients {
		if client.Conn != nil {
			fmt.Fprintf(client.Conn, "Сервер останавливается...\n")
		}
	}
	time.Sleep(100 * time.Millisecond)

	for _, client := range clients {
		if client.Conn != nil {
			client.Conn.Close()
			s.log.Debug("Закрыто соединение клиента", zap.String("client", client.ID))
		}
	}
	s.mtx.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.log.Info("Все горутины завершены")
		return nil
	case <-ctx.Done():
		s.log.Warn("Таймаут остановки сервера, принудительное завершение")
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
