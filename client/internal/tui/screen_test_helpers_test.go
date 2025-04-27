package tui //nolint:testpackage // Используем тот же пакет для доступа к неэкспортируемым типам

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScreenTestMockAPIClient_Login проверяет мок метода Login.
func TestScreenTestMockAPIClient_Login(t *testing.T) {
	mockClient := new(ScreenTestMockAPIClient)
	ctx := context.Background()
	username := "testuser"
	password := "testpass"
	expectedToken := "mock-token"
	expectedErr := errors.New("mock login error")

	// Тест успешного входа
	t.Run("Success", func(t *testing.T) {
		mockClient.On("Login", ctx, username, password).Return(expectedToken, nil).Once()
		token, err := mockClient.Login(ctx, username, password)
		require.NoError(t, err) // Используем require
		assert.Equal(t, expectedToken, token)
		mockClient.AssertExpectations(t)
	})

	// Тест ошибки входа
	t.Run("Error", func(t *testing.T) {
		mockClient.On("Login", ctx, username, password).Return("", expectedErr).Once()
		token, err := mockClient.Login(ctx, username, password)
		require.Error(t, err) // Используем require
		assert.Equal(t, expectedErr, err)
		assert.Empty(t, token)
		mockClient.AssertExpectations(t)
	})
}

// TestScreenTestMockAPIClient_Register проверяет мок метода Register.
func TestScreenTestMockAPIClient_Register(t *testing.T) {
	mockClient := new(ScreenTestMockAPIClient)
	ctx := context.Background()
	username := "newuser"
	password := "newpass"
	expectedErr := errors.New("mock register error")

	// Тест успешной регистрации
	t.Run("Success", func(t *testing.T) {
		mockClient.On("Register", ctx, username, password).Return(nil).Once()
		err := mockClient.Register(ctx, username, password)
		require.NoError(t, err) // Используем require
		mockClient.AssertExpectations(t)
	})

	// Тест ошибки регистрации
	t.Run("Error", func(t *testing.T) {
		mockClient.On("Register", ctx, username, password).Return(expectedErr).Once()
		err := mockClient.Register(ctx, username, password)
		require.Error(t, err) // Используем require
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})
}
