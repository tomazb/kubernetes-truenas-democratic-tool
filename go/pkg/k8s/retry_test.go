package k8s

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestIsTransientK8sError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"not found", apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "x"), false},
		{"forbidden", apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "x", errors.New("denied")), false},
		{"timeout", apierrors.NewTimeoutError("slow", 1), true},
		{"too many requests", apierrors.NewTooManyRequests("slow down", 1), true},
		{"service unavailable", apierrors.NewServiceUnavailable("unavailable"), true},
		{"internal error", apierrors.NewInternalError(errors.New("boom")), true},
		{"context canceled", context.Canceled, false},
		{"generic error", fmt.Errorf("validation failed"), false},
		{
			name: "timeout net error",
			err: &net.DNSError{
				Err:       "timeout",
				IsTimeout: true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTransientK8sError(tt.err)
			if got != tt.want {
				t.Fatalf("isTransientK8sError() = %v, want %v", got, tt.want)
			}
		})
	}
}
