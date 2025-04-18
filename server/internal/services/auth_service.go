package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/maynagashev/gophkeeper/models"
	"github.com/maynagashev/gophkeeper/server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// AuthService определяет интерфейс для сервиса аутентификации.
type AuthService interface {
	Register(username, password string) error
	Login(username, password string) (string, error) // Возвращает JWT токен или ошибку
}

// Константы для JWT.
const (
	// TODO: Вынести секретный ключ в конфигурацию/переменные окружения!
	jwtSecretKey = "your-very-secret-key"
	tokenTTL     = time.Hour * 24 // Время жизни токена - 24 часа
)

// Структура для пользовательских данных в JWT (claims).
type jwtClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// Убедимся, что authService удовлетворяет интерфейсу AuthService.
var _ AuthService = (*authService)(nil)

type authService struct {
	userRepo repository.UserRepository // Зависимость от репозитория пользователей
}

// NewAuthService создает новый экземпляр сервиса аутентификации.
func NewAuthService(userRepo repository.UserRepository) AuthService { // Возвращаем интерфейс
	return &authService{userRepo: userRepo}
}

// Register регистрирует нового пользователя.
func (s *authService) Register(username, password string) error {
	ctx := context.Background() // Используем фоновый контекст для операций сервиса

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[AuthService] Ошибка хеширования пароля для '%s': %v", username, err)
		return errors.New("внутренняя ошибка сервера при хешировании пароля")
	}

	user := &models.User{
		Username:     username,
		PasswordHash: string(hashedPassword),
	}

	// Создаем пользователя через репозиторий
	_, err = s.userRepo.CreateUser(ctx, user)
	if err != nil {
		if errors.Is(err, repository.ErrUsernameTaken) {
			log.Printf("[AuthService] Попытка регистрации с занятым именем: %s", username)
			return ErrUsernameTaken // Возвращаем ошибку слоя сервиса
		}
		log.Printf("[AuthService] Непредвиденная ошибка репозитория при регистрации '%s': %v", username, err)
		return errors.New("внутренняя ошибка сервера при создании пользователя")
	}

	log.Printf("[AuthService] Пользователь '%s' успешно зарегистрирован", username)
	return nil
}

// Login аутентифицирует пользователя и возвращает JWT токен.
func (s *authService) Login(username, password string) (string, error) {
	ctx := context.Background()

	// Получаем пользователя по имени пользователя
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			log.Printf("[AuthService] Попытка входа несуществующего пользователя: %s", username)
			return "", ErrInvalidCredentials // Общая ошибка для несуществующего пользователя и неверного пароля
		}
		log.Printf("[AuthService] Ошибка репозитория при поиске '%s': %v", username, err)
		return "", errors.New("внутренняя ошибка сервера при поиске пользователя")
	}

	// Сравниваем предоставленный пароль с хешем из базы данных
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		// Ошибка сравнения означает неверный пароль (или другую проблему bcrypt)
		log.Printf("[AuthService] Неверный пароль для пользователя: %s", username)
		return "", ErrInvalidCredentials // Общая ошибка
	}

	// Генерируем JWT токен
	token, err := s.generateJWT(user.ID)
	if err != nil {
		log.Printf("[AuthService] Ошибка генерации JWT для '%s': %v", username, err)
		return "", errors.New("внутренняя ошибка сервера при генерации токена")
	}

	log.Printf("[AuthService] Пользователь '%s' успешно аутентифицирован", username)
	return token, nil
}

// generateJWT создает и подписывает JWT токен для пользователя.
func (s *authService) generateJWT(userID int64) (string, error) {
	// Создаем claims (полезную нагрузку)
	claims := jwtClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)), // Время истечения
			IssuedAt:  jwt.NewNumericDate(time.Now()),               // Время выдачи
			NotBefore: jwt.NewNumericDate(time.Now()),               // Время, с которого токен валиден
			Issuer:    "gophkeeper-server",                          // Источник токена
		},
	}

	// Создаем токен с нашими claims и методом подписи HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен секретным ключом
	signedToken, err := token.SignedString([]byte(jwtSecretKey))
	if err != nil {
		return "", fmt.Errorf("ошибка подписи JWT: %w", err)
	}

	return signedToken, nil
}

// Кастомные ошибки сервиса.
var (
	ErrInvalidCredentials = errors.New("неверное имя пользователя или пароль")
	ErrUsernameTaken      = errors.New("имя пользователя уже занято")
)
