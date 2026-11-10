// Package objectstore provides access to an S3-compatible object store.
package objectstore

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config holds the information needed to connect to an object store.
type Config struct {
	Endpoint     string
	Bucket       string
	Region       string
	AccessKey    string
	SecretKey    string
	UsePathStyle bool
}

// Info represents metadata about an object.
type Info struct {
	Size int64
	ETag string
}

// Client manages access to an S3-compatible object store.
type Client struct {
	svc    *s3.Client
	bucket string
}

// New constructs a Client for use.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}
	if cfg.AccessKey != "" || cfg.SecretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading aws config: %w", err)
	}

	s3Opts := func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	}

	return &Client{
		svc:    s3.NewFromConfig(awsCfg, s3Opts),
		bucket: cfg.Bucket,
	}, nil
}

// HeadObject retrieves metadata for an object.
func (c *Client) HeadObject(ctx context.Context, bucket, key string) (Info, error) {
	if bucket == "" {
		bucket = c.bucket
	}

	out, err := c.svc.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return Info{}, fmt.Errorf("head object bucket[%s] key[%s]: %w", bucket, key, err)
	}

	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}

	var etag string
	if out.ETag != nil {
		etag = *out.ETag
	}

	return Info{Size: size, ETag: etag}, nil
}

// Bucket returns the configured default bucket.
func (c *Client) Bucket() string {
	return c.bucket
}
