package common

import (
	"context"
	"fmt"

	core_domain "github.com/moond0wner/chat/internal/core/domain"
	core_logger "github.com/moond0wner/chat/internal/core/logger"
	core_transport "github.com/moond0wner/chat/internal/core/transport"
	client_service "github.com/moond0wner/chat/internal/features/services/client"
	room_service "github.com/moond0wner/chat/internal/features/services/room"
	"go.uber.org/zap"
)

func RegisterAndJoin(
	ctx context.Context,
	conn core_transport.Conn,
	roomName string,
	welcomeMessage string,
	clientService *client_service.ClientService,
	roomService *room_service.RoomService,
	log *core_logger.Logger,
) *core_domain.Client {
	client := clientService.RegisterClient(ctx, conn)
	if client == nil {
		conn.Close()
		return nil
	}

	if err := roomService.JoinRoom(ctx, client, roomName); err != nil {
		log.Error("Error connect to room",
			zap.String("room", roomName),
			zap.Error(err))
		conn.Close()
		return nil
	}

	if _, err := fmt.Fprintf(client.Conn, "%s\n", welcomeMessage); err != nil {
		log.Warn("Error sending greeting",
			zap.Int("client_id", client.ID),
			zap.Error(err))
	}

	log.Debug("New connection",
		zap.Int("client_id", client.ID),
		zap.Int("room_id", client.RoomID))

	return client
}
