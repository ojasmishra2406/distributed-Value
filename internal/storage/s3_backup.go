package storage

import (
	"context"
	"compress/gzip"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3BackupManager struct {
	client *s3.Client
	bucket string
}

func NewS3BackupManager(ctx context.Context, bucket string) (*S3BackupManager, error) {
	cfg, err := config.LoadDefaultConfig(ctx) // Authenticates via IRSA automatically in EKS
	if err != nil {
		return nil, err
	}
	return &S3BackupManager{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
	}, nil
}

func (b *S3BackupManager) UploadSSTableCompressed(ctx context.Context, localPath, s3Key string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}

	pr, pw := io.Pipe()
	go func() {
		defer file.Close()
		defer pw.Close()
		
		gw := gzip.NewWriter(pw)
		defer gw.Close()

		_, err := io.Copy(gw, file)
		if err != nil {
			pw.CloseWithError(err)
		}
	}()

	_, err = b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(b.bucket),
		Key:         aws.String(s3Key + ".gz"),
		Body:        pr,
		ContentType: aws.String("application/gzip"),
	})

	return err
}
