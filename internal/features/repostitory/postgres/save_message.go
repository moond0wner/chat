package history_postgres_repository

import (
	"context"
	"fmt"
	core_domain "tcp_srv/internal/core/domain"
	"time"
)

func (r *HistoryRepository) SaveMessage(
	ctx context.Context,
	message *core_domain.Message,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	sqlQuery := `
        INSERT INTO messages (user_id, room_id, message, created_at)
        VALUES ($1, $2, $3, $4)
    `

	_, err := r.pool.Exec(
		ctx,
		sqlQuery,
		message.SenderID,
		message.RoomID,
		message.Text,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("exec insert message: %w", err)
	}

	return nil
}
