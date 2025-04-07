//go:build ignore
// +build ignore

// Этот файл используется для генерации тестовой базы данных KDBX.
// Запустить: go run generate_test_db.go

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

const (
	testDBPassword = "qwe"
	testDBPath     = "test.kdbx"
)

func main() {
	// Создаем директорию, если она не существует
	err := os.MkdirAll(filepath.Dir(testDBPath), 0755)
	if err != nil {
		log.Fatalf("ошибка создания директории: %v", err)
	}

	// Создаем новую БД
	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(testDBPassword)
	db.Content = &gokeepasslib.DBContent{
		Meta: &gokeepasslib.DBMeta{
			Generator:                  "GophKeeper Test",
			DatabaseName:               w.String("Test Database"),
			DatabaseNameChanged:        w.Time(w.Now()),
			DatabaseDescription:        w.String("Database for testing"),
			DatabaseDescriptionChanged: w.Time(w.Now()),
			DefaultUserName:            w.String("test"),
			DefaultUserNameChanged:     w.Time(w.Now()),
			MaintenanceHistoryDays:     w.Int32(365),
			RecycleBinEnabled:          w.Bool(true),
			RecycleBinUUID:             nil,
			RecycleBinChanged:          w.Time(w.Now()),
			HistoryMaxItems:            w.Int32(10),
			HistoryMaxSize:             w.Int32(6291456), // 6 MB
		},
		Root: &gokeepasslib.RootData{
			Groups: []gokeepasslib.Group{
				{
					Name: "TestGroup",
					UUID: gokeepasslib.NewUUID(),
					Times: gokeepasslib.TimeData{
						CreationTime:         w.Time(w.Now()),
						LastModificationTime: w.Time(w.Now()),
						LastAccessTime:       w.Time(w.Now()),
					},
					Entries: []gokeepasslib.Entry{
						{
							UUID: gokeepasslib.NewUUID(),
							Times: gokeepasslib.TimeData{
								CreationTime:         w.Time(w.Now()),
								LastModificationTime: w.Time(w.Now()),
								LastAccessTime:       w.Time(w.Now()),
							},
							Values: []gokeepasslib.ValueData{
								{Key: "Title", Value: gokeepasslib.V{Content: "test entry 1"}},
								{Key: "UserName", Value: gokeepasslib.V{Content: "user1"}},
								{Key: "Password", Value: gokeepasslib.V{Content: "password1", Protected: w.Bool(true)}},
								{Key: "URL", Value: gokeepasslib.V{Content: "https://example.com"}},
								{Key: "Notes", Value: gokeepasslib.V{Content: "Test notes"}},
							},
						},
						{
							UUID: gokeepasslib.NewUUID(),
							Times: gokeepasslib.TimeData{
								CreationTime:         w.Time(w.Now()),
								LastModificationTime: w.Time(w.Now()),
								LastAccessTime:       w.Time(w.Now()),
							},
							Values: []gokeepasslib.ValueData{
								{Key: "Title", Value: gokeepasslib.V{Content: "test entry 2"}},
								{Key: "UserName", Value: gokeepasslib.V{Content: "user2"}},
								{Key: "Password", Value: gokeepasslib.V{Content: "password2", Protected: w.Bool(true)}},
								{Key: "URL", Value: gokeepasslib.V{Content: "https://example.org"}},
							},
						},
					},
				},
			},
		},
	}

	// Шифруем защищенные значения
	db.LockProtectedEntries()

	// Открываем файл для записи
	file, err := os.Create(testDBPath)
	if err != nil {
		log.Fatalf("ошибка создания файла: %v", err)
	}
	defer file.Close()

	// Кодируем и записываем БД в файл
	keepassEncoder := gokeepasslib.NewEncoder(file)
	if err := keepassEncoder.Encode(db); err != nil {
		log.Fatalf("ошибка кодирования и записи БД: %v", err)
	}

	fmt.Printf("Тестовая база данных создана успешно по пути %s\n", testDBPath)
	fmt.Printf("Пароль: %s\n", testDBPassword)
}
