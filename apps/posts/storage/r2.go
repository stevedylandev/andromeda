package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2 struct {
	bucket    string
	publicURL string
	client    *s3.Client
}

func NewR2(accountID, accessKeyID, secretAccessKey, bucket, publicURL string) (*R2, error) {
	if strings.TrimSpace(accountID) == "" || strings.TrimSpace(accessKeyID) == "" || strings.TrimSpace(secretAccessKey) == "" || strings.TrimSpace(bucket) == "" || strings.TrimSpace(publicURL) == "" {
		return nil, fmt.Errorf("R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_BUCKET, and R2_PUBLIC_URL are required")
	}
	endpoint := "https://" + accountID + ".r2.cloudflarestorage.com"
	cfg := aws.Config{
		Region:      "auto",
		Credentials: credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	return &R2{bucket: bucket, publicURL: strings.TrimRight(publicURL, "/"), client: client}, nil
}

func (r *R2) Put(ctx context.Context, key, contentType string, data []byte) error {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	return err
}

func (r *R2) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(r.bucket), Key: aws.String(key)})
	return err
}

func (r *R2) PublicURL(key string) string {
	if r.publicURL == "" {
		return ""
	}
	return r.publicURL + "/" + url.PathEscape(key)
}

func (r *R2) Name() string { return "r2" }
