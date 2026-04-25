// Package proxy implements the Secure Serve reverse proxy and its supporting CA lifecycle.
package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const (
	caCertFile = "ca.pem"
	caKeyFile  = "ca-key.pem"
)

// CA holds a loaded or freshly-generated Root Certificate Authority.
type CA struct {
	Cert    *x509.Certificate
	CertPEM []byte
	Key     *ecdsa.PrivateKey
	KeyPEM  []byte
}

// VallaDir returns the path to the ~/.valla directory.
func VallaDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".valla"), nil
}

// LoadOrCreateCA loads the CA from ~/.valla/ or generates a new one if absent.
// Returns the CA and a boolean indicating whether it was newly created.
func LoadOrCreateCA() (*CA, bool, error) {
	dir, err := VallaDir()
	if err != nil {
		return nil, false, err
	}
	certPath := filepath.Join(dir, caCertFile)
	keyPath := filepath.Join(dir, caKeyFile)

	certPEM, certErr := os.ReadFile(certPath)
	keyPEM, keyErr := os.ReadFile(keyPath)
	if certErr == nil && keyErr == nil {
		// Reject key files with permissions wider than 0600 — anyone who can
		// read ca-key.pem can sign arbitrary trusted certificates.
		if info, err := os.Stat(keyPath); err == nil && info.Mode().Perm()&0o077 != 0 {
			return nil, false, fmt.Errorf(
				"CA key %s has unsafe permissions %v; run: chmod 600 %s",
				keyPath, info.Mode().Perm(), keyPath,
			)
		}
		ca, err := parseCAPEM(certPEM, keyPEM)
		if err == nil {
			return ca, false, nil
		}
		// Corrupt files - fall through to regenerate.
	}

	ca, err := generateCA()
	if err != nil {
		return nil, false, err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, false, err
	}

	// Write both files to temporaries first, then rename atomically so that a
	// crash or disk-full between the two writes can never leave a cert on disk
	// without its matching key (which would silently invalidate the trust store).
	certTmp := certPath + ".tmp"
	keyTmp := keyPath + ".tmp"
	if err := os.WriteFile(certTmp, ca.CertPEM, 0o644); err != nil {
		return nil, false, err
	}
	if err := os.WriteFile(keyTmp, ca.KeyPEM, 0o600); err != nil {
		_ = os.Remove(certTmp)
		return nil, false, err
	}
	if err := os.Rename(certTmp, certPath); err != nil {
		_ = os.Remove(certTmp)
		_ = os.Remove(keyTmp)
		return nil, false, err
	}
	if err := os.Rename(keyTmp, keyPath); err != nil {
		_ = os.Remove(keyTmp)
		return nil, false, err
	}
	return ca, true, nil
}

// CertPath returns the on-disk path to the CA certificate.
func CertPath() (string, error) {
	dir, err := VallaDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, caCertFile), nil
}

// generateCA creates a new self-signed Root CA using ECDSA P-256.
func generateCA() (*CA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, err := randSerial()
	if err != nil {
		return nil, err
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "Valla Local CA",
			Organization: []string{"Valla CLI"},
		},
		NotBefore:             time.Now().Add(-10 * time.Minute),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return &CA{Cert: cert, CertPEM: certPEM, Key: key, KeyPEM: keyPEM}, nil
}

// parseCAPEM reconstructs a CA from PEM-encoded certificate and private key bytes.
func parseCAPEM(certPEM, keyPEM []byte) (*CA, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("failed to decode CA certificate PEM")
	}
	if block.Type != "CERTIFICATE" {
		return nil, errors.New("unexpected PEM block type: " + block.Type)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	if !cert.IsCA {
		return nil, errors.New("loaded certificate is not a CA")
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, errors.New("failed to decode CA key PEM")
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return &CA{Cert: cert, CertPEM: certPEM, Key: key, KeyPEM: keyPEM}, nil
}

func randSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, limit)
}
