package truenas

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTLSConfig_default(t *testing.T) {
	cfg, err := buildTLSConfig(TLSOptions{})
	require.NoError(t, err)
	assert.False(t, cfg.InsecureSkipVerify)
	assert.Nil(t, cfg.RootCAs)
	assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
}

func TestBuildTLSConfig_insecure(t *testing.T) {
	cfg, err := buildTLSConfig(TLSOptions{InsecureSkipVerify: true})
	require.NoError(t, err)
	assert.True(t, cfg.InsecureSkipVerify)
}

func TestBuildTLSConfig_caFile_missing(t *testing.T) {
	_, err := buildTLSConfig(TLSOptions{CAFile: "/no/such/ca.pem"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/no/such/ca.pem")
}

func TestBuildTLSConfig_caFile_invalidPEM(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	require.NoError(t, os.WriteFile(caPath, []byte("not a pem"), 0o600))

	_, err := buildTLSConfig(TLSOptions{CAFile: caPath})
	require.Error(t, err)
	assert.Contains(t, err.Error(), caPath)
}

func TestBuildTLSConfig_caFile_trustsTestServer(t *testing.T) {
	caCert, serverCert := generateTestCAAndServerCert(t)
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	require.NoError(t, os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert.Raw,
	}), 0o600))

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}}
	server.StartTLS()
	t.Cleanup(server.Close)

	tlsCfg, err := buildTLSConfig(TLSOptions{CAFile: caPath})
	require.NoError(t, err)
	require.NotNil(t, tlsCfg.RootCAs)

	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
		Timeout:   5 * time.Second,
	}
	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func generateTestCAAndServerCert(t *testing.T) (*x509.Certificate, tls.Certificate) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caCert, err := x509.ParseCertificate(caCertDER)
	require.NoError(t, err)

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	require.NoError(t, err)

	serverLeaf, err := x509.ParseCertificate(serverCertDER)
	require.NoError(t, err)

	return caCert, tls.Certificate{
		Certificate: [][]byte{serverCertDER},
		PrivateKey:  serverKey,
		Leaf:        serverLeaf,
	}
}
