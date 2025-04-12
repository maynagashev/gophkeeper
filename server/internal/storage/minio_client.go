package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// FileStorage определяет интерфейс для взаимодействия с объектным хранилищем.
type FileStorage interface {
	UploadFile(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error
	DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error)
	// TODO: Добавить другие методы, если понадобятся (например, DeleteFile, GetFileInfo)
}

// MinioClient реализует FileStorage для MinIO.
type MinioClient struct {
	client     *minio.Client
	bucketName string
}

// MinioConfig содержит параметры для подключения к MinIO.
type MinioConfig struct {
	Endpoint        string // Адрес MinIO (например, "localhost:9000")
	AccessKeyID     string // Логин
	SecretAccessKey string // Пароль
	UseSSL          bool   // Использовать SSL (обычно false для локальной разработки)
	BucketName      string // Имя бакета для хранения файлов
	Region          string // Регион (не обязательно для MinIO, но может требоваться)
}

// NewMinioClient создает новый клиент MinIO.
func NewMinioClient(cfg MinioConfig) (*MinioClient, error) {
	log.Printf("Инициализация клиента MinIO для эндпоинта %s...", cfg.Endpoint)

	// Инициализация клиента MinIO
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации клиента MinIO: %w", err)
	}

	// Проверка доступности MinIO
	// Необязательно, но полезно для раннего обнаружения проблем
	_, err = minioClient.ListBuckets(context.Background())
	if err != nil {
		log.Printf("Предупреждение: не удалось проверить соединение с MinIO: %v. Проверьте доступность и креды.", err)
		// Не возвращаем ошибку, чтобы сервер мог запуститься, даже если MinIO временно недоступен
	}

	// Проверка существования бакета и создание при необходимости
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки существования бакета '%s': %w", cfg.BucketName, err)
	}
	if !exists {
		log.Printf("Бакет '%s' не найден, попытка создания...", cfg.BucketName)
		err = minioClient.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{Region: cfg.Region})
		if err != nil {
			return nil, fmt.Errorf("ошибка создания бакета '%s': %w", cfg.BucketName, err)
		}
		log.Printf("Бакет '%s' успешно создан.", cfg.BucketName)
	} else {
		log.Printf("Бакет '%s' уже существует.", cfg.BucketName)
	}

	log.Printf("Клиент MinIO успешно инициализирован для бакета '%s'.", cfg.BucketName)
	return &MinioClient{
		client:     minioClient,
		bucketName: cfg.BucketName,
	}, nil
}

// UploadFile загружает файл в MinIO.
func (c *MinioClient) UploadFile(
	ctx context.Context,
	objectKey string,
	reader io.Reader,
	size int64,
	contentType string,
) error {
	log.Printf("[Minio] Загрузка файла '%s' в бакет '%s'...", objectKey, c.bucketName)

	// Опции загрузки
	opts := minio.PutObjectOptions{
		ContentType: contentType,
		// Можно добавить другие метаданные при необходимости
		// UserMetadata: map[string]string{"x-amz-meta-my-key": "your-value"},
	}

	// Загружаем объект
	uploadInfo, err := c.client.PutObject(ctx, c.bucketName, objectKey, reader, size, opts)
	if err != nil {
		log.Printf("[Minio] Ошибка загрузки файла '%s': %v", objectKey, err)
		return fmt.Errorf("ошибка загрузки файла в MinIO: %w", err)
	}

	log.Printf("[Minio] Файл '%s' успешно загружен, размер: %d, ETag: %s", objectKey, uploadInfo.Size, uploadInfo.ETag)
	return nil
}

// DownloadFile скачивает файл из MinIO.
// Возвращает io.ReadCloser, который нужно закрыть после использования.
func (c *MinioClient) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	log.Printf("[Minio] Скачивание файла '%s' из бакета '%s'...", objectKey, c.bucketName)

	// Получаем объект
	object, err := c.client.GetObject(ctx, c.bucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		// Проверяем, является ли ошибка "NoSuchKey"
		var minioErr minio.ErrorResponse
		if errors.As(err, &minioErr) && minioErr.Code == "NoSuchKey" {
			log.Printf("[Minio] Файл '%s' не найден в бакете '%s'", objectKey, c.bucketName)
			return nil, ErrObjectNotFound // Возвращаем кастомную ошибку
		}
		log.Printf("[Minio] Ошибка получения файла '%s': %v", objectKey, err)
		return nil, fmt.Errorf("ошибка получения файла из MinIO: %w", err)
	}

	// Проверяем метаданные объекта (необязательно, но может быть полезно)
	/*
		stat, err := object.Stat()
		if err != nil {
			// Важно закрыть тело объекта, даже если Stat() вернул ошибку
			_ = object.Close()
			log.Printf("[Minio] Ошибка получения метаданных для файла '%s': %v", objectKey, err)
			return nil, fmt.Errorf("ошибка получения метаданных из MinIO: %w", err)
		}
		log.Printf("[Minio] Файл '%s' найден, размер: %d, ContentType: %s", objectKey, stat.Size, stat.ContentType)
	*/

	log.Printf("[Minio] Файл '%s' успешно получен для скачивания", objectKey)
	return object, nil // Возвращаем тело объекта (io.ReadCloser)
}

// Кастомная ошибка хранилища.
var (
	ErrObjectNotFound = errors.New("объект не найден в хранилище")
)
