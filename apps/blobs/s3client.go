package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Client struct {
	s3         *s3.Client
	presigner  *s3.PresignClient
	publicURLs map[string]string
	presignTTL time.Duration
}

type BucketInfo struct {
	Name         string
	CreationDate string
}

type ObjectInfo struct {
	Key          string
	Name         string
	Size         int64
	LastModified string
	ETag         string
}

type ObjectMeta struct {
	Key           string
	ContentType   string
	Size          int64
	LastModified  string
	ETag          string
}

func NewS3Client(endpoint, region, accessKey, secretKey string, publicURLs map[string]string, presignTTL time.Duration) (*S3Client, error) {
	if strings.TrimSpace(accessKey) == "" || strings.TrimSpace(secretKey) == "" {
		return nil, fmt.Errorf("S3 access key and secret key are required")
	}
	if strings.TrimSpace(endpoint) == "" {
		return nil, fmt.Errorf("S3 endpoint is required (set S3_ENDPOINT or R2_ACCOUNT_ID)")
	}
	if region == "" {
		region = "auto"
	}
	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	if presignTTL <= 0 {
		presignTTL = time.Hour
	}
	return &S3Client{
		s3:         client,
		presigner:  s3.NewPresignClient(client),
		publicURLs: publicURLs,
		presignTTL: presignTTL,
	}, nil
}

func (c *S3Client) ListBuckets(ctx context.Context) ([]BucketInfo, error) {
	out, err := c.s3.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	buckets := make([]BucketInfo, 0, len(out.Buckets))
	for _, b := range out.Buckets {
		bi := BucketInfo{Name: aws.ToString(b.Name)}
		if b.CreationDate != nil {
			bi.CreationDate = b.CreationDate.UTC().Format("2006-01-02 15:04:05")
		}
		buckets = append(buckets, bi)
	}
	return buckets, nil
}

func (c *S3Client) List(ctx context.Context, bucket, prefix string) ([]string, []ObjectInfo, error) {
	folders := []string{}
	files := []ObjectInfo{}
	var token *string
	for {
		out, err := c.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String(prefix),
			Delimiter:         aws.String("/"),
			ContinuationToken: token,
		})
		if err != nil {
			return nil, nil, err
		}
		for _, cp := range out.CommonPrefixes {
			folders = append(folders, aws.ToString(cp.Prefix))
		}
		for _, obj := range out.Contents {
			key := aws.ToString(obj.Key)
			if key == prefix {
				continue
			}
			name := strings.TrimPrefix(key, prefix)
			oi := ObjectInfo{
				Key:  key,
				Name: name,
				ETag: strings.Trim(aws.ToString(obj.ETag), `"`),
			}
			if obj.Size != nil {
				oi.Size = *obj.Size
			}
			if obj.LastModified != nil {
				oi.LastModified = obj.LastModified.UTC().Format("2006-01-02 15:04:05")
			}
			files = append(files, oi)
		}
		if out.IsTruncated == nil || !*out.IsTruncated {
			break
		}
		token = out.NextContinuationToken
	}
	return folders, files, nil
}

func (c *S3Client) Head(ctx context.Context, bucket, key string) (*ObjectMeta, error) {
	out, err := c.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	m := &ObjectMeta{
		Key:         key,
		ContentType: aws.ToString(out.ContentType),
		ETag:        strings.Trim(aws.ToString(out.ETag), `"`),
	}
	if out.ContentLength != nil {
		m.Size = *out.ContentLength
	}
	if out.LastModified != nil {
		m.LastModified = out.LastModified.UTC().Format("2006-01-02 15:04:05")
	}
	return m, nil
}

func (c *S3Client) Get(ctx context.Context, bucket, key string) (io.ReadCloser, *ObjectMeta, error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, nil, err
	}
	m := &ObjectMeta{
		Key:         key,
		ContentType: aws.ToString(out.ContentType),
		ETag:        strings.Trim(aws.ToString(out.ETag), `"`),
	}
	if out.ContentLength != nil {
		m.Size = *out.ContentLength
	}
	if out.LastModified != nil {
		m.LastModified = out.LastModified.UTC().Format("2006-01-02 15:04:05")
	}
	return out.Body, m, nil
}

func (c *S3Client) Put(ctx context.Context, bucket, key, contentType string, body io.Reader, size int64) error {
	in := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	if size > 0 {
		in.ContentLength = aws.Int64(size)
	}
	_, err := c.s3.PutObject(ctx, in)
	return err
}

func (c *S3Client) Delete(ctx context.Context, bucket, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func (c *S3Client) PresignGet(ctx context.Context, bucket, key string) (string, error) {
	out, err := c.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(c.presignTTL))
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

func (c *S3Client) PublicURL(bucket, key string) (string, bool) {
	prefix, ok := c.publicURLs[bucket]
	if !ok || prefix == "" {
		return "", false
	}
	return strings.TrimRight(prefix, "/") + "/" + url.PathEscape(key), true
}

var _ s3types.Object
