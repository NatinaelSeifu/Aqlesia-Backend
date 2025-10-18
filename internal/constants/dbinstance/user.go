package dbinstance

import (
	"aqlesia/internal/constants/model/dto"
	"context"
)

const getUsers = `-- name: GetUsers :many
SELECT id, name, lastname, phone_number, password, telegram_id, created_at, updated_at, deleted_at FROM users WHERE deleted_at IS NULL
`

func (q *DBInstance) GetAllUsers(ctx context.Context) ([]dto.User, error) {
	rows, err := q.Pool.Query(ctx, getUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []dto.User
	for rows.Next() {
		var i dto.User
		var password string // Don't include password in response
		var telegramID *string
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.LastName,
			&i.PhoneNumber,
			&password, // Skip password in response
			&telegramID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.DeletedAt,
		); err != nil {
			return nil, err
		}
		i.TelegramID = telegramID
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
