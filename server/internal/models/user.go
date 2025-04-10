package models

import "time"

// User представляет пользователя системы.
// Тэги `db` используются для маппинга с полями БД с помощью sqlx.
// Тэги `json` используются для (де)сериализации JSON.
type User struct {
	ID           int64     `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"` // Не отправляем хеш пароля в JSON
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// RegisterRequest представляет тело запроса на регистрацию.
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest представляет тело запроса на вход.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse представляет тело ответа при успешном входе.
type LoginResponse struct {
	Token string `json:"token"`
}
