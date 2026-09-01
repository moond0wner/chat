package history_postgres_repository

import (
	"context"
	"fmt"
	core_domain "tcp_srv/internal/core/domain"
)

func (r *HistoryRepository) SaveUser(ctx context.Context, client *core_domain.Client) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	var userID int
	var roomID int
	sqlQuery := `
	INSERT INTO users (nickname, room_id)
	VALUES ($1, (SELECT id FROM rooms WHERE name='register'))
	RETURNING id, room_id;
    `

	err := r.pool.QueryRow(ctx, sqlQuery, client.Name).Scan(&userID, &roomID)
	if err != nil {
		return fmt.Errorf("insert error: %w", err)
	}
	client.ID = userID
	client.RoomID = roomID
	return nil
}

func (r *HistoryRepository) UpdateUser(ctx context.Context, client *core_domain.Client) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	sqlQuery := `
	UPDATE users
	SET nickname = $1
	WHERE id = $2;
	`
	if _, err := r.pool.Exec(ctx, sqlQuery, client.Name, client.ID); err != nil {
		return fmt.Errorf("Exec error: %v", err)
	}
	return nil
}

func (r *HistoryRepository) IsNicknameTaken(ctx context.Context, nickname string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	var exists bool
	err := r.pool.QueryRow(ctx, `
    SELECT EXISTS (
    SELECT 1 FROM users WHERE nickname = $1
    )
    `, nickname).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check nickname exists error: %w", err)
	}
	return exists, nil
}

func (r *HistoryRepository) GetAllUsers(ctx context.Context) (*[]core_domain.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	sqlQuery := `
	SELECT id, nickname, room_id
	FROM users
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

	clients := make([]core_domain.Client, 0)
	for rows.Next() {
		var client core_domain.Client
		if err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.RoomID,
		); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		clients = append(clients, client)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}
	return &clients, nil
}
