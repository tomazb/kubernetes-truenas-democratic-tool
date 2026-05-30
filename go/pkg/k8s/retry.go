package k8s

import (
	"context"
	"errors"
	"net"
	"os"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// isTransientK8sError reports whether a Kubernetes API error is worth retrying.
func isTransientK8sError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	switch {
	case apierrors.IsTimeout(err),
		apierrors.IsServerTimeout(err),
		apierrors.IsTooManyRequests(err),
		apierrors.IsServiceUnavailable(err),
		apierrors.IsInternalError(err),
		apierrors.IsUnexpectedServerError(err):
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	return false
}
