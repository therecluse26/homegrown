package adapters

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/homegrown-academy/homegrown-academy/internal/media"
)

// S3StorageAdapter implements ObjectStorageAdapter using AWS SDK v2.
// Provider-agnostic — works with S3, R2, MinIO, etc. via custom endpoint. [ARCH §2.10]
type S3StorageAdapter struct {
	client     *s3.Client
	presigner  *s3.PresignClient
	bucket     string
}

// Compile-time interface check.
var _ media.ObjectStorageAdapter = (*S3StorageAdapter)(nil)

// S3Config holds the configuration for the S3 storage adapter.
type S3Config struct {
	Endpoint        string // Custom endpoint URL (e.g., R2, MinIO). Empty = default AWS.
	Region          string // AWS region or "auto" for R2.
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

// NewS3StorageAdapter constructs an S3StorageAdapter.
func NewS3StorageAdapter(ctx context.Context, cfg S3Config) (*S3StorageAdapter, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)),
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	s3Opts := []func(*s3.Options){}
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // Required for MinIO and some R2 configurations
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)
	presigner := s3.NewPresignClient(client)

	return &S3StorageAdapter{
		client:    client,
		presigner: presigner,
		bucket:    cfg.Bucket,
	}, nil
}

func (a *S3StorageAdapter) PresignedPut(ctx context.Context, key string, _ uint64, contentType string, expiresSeconds uint32) (string, error) {
	req, err := a.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      &a.bucket,
		Key:         &key,
		ContentType: &contentType,
	}, s3.WithPresignExpires(time.Duration(expiresSeconds)*time.Second))
	if err != nil {
		return "", &media.StorageError{Code: "presign_failed", Message: fmt.Sprintf("presign PUT failed: %v", err)}
	}
	return req.URL, nil
}

func (a *S3StorageAdapter) PresignedGet(ctx context.Context, key string, expiresSeconds uint32) (string, error) {
	req, err := a.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &a.bucket,
		Key:    &key,
	}, s3.WithPresignExpires(time.Duration(expiresSeconds)*time.Second))
	if err != nil {
		return "", &media.StorageError{Code: "presign_failed", Message: fmt.Sprintf("presign GET failed: %v", err)}
	}
	return req.URL, nil
}

func (a *S3StorageAdapter) PutObject(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := a.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &a.bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	})
	if err != nil {
		return &media.StorageError{Code: "operation_failed", Message: fmt.Sprintf("PUT object failed: %v", err)}
	}
	return nil
}

func (a *S3StorageAdapter) GetObjectHead(ctx context.Context, key string) (*media.ObjectMetadata, error) {
	resp, err := a.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &a.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, &media.StorageError{Code: "not_found", Message: fmt.Sprintf("HEAD object failed: %v", err)}
	}
	var ct *string
	if resp.ContentType != nil {
		ct = resp.ContentType
	}
	return &media.ObjectMetadata{
		ContentLength: uint64(aws.ToInt64(resp.ContentLength)),
		ContentType:   ct,
	}, nil
}

func (a *S3StorageAdapter) GetObjectBytes(ctx context.Context, key string, start uint64, end uint64) ([]byte, error) {
	rangeStr := fmt.Sprintf("bytes=%d-%d", start, end-1)
	resp, err := a.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &a.bucket,
		Key:    &key,
		Range:  &rangeStr,
	})
	if err != nil {
		return nil, &media.StorageError{Code: "operation_failed", Message: fmt.Sprintf("GET object failed: %v", err)}
	}
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(resp.Body)
}

func (a *S3StorageAdapter) DeleteObject(ctx context.Context, key string) error {
	_, err := a.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &a.bucket,
		Key:    &key,
	})
	if err != nil {
		return &media.StorageError{Code: "operation_failed", Message: fmt.Sprintf("DELETE object failed: %v", err)}
	}
	return nil
}

func (a *S3StorageAdapter) DownloadToFile(ctx context.Context, key string, filepath string) error {
	resp, err := a.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &a.bucket,
		Key:    &key,
	})
	if err != nil {
		return &media.StorageError{Code: "operation_failed", Message: fmt.Sprintf("GET object for download failed: %v", err)}
	}
	defer func() { _ = resp.Body.Close() }()

	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("creating local file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return &media.StorageError{Code: "operation_failed", Message: fmt.Sprintf("downloading object to file: %v", err)}
	}
	return nil
}

func (a *S3StorageAdapter) UploadFromFile(ctx context.Context, key string, filepath string, contentType string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("opening local file: %w", err)
	}
	defer func() { _ = f.Close() }()

	_, err = a.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &a.bucket,
		Key:         &key,
		Body:        f,
		ContentType: &contentType,
	})
	if err != nil {
		return &media.StorageError{Code: "operation_failed", Message: fmt.Sprintf("PUT object from file failed: %v", err)}
	}
	return nil
}
