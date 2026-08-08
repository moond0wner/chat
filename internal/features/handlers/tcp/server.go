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
				if err := s.SendMessageToUser(client, text); err != nil {
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
				text, err := s.roomService.GetInfoAboutRoom(client.ID, client.RoomID)
				if err != nil {
					return fmt.Errorf("Error get info about room: %v", err)
				}
				if err := s.SendMessageToUser(client, text); err != nil {
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
		{
			Name:        "/msg",
			Description: "Отправить личное сообщение пользователю",
			Usage:       "/msg <user_name> <text>",
			MinArgs:     2,
			Handler: func(client *core_domain.Client, args []string) error {
				message := core_domain.PrivateMessage{
					SenderName:    client.Name,
					RecipientName: args[0],
					Text:          args[1],
				}
				return s.clientService.SendPrivateMessage(message)
			},
		},
		{
			Name:        "/reg",
			Description: "Зарегистрировать ник",
			Usage:       "/reg <nickname>",
			MinArgs:     1,
			Handler: func(client *core_domain.Client, args []string) error {
				newName := args[0]
				if err := s.clientService.ChangeNick(client.ID, newName); err != nil {
					return err
				}
				return s.roomService.JoinRoom(client, "general")
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

	s.roomService.CreateRoom("register")
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
			go s.RegisterInServer(ctx, conn)
		}
	}()

	select {
	case <-ctx.Done():
		s.log.Warn("Остановка TCP сервера...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("Остановка TCP сервера: %w", err)
		}
		s.log.Warn("TCP сервер остановлен!")
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("listen TCP server: %w", err)
		}
	}
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
			s.log.Debug("Закрыто соединение клиента", zap.String("client", client.ID))
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
			zap.String("client_id", client.ID),
			zap.Error(err))
		return fmt.Errorf("Ошибка отправки сообщения: %v", err)
	}
	return nil
}
