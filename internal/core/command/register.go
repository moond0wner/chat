package core_command

import (
	"context"
	"errors"

	core_domain "github.com/moond0wner/chat/internal/core/domain"
	core_logger "github.com/moond0wner/chat/internal/core/logger"
	client_service "github.com/moond0wner/chat/internal/features/services/client"
	room_service "github.com/moond0wner/chat/internal/features/services/room"
	"go.uber.org/zap"
)

func RegisterCommands(
	ctx context.Context,
	log *core_logger.Logger,
	roomService *room_service.RoomService,
	clientService *client_service.ClientService,
) *Manager {
	mng := NewManager()
	mng.Register([]CommandHandler{
		{
			Name:        "/join",
			Description: "Подключение к комнате",
			Usage:       "/join <room_name>",
			MinArgs:     1,
			Handler: func(ctx context.Context, client *core_domain.Client, args []string) error {
				if args[0] == "register" {
					return errors.New("Вы уже зарегистрированы, используйте /join для подключения к другим комнатам")
				}
				if client.RoomID == roomService.RegisterRoomID {
					return errors.New("Сначала зарегистрируйтесь с помощью команды /reg <nickname>")
				}
				return roomService.JoinRoom(ctx, client, args[0])
			},
		},
		{
			Name:        "/all_info",
			Description: "Информация о сервере: комнаты и пользователи в ней",
			Usage:       "/all_info",
			MinArgs:     0,
			Handler: func(ctx context.Context, client *core_domain.Client, args []string) error {
				text, err := roomService.GetAllInfo(client.ID)
				if err != nil {
					log.Warn("Error get all info about server", zap.Error(err))
					return errors.New("Не удалось собрать информацию о сервере")
				}
				if err = clientService.SendMessageToUser(client, text); err != nil {
					log.Warn("Error send message to user", zap.Error(err))
					return errors.New("Возникла непредвидения ошибка")
				}
				return nil
			},
		},
		{
			Name:        "/info",
			Description: "Информация о канале: пользователи в нем",
			Usage:       "/info",
			MinArgs:     0,
			Handler: func(ctx context.Context, client *core_domain.Client, args []string) error {
				room, err := roomService.GetRoomByID(ctx, client.RoomID)
				if err != nil {
					log.Warn("Error get room by id", zap.Error(err))
					return errors.New("Возникла непредвидения ошибка")
				}
				text, err := roomService.GetInfoAboutRoom(client.ID, room.Name)
				if err != nil {
					log.Warn("Error get info about room", zap.Error(err))
					return errors.New("Не удалось собрать информацию о комнате")
				}
				if err = clientService.SendMessageToUser(client, text); err != nil {
					log.Warn("Error send message to user", zap.Error(err))
					return errors.New("Возникла непредвидения ошибка")
				}
				return nil
			},
		},
		{
			Name:        "/nick",
			Description: "Сменить никнейм",
			Usage:       "/nick <new_name>",
			MinArgs:     1,
			Handler: func(ctx context.Context, client *core_domain.Client, args []string) error {
				if client.RoomID == roomService.RegisterRoomID {
					return errors.New("Сначала зарегистрируйтесь, используйте команду /reg <nickname>")
				}
				return clientService.ChangeNick(ctx, client, args[0])
			},
		},
		{
			Name:        "/leave",
			Description: "Покинуть текущую команту",
			Usage:       "/leave",
			MinArgs:     0,
			Handler: func(ctx context.Context, client *core_domain.Client, args []string) error {
				return roomService.LeaveRoom(ctx, client)
			},
		},
		{
			Name:        "/msg",
			Description: "Отправить личное сообщение пользователю",
			Usage:       "/msg <user_name> <text>",
			MinArgs:     2,
			Handler: func(ctx context.Context, client *core_domain.Client, args []string) error {
				if client.RoomID == roomService.RegisterRoomID {
					return errors.New("Сначала зарегистрируйтесь, используйте команду /reg <nickname>")
				}
				message := core_domain.PrivateMessage{
					SenderName:    client.Name,
					RecipientName: args[0],
					Text:          args[1],
				}
				return clientService.SendPrivateMessage(message, client)
			},
		},
		{
			Name:        "/reg",
			Description: "Зарегистрировать ник",
			Usage:       "/reg <nickname>",
			MinArgs:     1,
			Handler: func(ctx context.Context, client *core_domain.Client, args []string) error {
				newName := args[0]
				if err := clientService.ChangeNick(ctx, client, newName); err != nil {
					log.Warn("Error change nick", zap.Int("client_id", client.ID), zap.Error(err))
					return errors.New("Не удалось изменить имя")
				}
				return roomService.JoinRoom(ctx, client, "general")
			},
		},
	})

	return mng
}
