package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/dreamoutbox/delivery-app/internal/config"
)

type StorageService struct {
	Client        *s3.Client
	PublicBucket  string
	PrivateBucket string
	PublicURL     string
}

func InitStorage(cfg *config.Config) (*StorageService, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               fmt.Sprintf("http://%s", cfg.MinIOEndpoint),
			SigningRegion:     "us-east-1",
			Source:            aws.EndpointSourceCustom,
			HostnameImmutable: true,
		}, nil
	})

	awsCfg, err := s3config.LoadDefaultConfig(context.TODO(),
		s3config.WithRegion("us-east-1"),
		s3config.WithEndpointResolverWithOptions(customResolver),
		s3config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.MinIOAccessKey, cfg.MinIOSecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	svc := &StorageService{
		Client:        client,
		PublicBucket:  cfg.MinIOPublicBucket,
		PrivateBucket: cfg.MinIOPrivateBucket,
		PublicURL:     cfg.MinIOPublicURL,
	}

	// Ensure buckets exist
	if err := svc.ensureBucket(cfg.MinIOPublicBucket, true); err != nil {
		log.Printf("Warning: failed to ensure public bucket %s: %v", cfg.MinIOPublicBucket, err)
	}
	if err := svc.ensureBucket(cfg.MinIOPrivateBucket, false); err != nil {
		log.Printf("Warning: failed to ensure private bucket %s: %v", cfg.MinIOPrivateBucket, err)
	}

	return svc, nil
}

func (s *StorageService) ensureBucket(bucketName string, isPublic bool) error {
	ctx := context.Background()
	_, err := s.Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		log.Printf("Creating bucket: %s", bucketName)
		_, err = s.Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return fmt.Errorf("could not create bucket %s: %w", bucketName, err)
		}
	}

	if isPublic {
		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {"AWS": ["*"]},
					"Action": ["s3:GetObject"],
					"Resource": ["arn:aws:s3:::%s/*"]
				}
			]
		}`, bucketName)

		_, _ = s.Client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
			Bucket: aws.String(bucketName),
			Policy: aws.String(policy),
		})
	}

	return nil
}

func (s *StorageService) UploadPublicFile(fileReader io.Reader, filename string, contentType string) (string, error) {
	objectKey := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(filename))

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, fileReader); err != nil {
		return "", err
	}

	if contentType == "" {
		contentType = http.DetectContentType(buf.Bytes())
	}

	_, err := s.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.PublicBucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload public object: %w", err)
	}

	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(s.PublicURL, "/"), objectKey)
	return url, nil
}

func (s *StorageService) UploadPrivateFile(fileReader io.Reader, filename string, contentType string) (string, error) {
	objectKey := fmt.Sprintf("proofs/%d_%s", time.Now().UnixNano(), filepath.Base(filename))

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, fileReader); err != nil {
		return "", err
	}

	if contentType == "" {
		contentType = http.DetectContentType(buf.Bytes())
	}

	_, err := s.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.PrivateBucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload private object: %w", err)
	}

	return objectKey, nil
}

func (s *StorageService) GetPrivateFile(objectKey string) (io.ReadCloser, string, error) {
	out, err := s.Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.PrivateBucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get private object: %w", err)
	}

	contentType := "image/jpeg"
	if out.ContentType != nil {
		contentType = *out.ContentType
	}
	return out.Body, contentType, nil
}
