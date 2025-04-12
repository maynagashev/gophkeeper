package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Тип для ключа контекста.
type contextKey string

// Ключ для хранения ID пользователя в контексте.
const UserIDKey contextKey = "userID"

// TODO: Вынести секретный ключ в конфигурацию/переменные окружения! (Дублируется с services)
const jwtSecretKey = "your-very-secret-key"

// Структура для пользовательских данных в JWT (claims) - должна совпадать с той, что в services.
type jwtClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// Authenticator проверяет JWT токен аутентификации.
func Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем заголовок Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Println("[AuthMiddleware] Заголовок Authorization отсутствует")
			http.Error(w, "Требуется аутентификация", http.StatusUnauthorized)
			return
		}

		// Проверяем формат "Bearer token"
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || strings.ToLower(headerParts[0]) != "bearer" {
			log.Printf("[AuthMiddleware] Неверный формат заголовка Authorization: %s", authHeader)
			http.Error(w, "Неверный формат токена", http.StatusUnauthorized)
			return
		}

		tokenString := headerParts[1]

		// Парсим и валидируем токен
		claims := &jwtClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Убеждаемся, что метод подписи - HS256
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
			}
			// Возвращаем секретный ключ
			return []byte(jwtSecretKey), nil
		})

		if err != nil {
			log.Printf("[AuthMiddleware] Ошибка парсинга/валидации токена: %v", err)
			http.Error(w, "Невалидный токен", http.StatusUnauthorized)
			return
		}

		// Проверяем валидность токена (включая время жизни, issuer и т.д.)
		if !token.Valid {
			log.Println("[AuthMiddleware] Предоставлен невалидный токен (возможно, истек)")
			http.Error(w, "Невалидный токен", http.StatusUnauthorized)
			return
		}

		// Добавляем UserID в контекст запроса
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)

		// Логируем успешную аутентификацию
		log.Printf("[AuthMiddleware] Пользователь %d успешно аутентифицирован", claims.UserID)

		// Передаем управление следующему обработчику с обновленным контекстом
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext извлекает UserID из контекста запроса.
// Возвращает ID пользователя и true, если ID найден, иначе 0 и false.
func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}
