package tcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	core_command "tcp_srv/internal/core/command"
	core_config "tcp_srv/internal/core/config"
	core_domain "tcp_srv/internal/core/domain"
	core_logger "tcp_srv/internal/core/logger"
	client_service "tcp_srv/internal/features/services/client"
	room_service "tcp_srv/internal/features/services/room"
	"time"

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

func NewServer(logger *core_logger.Logger, rs *room_service.RoomService, cs *client_service.ClientService, cfg *core_config.Config) *Server {
	srv := &Server{
		roomService:   rs,
		clientService: cs,
		Manager:       core_command.NewManager(),
		config:        cfg,
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
				if args[0] == "register" {
					return errors.New("already registered. Use /join to join another room")
				}
				if client.RoomID == s.roomService.RegisterRoomID {
					return errors.New("please register first using /reg <nickname>")
				}
				return s.roomService.JoinRoom(context.Background(), client, args[0])
			},
		},
		{
			Name:        "/all_info",
			Description: "Информация о сервере: комнаты и пользователи в ней",
			Usage:       "/all_info",
			MinArgs:     0,
			Handler: func(client *core_domain.Client, args []string) error {
				text, err := s.roomService.GetAllInfo(client.ID)
				if err != nil {
					return fmt.Errorf("Error get all info about server: %v", err)
				}
				if err = s.SendMessageToUser(client, text); err != nil {
					return fmt.Errorf("Error send message to user: %v", err)
				}
				return nil
			},
		},
		{
			Name:        "/info",
			Description: "Информация о канале: пользователи в нем",
			Usage:       "/info",
			MinArgs:     0,
			Handler: func(client *core_domain.Client, args []string) error {
				room, err := s.roomService.GetRoomByID(context.Background(), client.RoomID)
				if err != nil {
					return fmt.Errorf("Error get room by id: %v", err)
				}
				text, err := s.roomService.GetInfoAboutRoom(client.ID, room.Name)
				if err != nil {
					return fmt.Errorf("Error get info about room: %v", err)
				}
				if err = s.SendMessageToUser(client, text); err != nil {
					return fmt.Errorf("Error send message to user: %v", err)
				}
				return nil
			},
		},
		{
			Name:        "/nick",
			Description: "Сменить никнейм",
			Usage:       "/nick <new_name>",
			MinArgs:     1,
			Handler: func(client *core_domain.Client, args []string) error {
				if client.RoomID == s.roomService.RegisterRoomID {
					return errors.New("registration required")
				}
				return s.clientService.ChangeNick(context.Background(), client, args[0])
			},
		},
		{
			Name:        "/leave",
			Description: "Покинуть текущую команту",
			Usage:       "/leave",
			MinArgs:     0,
			Handler: func(client *core_domain.Client, args []string) error {
				return s.roomService.LeaveRoom(context.Background(), client)
			},
		},
		{
			Name:        "/msg",
			Description: "Отправить личное сообщение пользователю",
			Usage:       "/msg <user_name> <text>",
			MinArgs:     2,
			Handler: func(client *core_domain.Client, args []string) error {
				if client.RoomID == s.roomService.RegisterRoomID {
					return errors.New("registration required")
				}
				message := core_domain.PrivateMessage{
					SenderName:    client.Name,
					RecipientName: args[0],
					Text:          args[1],
				}
				return s.clientService.SendPrivateMessage(message, client)
			},
		},
		{
			Name:        "/reg",
			Description: "Зарегистрировать ник",
			Usage:       "/reg <nickname>",
			MinArgs:     1,
			Handler: func(client *core_domain.Client, args []string) error {
				newName := args[0]
				if err := s.clientService.ChangeNick(context.Background(), client, newName); err != nil {
					return err
				}
				return s.roomService.JoinRoom(context.Background(), client, "general")
			},
		},
	})
}

func (s *Server) Start(ctx context.Context) error {
	var err error
	s.listener, err = net.Listen("tcp", s.config.ServerPort)
	if err != nil {
		return fmt.Errorf("error listening: %w", err)
	}
	s.log.Info("Server started", zap.String("port", s.config.ServerPort))

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

			var conn net.Conn
			conn, err = s.listener.Accept()
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
	s.log.Warn("Остановка сервера...")
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

	for _, client := range clients {
		if client.Conn != nil {
			client.Conn.Close()
			s.log.Debug("Закрыто соединение клиента", zap.Int("client", client.ID))
		}
	}

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

func (s *Server) SendMessageToUser(client *core_domain.Client, text string) error {
	if _, err := fmt.Fprintln(client.Conn, text); err != nil {
		s.log.Warn("Ошибка отправки сообщения",
			zap.Int("client_id", client.ID),
			zap.Error(err))
		return fmt.Errorf("Ошибка отправки сообщения: %v", err)
	}
	return nil
}
