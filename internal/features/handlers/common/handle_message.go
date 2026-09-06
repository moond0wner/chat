package common

import (
	"context"
	"fmt"
	"strings"
	core_logger "github.com/moond0wner/chat/internal/core/logger"
	room_service "github.com/moond0wner/chat/internal/features/services/room"
	"time"

	core_command "github.com/moond0wner/chat/internal/core/command"
	core_domain "github.com/moond0wner/chat/internal/core/domain"
	"go.uber.org/zap"
)

func ProcessMessage(
	ctx context.Context,
	client *core_domain.Client,
	msg string,
	manager *core_command.Manager,
	roomService *room_service.RoomService,
	logger *core_logger.Logger,
) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}

	if client.RoomID == roomService.RegisterRoomID {
		if !strings.HasPrefix(msg, "/") {
			fmt.Fprintln(client.Conn, "Сначала зарегистрируйтесь: /reg <ник>")
			return
		}
	}

	if strings.HasPrefix(msg, "/") {
		if err := manager.Execute(ctx, msg, client); err != nil {
			fmt.Fprintf(client.Conn, "Error: %s\n", err.Error())
			logger.Warn("Command error",
				zap.Int("client_id", client.ID),
				zap.String("command", msg),
				zap.Error(err))
		}
		return
	}

	formattedMsg := fmt.Sprintf("[%d] %s: %s\n", client.RoomID, client.Name, msg)
	logger.Info("Сообщение",
		zap.Int("room_id", client.RoomID),
		zap.String("sender", client.Name),
		zap.String("message", msg),
	)

	broadcastMsg := core_domain.Message{
		RoomID:     client.RoomID,
		SenderName: client.Name,
		SenderID:   client.ID,
		Text:       formattedMsg,
		SendAt:     time.Now(),
		IsSystem:   false,
	}

	if err := roomService.Broadcast(ctx, broadcastMsg); err != nil {
		logger.Error("Error broadcast",
			zap.Int("room_id", client.RoomID),
			zap.Error(err))
	}
}
