package truenas

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_requiresURL(t *testing.T) {
	_, err := NewClient(Config{Username: "u", Password: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "URL is required")
}

func TestNewClient_requiresCredentials(t *testing.T) {
	_, err := NewClient(Config{URL: "https://example.com", Password: "p"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "username")

	_, err = NewClient(Config{URL: "https://example.com", Username: "u"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "password")
}

func TestNewClient_invalidCAFile(t *testing.T) {
	_, err := NewClient(Config{
		URL:      "https://example.com",
		Username: "u",
		Password: "p",
		CAFile:   "/no/such/ca.pem",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to configure TLS")
}

func TestNewClient_defaultTLSIsSecure(t *testing.T) {
	c, err := NewClient(Config{
		URL:      "https://example.com",
		Username: "u",
		Password: "p",
	})
	require.NoError(t, err)

	cl := c.(*client)
	transport, ok := cl.httpClient.GetClient().Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)
	assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestNewClient_insecureTLS(t *testing.T) {
	c, err := NewClient(Config{
		URL:      "https://example.com",
		Username: "u",
		Password: "p",
		Insecure: true,
	})
	require.NoError(t, err)

	cl := c.(*client)
	transport, ok := cl.httpClient.GetClient().Transport.(*http.Transport)
	require.True(t, ok)
	assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestNewClient_testConnection_withCAFile(t *testing.T) {
	caCert, serverCert := generateTestCAAndServerCert(t)
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	writeCACertPEM(t, caPath, caCert)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "test"})
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}}
	server.StartTLS()
	t.Cleanup(server.Close)

	client, err := NewClient(Config{
		URL:      server.URL,
		Username: "u",
		Password: "p",
		CAFile:   caPath,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, client.TestConnection(ctx))
}

func TestNewClient_testConnection_secureDefaultRejectsUntrustedCert(t *testing.T) {
	_, serverCert := generateTestCAAndServerCert(t)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}}
	server.StartTLS()
	t.Cleanup(server.Close)

	client, err := NewClient(Config{
		URL:      server.URL,
		Username: "u",
		Password: "p",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.TestConnection(ctx)
	require.Error(t, err)
}

func TestNewClient_testConnection_insecureAcceptsUntrustedCert(t *testing.T) {
	_, serverCert := generateTestCAAndServerCert(t)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": "test"})
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}}
	server.StartTLS()
	t.Cleanup(server.Close)

	client, err := NewClient(Config{
		URL:      server.URL,
		Username: "u",
		Password: "p",
		Insecure: true,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, client.TestConnection(ctx))
}

func writeCACertPEM(t *testing.T, path string, caCert *x509.Certificate) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert.Raw,
	}), 0o600))
}
