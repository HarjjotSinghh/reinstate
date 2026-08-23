// Package s3 implements Backend using AWS SDK v2 against S3-compatible APIs (R2).
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
)

// Config for an S3-compatible endpoint.
type Config struct {
	Endpoint string
	Region   string
	Bucket   string
	Prefix   string
	// AccessKey and SecretKey are static BYO keys. They are equivalent to
	// Credentials: Static(AccessKey, SecretKey) and ignored when Credentials
	// is set.
	AccessKey string
	SecretKey string
	// Credentials is an optional source of possibly expiring keys (for example
	// hourly locker credentials). When nil and AccessKey is empty, the AWS SDK
	// default chain is used, exactly as before.
	Credentials CredentialSource
	// HTTPClient optional for tests (fake server).
	HTTPClient *http.Client
}

// Client wraps aws s3 client.
type Client struct {
	api    *s3.Client
	bucket string
	prefix string
	// creds is the SDK cache in front of Config.Credentials; nil when the SDK
	// default chain is in use.
	creds *aws.CredentialsCache
	// refreshable is true when a rejected credential may be replaced by asking
	// the source again. Static keys are never retried.
	refreshable bool
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
	source := cfg.Credentials
	if source == nil && cfg.AccessKey != "" {
		source = Static(cfg.AccessKey, cfg.SecretKey)
	}
	var cache *aws.CredentialsCache
	refreshable := false
	if source != nil {
		_, static := source.(StaticSource)
		refreshable = !static
		cache = aws.NewCredentialsCache(sourceProvider{source: source}, func(o *aws.CredentialsCacheOptions) {
			o.ExpiryWindow = refreshExpiryWindow
		})
		opts = append(opts, config.WithCredentialsProvider(cache))
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
	return &Client{
		api: client, bucket: cfg.Bucket, prefix: strings.Trim(cfg.Prefix, "/"),
		creds: cache, refreshable: refreshable,
	}, nil
}

// withCredentialRetry runs op once; if the endpoint rejected the credential
// and the source can refresh, it drops the cached credential and runs op a
// second time. rewind, when non-nil, resets the request body before the retry
// and its error cancels the retry, so a body that cannot be rewound is never
// resent.
func (c *Client) withCredentialRetry(rewind func() error, op func() error) error {
	err := op()
	if err == nil || !c.refreshable || !credentialRejected(err) {
		return err
	}
	if rewind != nil {
		if rerr := rewind(); rerr != nil {
			return err
		}
	}
	c.creds.Invalidate()
	return op()
}

var errBodyNotRewindable = errors.New("s3: request body cannot be rewound")

// bodyRewinder returns a function that seeks r back to its current offset.
// For a non-seekable r the returned function fails, which disables the retry.
func bodyRewinder(r io.Reader) func() error {
	seeker, ok := r.(io.Seeker)
	if !ok {
		return func() error { return errBodyNotRewindable }
	}
	start, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return func() error { return errBodyNotRewindable }
	}
	return func() error {
		_, err := seeker.Seek(start, io.SeekStart)
		return err
	}
}

// credentialRejected reports whether err means the storage endpoint refused
// the credential itself (expired, revoked, unknown, or badly signed), as
// opposed to a missing object or a failed precondition.
func credentialRejected(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.ErrorCode() {
	case "ExpiredToken", "ExpiredTokenException", "InvalidToken", "TokenRefreshRequired",
		"AccessDenied", "InvalidAccessKeyId", "SignatureDoesNotMatch":
		return true
	case "Forbidden":
		// HEAD responses carry no XML body, so the SDK only sees the status.
		return true
	}
	return false
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
	var out *s3.PutObjectOutput
	err := c.withCredentialRetry(bodyRewinder(r), func() (err error) {
		out, err = c.api.PutObject(ctx, in)
		return err
	})
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
	var out *s3.GetObjectOutput
	err := c.withCredentialRetry(nil, func() (err error) {
		out, err = c.api.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(c.key(key)),
		})
		return err
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
	var out *s3.HeadObjectOutput
	err := c.withCredentialRetry(nil, func() (err error) {
		out, err = c.api.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(c.key(key)),
		})
		return err
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
	err := c.withCredentialRetry(nil, func() error {
		_, err := c.api.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(c.key(key)),
		})
		return err
	})
	return mapErr(err)
}

func (c *Client) List(ctx context.Context, prefix string) ([]backend.ObjectMeta, error) {
	full := c.key(prefix)
	var out *s3.ListObjectsV2Output
	err := c.withCredentialRetry(nil, func() (err error) {
		out, err = c.api.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(c.bucket),
			Prefix: aws.String(full),
		})
		return err
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
		case "AccessDenied", "InvalidAccessKeyId", "SignatureDoesNotMatch", "Forbidden",
			"ExpiredToken", "ExpiredTokenException", "InvalidToken", "TokenRefreshRequired":
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
