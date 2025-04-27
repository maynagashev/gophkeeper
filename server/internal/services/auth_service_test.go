package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/maynagashev/gophkeeper/models"
	"github.com/maynagashev/gophkeeper/server/internal/mocks"
	"github.com/maynagashev/gophkeeper/server/internal/repository"
	"github.com/maynagashev/gophkeeper/server/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestNewAuthService(t *testing.T) {
	mockUserRepo := new(mocks.UserRepository)

	authService := services.NewAuthService(mockUserRepo)

	require.NotNil(t, authService)
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()
	username := "testuser"
	password := "password123"

	tests := []struct {
		name          string
		mockSetup     func(mockUserRepo *mocks.UserRepository)
		expectedError error
	}{
		{
			name: "Успешная регистрация",
			mockSetup: func(mockUserRepo *mocks.UserRepository) {
				mockUserRepo.EXPECT().
					CreateUser(ctx, mock.AnythingOfType("*models.User")).
					Return(int64(1), nil).Once()
			},
			expectedError: nil,
		},
		{
			name: "Имя пользователя занято",
			mockSetup: func(mockUserRepo *mocks.UserRepository) {
				mockUserRepo.EXPECT().
					CreateUser(ctx, mock.AnythingOfType("*models.User")).
					Return(int64(0), repository.ErrUsernameTaken).Once()
			},
			expectedError: services.ErrUsernameTaken,
		},
		{
			name: "Ошибка репозитория при создании",
			mockSetup: func(mockUserRepo *mocks.UserRepository) {
				mockUserRepo.EXPECT().
					CreateUser(ctx, mock.AnythingOfType("*models.User")).
					Return(int64(0), errors.New("some db error")).Once()
			},
			expectedError: errors.New("внутренняя ошибка сервера при создании пользователя"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(mocks.UserRepository)
			tt.mockSetup(mockUserRepo)

			authService := services.NewAuthService(mockUserRepo)
			err := authService.Register(username, password)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}

			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	username := "testuser"
	password := "password123"
	wrongPassword := "wrongpassword"
	userID := int64(1)
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err, "Не удалось сгенерировать хеш пароля для тестов")
	hashedPassword := string(hashedPasswordBytes)

	correctUser := &models.User{
		ID:           userID,
		Username:     username,
		PasswordHash: hashedPassword,
	}

	tests := []struct {
		name          string
		passwordToUse string
		mockSetup     func(mockUserRepo *mocks.UserRepository)
		expectedToken bool
		expectedError error
	}{
		{
			name:          "Успешный вход",
			passwordToUse: password,
			mockSetup: func(mockUserRepo *mocks.UserRepository) {
				mockUserRepo.EXPECT().
					GetUserByUsername(ctx, username).
					Return(correctUser, nil).Once()
			},
			expectedToken: true,
			expectedError: nil,
		},
		{
			name:          "Пользователь не найден",
			passwordToUse: password,
			mockSetup: func(mockUserRepo *mocks.UserRepository) {
				mockUserRepo.EXPECT().
					GetUserByUsername(ctx, username).
					Return(nil, repository.ErrUserNotFound).Once()
			},
			expectedToken: false,
			expectedError: services.ErrInvalidCredentials,
		},
		{
			name:          "Неверный пароль",
			passwordToUse: wrongPassword,
			mockSetup: func(mockUserRepo *mocks.UserRepository) {
				mockUserRepo.EXPECT().
					GetUserByUsername(ctx, username).
					Return(correctUser, nil).Once()
			},
			expectedToken: false,
			expectedError: services.ErrInvalidCredentials,
		},
		{
			name:          "Ошибка репозитория при поиске",
			passwordToUse: password,
			mockSetup: func(mockUserRepo *mocks.UserRepository) {
				mockUserRepo.EXPECT().
					GetUserByUsername(ctx, username).
					Return(nil, errors.New("some db error")).Once()
			},
			expectedToken: false,
			expectedError: errors.New("внутренняя ошибка сервера при поиске пользователя"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(mocks.UserRepository)
			tt.mockSetup(mockUserRepo)

			authService := services.NewAuthService(mockUserRepo)
			token, loginErr := authService.Login(username, tt.passwordToUse)

			if tt.expectedError != nil {
				require.Error(t, loginErr)
				require.EqualError(t, loginErr, tt.expectedError.Error())
				assert.Empty(t, token)
			} else {
				require.NoError(t, loginErr)
				assert.NotEmpty(t, token)
			}

			mockUserRepo.AssertExpectations(t)
		})
	}
}
