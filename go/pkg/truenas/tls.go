package truenas

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// TLSOptions configures outbound TLS to TrueNAS.
type TLSOptions struct {
	InsecureSkipVerify bool
	CAFile             string
}

func buildTLSConfig(opts TLSOptions) (*tls.Config, error) {
	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: opts.InsecureSkipVerify,
	}

	if opts.CAFile == "" {
		return tlsCfg, nil
	}

	pemData, err := os.ReadFile(opts.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read truenas CA file %q: %w", opts.CAFile, err)
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}

	if !pool.AppendCertsFromPEM(pemData) {
		return nil, fmt.Errorf("failed to parse certificates from truenas CA file %q", opts.CAFile)
	}

	tlsCfg.RootCAs = pool
	return tlsCfg, nil
}
