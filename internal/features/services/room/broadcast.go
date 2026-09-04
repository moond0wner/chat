package room_service

import (
	"context"
	"fmt"
	core_domain "tcp_srv/internal/core/domain"
	"time"

	"go.uber.org/zap"
)

func (rs *RoomService) Broadcast(ctx context.Context, message core_domain.Message) error {
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	room, ok := rs.rooms[message.RoomID]
	if !ok {
		return fmt.Errorf("room '%d' not found", message.RoomID)
	}
	if !message.IsSystem {
		if err := rs.historyRepository.SaveMessage(ctx, &message); err != nil {
			return fmt.Errorf("db error: %w", err)
		}

	}
	var errs []error
	for id, client := range room.Clients {
		if message.SenderID != 0 && id == message.SenderID {
			continue
		}

		if err := client.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			rs.log.Warn("Timeout write data", zap.String("user_name", client.Name), zap.Int("user_id", client.ID), zap.Error(err))
			continue
		}
		_, err := fmt.Fprintln(client.Conn, message.Text)
		if err != nil {
			rs.log.Warn("Write to client failed", zap.Error(err))
		}
		if err = client.Conn.SetWriteDeadline(time.Time{}); err != nil {
			rs.log.Warn("Reset timeout", zap.String("user_name", client.Name), zap.Int("user_id", client.ID), zap.Error(err))
		}

		if err != nil {
			errs = append(errs, fmt.Errorf("client %d: %w", id, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("broadcast errors: %v", errs)
	}
	return nil
}
