package runtime

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

type CertificateStatus struct {
	TrustMode       string    `json:"trust_mode"`
	CertFile        string    `json:"cert_file,omitempty"`
	KeyFile         string    `json:"key_file,omitempty"`
	CAFile          string    `json:"ca_file,omitempty"`
	OK              bool      `json:"ok"`
	Mode            string    `json:"mode"`
	Subject         string    `json:"subject,omitempty"`
	Issuer          string    `json:"issuer,omitempty"`
	DNSNames        []string  `json:"dns_names,omitempty"`
	IPAddresses     []string  `json:"ip_addresses,omitempty"`
	NotBefore       time.Time `json:"not_before,omitempty"`
	NotAfter        time.Time `json:"not_after,omitempty"`
	CAReadable      bool      `json:"ca_readable,omitempty"`
	Error           string    `json:"error,omitempty"`
	OpenSSLHint     string    `json:"openssl_hint,omitempty"`
}

func BuildCertificateStatus(cfg *Config) CertificateStatus {
	st := CertificateStatus{TrustMode: cfg.Security.TrustMode, CertFile: cfg.Security.CertFile, KeyFile: cfg.Security.KeyFile, CAFile: cfg.Security.CAFile, Mode: cfg.Security.TrustMode}
	if cfg.Security.TrustMode == "plaintext_lab" || cfg.Security.TrustMode == "" {
		st.OK = true
		st.Mode = "plaintext_lab"
		return st
	}
	cert, err := tls.LoadX509KeyPair(cfg.Security.CertFile, cfg.Security.KeyFile)
	if err != nil {
		st.Error = fmt.Sprintf("cert/key load failed: %v", err)
		return st
	}
	if len(cert.Certificate) == 0 {
		st.Error = "cert/key load returned no certificates"
		return st
	}
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		st.Error = fmt.Sprintf("certificate parse failed: %v", err)
		return st
	}
	st.Subject = parsed.Subject.String()
	st.Issuer = parsed.Issuer.String()
	st.DNSNames = append([]string(nil), parsed.DNSNames...)
	for _, ip := range parsed.IPAddresses {
		st.IPAddresses = append(st.IPAddresses, ip.String())
	}
	st.NotBefore = parsed.NotBefore
	st.NotAfter = parsed.NotAfter
	if cfg.Security.CAFile != "" {
		caBytes, err := os.ReadFile(cfg.Security.CAFile)
		if err != nil {
			st.Error = fmt.Sprintf("ca_file unreadable: %v", err)
			return st
		}
		block, _ := pem.Decode(caBytes)
		if block == nil {
			st.Error = "ca_file does not contain PEM data"
			return st
		}
		st.CAReadable = true
	}
	if cfg.Security.TrustMode == "strict_mtls" || cfg.Security.TrustMode == "mtls_no_revocation" {
		if cfg.Security.CAFile == "" {
			st.Error = "ca_file is required for mutual TLS"
			return st
		}
	}
	st.OK = true
	st.OpenSSLHint = fmt.Sprintf("openssl x509 -in %s -noout -subject -issuer -dates -ext subjectAltName", cfg.Security.CertFile)
	return st
}
