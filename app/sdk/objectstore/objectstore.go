// Package objectstore provides access to an S3-compatible object store.
package objectstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Config holds the information needed to connect to an object store.
type Config struct {
	Endpoint       string
	PublicEndpoint string
	Bucket         string
	Region         string
	AccessKey      string
	SecretKey      string
	UsePathStyle   bool
}

// Info represents metadata about an object.
type Info struct {
	Size int64
	ETag string
}

// Client manages access to an S3-compatible object store.
type Client struct {
	svc     *s3.Client
	presign *s3.PresignClient
	bucket  string
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

	publicEndpoint := cfg.PublicEndpoint
	if publicEndpoint == "" {
		publicEndpoint = cfg.Endpoint
	}

	internal := newS3Client(awsCfg, cfg.Endpoint, cfg.UsePathStyle)
	public := newS3Client(awsCfg, publicEndpoint, cfg.UsePathStyle)

	return &Client{
		svc:     internal,
		presign: s3.NewPresignClient(public),
		bucket:  cfg.Bucket,
	}, nil
}

func newS3Client(awsCfg aws.Config, endpoint string, pathStyle bool) *s3.Client {
	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = pathStyle
	})
}

// HeadObject retrieves metadata for an object.
func (c *Client) HeadObject(ctx context.Context, bucket, key string) (info Info, err error) {
	if bucket == "" {
		bucket = c.bucket
	}

	ctx, span := startSpan(ctx, "objectstore.head_object",
		attribute.String("object.bucket", bucket),
		attribute.String("object.key", key),
	)
	defer func() { endSpan(span, err) }()

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

	span.SetAttributes(attribute.Int64("object.size", size))
	return Info{Size: size, ETag: etag}, nil
}

// PutObject writes one object. The caller registers the artifact after this returns.
func (c *Client) PutObject(ctx context.Context, bucket, key, contentType string, body []byte) (err error) {
	if bucket == "" {
		bucket = c.bucket
	}

	ctx, span := startSpan(ctx, "objectstore.put_object",
		attribute.String("object.bucket", bucket),
		attribute.String("object.key", key),
		attribute.String("object.content_type", contentType),
		attribute.Int("object.size", len(body)),
	)
	defer func() { endSpan(span, err) }()

	_, err = c.svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put object bucket[%s] key[%s]: %w", bucket, key, err)
	}

	return nil
}

// Bucket returns the configured default bucket.
func (c *Client) Bucket() string {
	return c.bucket
}

// PresignGet signs a GetObject URL the browser can open. Signing does not fetch the object.
func (c *Client) PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (signed string, expiresAt time.Time, err error) {
	if bucket == "" {
		bucket = c.bucket
	}

	ctx, span := startSpan(ctx, "objectstore.presign_get",
		attribute.String("object.bucket", bucket),
		attribute.String("object.key", key),
		attribute.Int64("presign.ttl_seconds", int64(ttl.Seconds())),
	)
	defer func() { endSpan(span, err) }()

	out, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("presign get bucket[%s] key[%s]: %w", bucket, key, err)
	}

	if parsed, parseErr := url.Parse(out.URL); parseErr == nil {
		span.SetAttributes(attribute.String("server.address", parsed.Host))
	}

	return out.URL, time.Now().Add(ttl), nil
}

// DeleteObject removes one object. A missing object is success so a retry can finish.
func (c *Client) DeleteObject(ctx context.Context, bucket, key string) (err error) {
	if bucket == "" {
		bucket = c.bucket
	}

	ctx, span := startSpan(ctx, "objectstore.delete_object",
		attribute.String("object.bucket", bucket),
		attribute.String("object.key", key),
	)
	defer func() { endSpan(span, err) }()

	_, err = c.svc.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil && !missingObject(err) {
		return fmt.Errorf("delete object bucket[%s] key[%s]: %w", bucket, key, err)
	}

	return nil
}

func missingObject(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.ErrorCode() {
	case "NotFound", "NoSuchKey", "404":
		return true
	default:
		return false
	}
}

func startSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return trace.SpanFromContext(ctx).TracerProvider().Tracer("app.sdk.objectstore").Start(
		ctx,
		name,
		trace.WithAttributes(attrs...),
	)
}

func endSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}
