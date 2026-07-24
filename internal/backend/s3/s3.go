// Package s3 implements Backend using AWS SDK v2 against S3-compatible APIs (R2).
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// Config for an S3-compatible endpoint.
type Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	Prefix    string
	AccessKey string
	SecretKey string
	// HTTPClient optional for tests (fake server).
	HTTPClient *http.Client
}

// Client wraps aws s3 client.
type Client struct {
	api    *s3.Client
	bucket string
	prefix string
}

// New creates a Client.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket required")
	}
	if cfg.Region == "" {
		cfg.Region = "auto"
	}
	var opts []func(*config.LoadOptions) error
	opts = append(opts, config.WithRegion(cfg.Region))
	if cfg.AccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")))
	}
	if cfg.HTTPClient != nil {
		opts = append(opts, config.WithHTTPClient(cfg.HTTPClient))
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	})
	return &Client{api: client, bucket: cfg.Bucket, prefix: strings.Trim(cfg.Prefix, "/")}, nil
}

func (c *Client) key(k string) string {
	k = strings.TrimPrefix(k, "/")
	if c.prefix == "" {
		return k
	}
	return c.prefix + "/" + k
}

func (c *Client) Put(ctx context.Context, key string, r io.Reader, size int64, opts backend.PutOptions) (backend.ObjectMeta, error) {
	in := &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(c.key(key)),
		Body:   r,
	}
	if size > 0 {
		in.ContentLength = aws.Int64(size)
	}
	if opts.ContentType != "" {
		in.ContentType = aws.String(opts.ContentType)
	}
	if opts.IfMatch != "" {
		in.IfMatch = aws.String(opts.IfMatch)
	}
	if opts.IfNoneMatch {
		in.IfNoneMatch = aws.String("*")
	}
	out, err := c.api.PutObject(ctx, in)
	if err != nil {
		return backend.ObjectMeta{}, mapErr(err)
	}
	et := ""
	if out.ETag != nil {
		et = strings.Trim(*out.ETag, `"`)
	}
	return backend.ObjectMeta{Key: key, ETag: et, Size: size}, nil
}

func (c *Client) Get(ctx context.Context, key string) (io.ReadCloser, backend.ObjectMeta, error) {
	out, err := c.api.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(c.key(key)),
	})
	if err != nil {
		return nil, backend.ObjectMeta{}, mapErr(err)
	}
	et := ""
	if out.ETag != nil {
		et = strings.Trim(*out.ETag, `"`)
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return out.Body, backend.ObjectMeta{Key: key, ETag: et, Size: size}, nil
}

func (c *Client) Head(ctx context.Context, key string) (backend.ObjectMeta, error) {
	out, err := c.api.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(c.key(key)),
	})
	if err != nil {
		return backend.ObjectMeta{}, mapErr(err)
	}
	et := ""
	if out.ETag != nil {
		et = strings.Trim(*out.ETag, `"`)
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return backend.ObjectMeta{Key: key, ETag: et, Size: size}, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.api.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(c.key(key)),
	})
	return mapErr(err)
}

func (c *Client) List(ctx context.Context, prefix string) ([]backend.ObjectMeta, error) {
	full := c.key(prefix)
	out, err := c.api.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(full),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	var res []backend.ObjectMeta
	for _, o := range out.Contents {
		k := aws.ToString(o.Key)
		if c.prefix != "" {
			k = strings.TrimPrefix(k, c.prefix+"/")
		}
		et := strings.Trim(aws.ToString(o.ETag), `"`)
		var size int64
		if o.Size != nil {
			size = *o.Size
		}
		res = append(res, backend.ObjectMeta{Key: k, ETag: et, Size: size})
	}
	return res, nil
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return backend.ErrNotFound
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchKey", "404":
			return backend.ErrNotFound
		case "PreconditionFailed", "412":
			return backend.ErrPrecondition
		case "AccessDenied", "InvalidAccessKeyId", "SignatureDoesNotMatch":
			return backend.ErrUnauthorized
		}
	}
	// HeadObject often returns 404 as http status without typed error on all backends
	if strings.Contains(err.Error(), "StatusCode: 404") || strings.Contains(err.Error(), "NotFound") {
		return backend.ErrNotFound
	}
	if strings.Contains(err.Error(), "PreconditionFailed") {
		return backend.ErrPrecondition
	}
	return err
}
