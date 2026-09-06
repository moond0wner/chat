package history_postgres_repository

import (
	"context"
	"fmt"

	core_domain "github.com/moond0wner/chat/internal/core/domain"
)

func (r *HistoryRepository) SaveRoom(ctx context.Context, room *core_domain.Room) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	var roomID int
	sqlQuery := `
	INSERT INTO rooms (name)
	VALUES ($1)
	RETURNING id;
	`
	row := r.pool.QueryRow(ctx, sqlQuery, room.Name)
	if err := row.Scan(
		&roomID,
	); err != nil {
		return fmt.Errorf("scan error: %v", err)
	}

	room.ID = roomID
	return nil
}

func (r *HistoryRepository) GetIDRoomByName(ctx context.Context, name string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	var roomID int
	sqlQuery := `
	SELECT id FROM rooms
	WHERE name=$1;
	`
	row := r.pool.QueryRow(ctx, sqlQuery, name)
	if err := row.Scan(
		&roomID,
	); err != nil {
		return 0, fmt.Errorf("scan error: %v", err)
	}

	return roomID, nil
}

func (r *HistoryRepository) GetAllRooms(ctx context.Context) (*[]core_domain.Room, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	sqlQuery := `
	SELECT id, name 
	FROM rooms
	ORDER BY id ASC;
	`

	rows, err := r.pool.Query(
		ctx,
		sqlQuery,
	)

	if err != nil {
		return nil, fmt.Errorf("select users: %w", err)
	}
	defer rows.Close()

	rooms := make([]core_domain.Room, 0)
	for rows.Next() {
		var room core_domain.Room
		if err = rows.Scan(
			&room.ID,
			&room.Name,
		); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		rooms = append(rooms, room)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}
	return &rooms, nil
}
