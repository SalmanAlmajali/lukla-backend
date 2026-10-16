package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Storage mendefinisikan antarmuka abstraksi penyimpanan berkas/objek.
type Storage interface {
	Upload(ctx context.Context, file io.Reader, key string, contentType string) (string, error)
	Download(ctx context.Context, key string) (io.ReadCloser, string, int64, error)
	Delete(ctx context.Context, key string) error
	Bucket() string
	Region() string
}

// Config menyimpan konfigurasi koneksi Amazon S3.
type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

// S3Storage adalah implementasi Storage menggunakan Amazon S3 SDK v2.
type S3Storage struct {
	client *s3.Client
	bucket string
	region string
}

// NewS3Storage membuat instance S3Storage baru dengan kredensial yang diberikan.
func NewS3Storage(ctx context.Context, cfg Config) (*S3Storage, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket name is required")
	}
	if cfg.Region == "" {
		cfg.Region = "ap-southeast-1"
	}

	var optFns []func(*awsconfig.LoadOptions) error
	optFns = append(optFns, awsconfig.WithRegion(cfg.Region))

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		optFns = append(optFns, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)

	return &S3Storage{
		client: client,
		bucket: cfg.Bucket,
		region: cfg.Region,
	}, nil
}

// Upload mengunggah stream berkas ke bucket Amazon S3 dengan key dan MIME type yang ditentukan.
func (s *S3Storage) Upload(ctx context.Context, file io.Reader, key string, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("s3 put object %s: %w", key, err)
	}

	// Format URL objek S3 standar
	fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
	return fileURL, nil
}

// Download mengambil stream berkas dari Amazon S3 beserta content-type dan ukurannya.
func (s *S3Storage) Download(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	out, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, "", 0, fmt.Errorf("s3 get object %s: %w", key, err)
	}

	contentType := "application/octet-stream"
	if out.ContentType != nil {
		contentType = *out.ContentType
	}

	var contentLength int64
	if out.ContentLength != nil {
		contentLength = *out.ContentLength
	}

	return out.Body, contentType, contentLength, nil
}

// Delete menghapus objek dari bucket Amazon S3 berdasarkan key.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("s3 delete object %s: %w", key, err)
	}

	return nil
}

// Bucket mengembalikan nama bucket S3 yang sedang digunakan.
func (s *S3Storage) Bucket() string {
	return s.bucket
}

// Region mengembalikan region AWS bucket S3.
func (s *S3Storage) Region() string {
	return s.region
}
