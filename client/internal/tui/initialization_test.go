package tui

import (
	"fmt"
	"testing"
	"time"

	"github.com/maynagashev/gophkeeper/models"
	"github.com/stretchr/testify/assert"
)

// TestVersionItem_Title проверяет метод Title для versionItem.
func TestVersionItem_Title(t *testing.T) {
	tests := []struct {
		name      string
		item      versionItem
		wantTitle string
	}{
		{
			name: "Обычная версия",
			item: versionItem{
				version: models.VaultVersion{
					ID: 123,
				},
				isCurrent: false,
			},
			wantTitle: "Версия #123",
		},
		{
			name: "Текущая версия",
			item: versionItem{
				version: models.VaultVersion{
					ID: 456,
				},
				isCurrent: true,
			},
			wantTitle: "Версия #456 (Текущая)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantTitle, tt.item.Title())
		})
	}
}

// TestVersionItem_Description проверяет метод Description для versionItem.
func TestVersionItem_Description(t *testing.T) {
	now := time.Now()
	nowStr := now.Format(time.RFC3339)
	size := int64(2048) // 2 KB

	tests := []struct {
		name            string
		item            versionItem
		wantDescription string
	}{
		{
			name: "Только ID",
			item: versionItem{
				version: models.VaultVersion{
					ID: 1,
				},
			},
			wantDescription: "ID: 1",
		},
		{
			name: "Только время модификации",
			item: versionItem{
				version: models.VaultVersion{
					ID:                2,
					ContentModifiedAt: &now,
				},
			},
			wantDescription: fmt.Sprintf("Изменена: %s", nowStr),
		},
		{
			name: "Только размер",
			item: versionItem{
				version: models.VaultVersion{
					ID:        3,
					SizeBytes: &size,
				},
			},
			wantDescription: "Размер: 2.00 KB",
		},
		{
			name: "Время и размер",
			item: versionItem{
				version: models.VaultVersion{
					ID:                4,
					ContentModifiedAt: &now,
					SizeBytes:         &size,
				},
			},
			wantDescription: fmt.Sprintf("Изменена: %s | Размер: 2.00 KB", nowStr),
		},
		{
			name: "Текущая версия со временем и размером", // isCurrent не влияет на Description
			item: versionItem{
				version: models.VaultVersion{
					ID:                5,
					ContentModifiedAt: &now,
					SizeBytes:         &size,
				},
				isCurrent: true,
			},
			wantDescription: fmt.Sprintf("Изменена: %s | Размер: 2.00 KB", nowStr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantDescription, tt.item.Description())
		})
	}
}

// TestVersionItem_FilterValue проверяет метод FilterValue для versionItem.
func TestVersionItem_FilterValue(t *testing.T) {
	// FilterValue просто возвращает Title, поэтому тесты аналогичны TestVersionItem_Title
	tests := []struct {
		name            string
		item            versionItem
		wantFilterValue string
	}{
		{
			name: "Обычная версия",
			item: versionItem{
				version: models.VaultVersion{
					ID: 789,
				},
				isCurrent: false,
			},
			wantFilterValue: "Версия #789",
		},
		{
			name: "Текущая версия",
			item: versionItem{
				version: models.VaultVersion{
					ID: 101,
				},
				isCurrent: true,
			},
			wantFilterValue: "Версия #101 (Текущая)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantFilterValue, tt.item.FilterValue())
		})
	}
}
