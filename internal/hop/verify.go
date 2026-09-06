package hop

import (
	"context"
	"errors"
	"net/http"
)

// Reference is the control plane's reference locker: a bucket the operator
// owns that an account's credentials must be refused from. rein sync verify
// lists and reads it to show isolation.
type Reference struct {
	Endpoint string `json:"endpoint"`
	Bucket   string `json:"bucket"`
	Region   string `json:"region"`
	Key      string `json:"key"`
}

// ErrNoReference reports a control plane without a reference locker; the
// isolation check cannot run against it.
var ErrNoReference = errors.New("this control plane has no reference locker; the isolation check cannot run here")

// VerifyReference asks where the reference locker's probe object lives.
func (c *Client) VerifyReference(ctx context.Context, token string) (Reference, error) {
	var out Reference
	err := c.do(ctx, http.MethodGet, "/v1/verify/reference", token, nil, &out)
	var he *Error
	if errors.As(err, &he) && he.Code == "no_reference" {
		return Reference{}, ErrNoReference
	}
	if err != nil {
		return Reference{}, lockerError(err)
	}
	if out.Endpoint == "" || out.Bucket == "" || out.Key == "" {
		return Reference{}, errors.New("control plane returned an incomplete reference locker")
	}
	return out, nil
}

// VerifyReportReceipt acknowledges a stored verification report.
type VerifyReportReceipt struct {
	ID         int64  `json:"id"`
	ReceivedAt string `json:"received_at"`
}

// PostVerifyReport stores a verification report for this device. report
// is marshalled as sent; callers pass the upload form that carries step
// results only (see internal/verify).
func (c *Client) PostVerifyReport(ctx context.Context, token string, report any) (VerifyReportReceipt, error) {
	var out VerifyReportReceipt
	if err := c.do(ctx, http.MethodPost, "/v1/verify-reports", token, report, &out); err != nil {
		return VerifyReportReceipt{}, lockerError(err)
	}
	return out, nil
}
