package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"sync"
	"time"
)

// CertCache generates and caches leaf TLS certificates signed by a CA.
type CertCache struct {
	ca    *CA
	mu    sync.Mutex
	cache map[string]*tls.Certificate
}

// NewCertCache creates a CertCache backed by the given CA.
func NewCertCache(ca *CA) *CertCache {
	return &CertCache{ca: ca, cache: make(map[string]*tls.Certificate)}
}

// GetCertificate is a tls.Config.GetCertificate callback.
// It returns a cached cert for the SNI hostname, generating one if absent.
func (c *CertCache) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	host := hello.ServerName
	if host == "" {
		host = "localhost"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if cert, ok := c.cache[host]; ok {
		return cert, nil
	}
	cert, err := c.generate([]string{host})
	if err != nil {
		return nil, err
	}
	c.cache[host] = cert
	return cert, nil
}

// Generate returns a fresh tls.Certificate for the given SANs signed by the CA.
func (c *CertCache) Generate(hosts []string) (*tls.Certificate, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generate(hosts)
}

func (c *CertCache) generate(hosts []string) (*tls.Certificate, error) {
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
			CommonName:   hosts[0],
			Organization: []string{"Valla CLI"},
		},
		DNSNames:    hosts,
		NotBefore:   time.Now().Add(-10 * time.Minute),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, c.ca.Cert, &key.PublicKey, c.ca.Key)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tlsCert, nil
}
