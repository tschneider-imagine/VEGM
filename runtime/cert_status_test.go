package runtime

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildCertificateStatusPlaintextLab(t *testing.T) {
	cfg := &Config{Security: SecurityConfig{TrustMode: "plaintext_lab"}}
	st := BuildCertificateStatus(cfg)
	if !st.OK {
		t.Fatalf("expected plaintext status ok, got error=%q", st.Error)
	}
	if st.Mode != "plaintext_lab" {
		t.Fatalf("expected plaintext_lab mode, got %q", st.Mode)
	}
}

func TestBuildCertificateStatusMissingFiles(t *testing.T) {
	cfg := &Config{Security: SecurityConfig{TrustMode: "tls_server_only", CertFile: "missing.pem", KeyFile: "missing.key"}}
	st := BuildCertificateStatus(cfg)
	if st.OK {
		t.Fatalf("expected missing cert/key to fail")
	}
	if st.Error == "" {
		t.Fatalf("expected error text")
	}
}

func TestBuildCertificateStatusValidSelfSigned(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.pem")
	keyPath := filepath.Join(dir, "server.key")
	writeSelfSignedCert(t, certPath, keyPath)

	cfg := &Config{Security: SecurityConfig{TrustMode: "tls_server_only", CertFile: certPath, KeyFile: keyPath}}
	st := BuildCertificateStatus(cfg)
	if !st.OK {
		t.Fatalf("expected valid cert status ok, got error=%q", st.Error)
	}
	if st.Subject == "" || st.Issuer == "" {
		t.Fatalf("expected subject and issuer, got subject=%q issuer=%q", st.Subject, st.Issuer)
	}
	if len(st.DNSNames) != 1 || st.DNSNames[0] != "vegm-test.local" {
		t.Fatalf("expected DNS SAN vegm-test.local, got %#v", st.DNSNames)
	}
	if st.OpenSSLHint == "" {
		t.Fatalf("expected OpenSSL hint")
	}
}

func writeSelfSignedCert(t *testing.T, certPath, keyPath string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "vegm-test.local"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"vegm-test.local"},
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	certOut, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("create cert file: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	_ = certOut.Close()
	keyOut, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}); err != nil {
		t.Fatalf("write key: %v", err)
	}
	_ = keyOut.Close()
}
