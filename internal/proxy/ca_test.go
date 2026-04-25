package proxy

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// withTempHome redirects ~/.valla writes to a temp directory for test isolation.
func withTempHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	return filepath.Join(tmp, ".valla")
}

func TestGenerateCA_Fields(t *testing.T) {
	ca, err := generateCA()
	if err != nil {
		t.Fatalf("generateCA: %v", err)
	}
	if !ca.Cert.IsCA {
		t.Error("generated certificate should be a CA")
	}
	if ca.Cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Error("CA must have KeyUsageCertSign")
	}
	if time.Until(ca.Cert.NotAfter) < 9*365*24*time.Hour {
		t.Error("CA expiry should be at least 9 years in the future")
	}
	if ca.Cert.Subject.CommonName != "Valla Local CA" {
		t.Errorf("unexpected CN: %s", ca.Cert.Subject.CommonName)
	}
	if ca.Key == nil {
		t.Error("CA key must not be nil")
	}
	if len(ca.CertPEM) == 0 || len(ca.KeyPEM) == 0 {
		t.Error("CA PEM fields must not be empty")
	}
}

func TestLoadOrCreateCA_CreatesFiles(t *testing.T) {
	vallaDir := withTempHome(t)

	ca, created, err := LoadOrCreateCA()
	if err != nil {
		t.Fatalf("LoadOrCreateCA: %v", err)
	}
	if !created {
		t.Error("expected created=true on first call")
	}
	if ca == nil {
		t.Fatal("CA must not be nil")
	}

	certPath := filepath.Join(vallaDir, caCertFile)
	keyPath := filepath.Join(vallaDir, caKeyFile)
	if _, err := os.Stat(certPath); err != nil {
		t.Errorf("ca.pem not created: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("ca-key.pem not created: %v", err)
	}

	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o044 != 0 {
		t.Errorf("ca-key.pem should not be group/world readable, got %v", info.Mode().Perm())
	}
}

func TestLoadOrCreateCA_Idempotent(t *testing.T) {
	withTempHome(t)

	ca1, created1, err := LoadOrCreateCA()
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if !created1 {
		t.Error("expected created=true on first call")
	}

	ca2, created2, err := LoadOrCreateCA()
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if created2 {
		t.Error("expected created=false on second call (idempotent)")
	}
	if string(ca1.CertPEM) != string(ca2.CertPEM) {
		t.Error("second call should return the same certificate")
	}
}

func TestParseCAPEM_RoundTrip(t *testing.T) {
	ca, err := generateCA()
	if err != nil {
		t.Fatal(err)
	}
	ca2, err := parseCAPEM(ca.CertPEM, ca.KeyPEM)
	if err != nil {
		t.Fatalf("parseCAPEM: %v", err)
	}
	if ca2.Cert.SerialNumber.Cmp(ca.Cert.SerialNumber) != 0 {
		t.Error("serial numbers should match after round-trip")
	}
}

func TestParseCAPEM_RejectsWrongBlockType(t *testing.T) {
	ca, err := generateCA()
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(ca.CertPEM)
	fakePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: block.Bytes})
	_, err = parseCAPEM(fakePEM, ca.KeyPEM)
	if err == nil {
		t.Error("parseCAPEM should reject PEM with wrong block type")
	}
}
