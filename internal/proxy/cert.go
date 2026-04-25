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
	mu    sync.RWMutex
	cache map[string]*tls.Certificate
}

// NewCertCache creates a CertCache backed by the given CA.
func NewCertCache(ca *CA) *CertCache {
	return &CertCache{ca: ca, cache: make(map[string]*tls.Certificate)}
}

// GetCertificate is a tls.Config.GetCertificate callback.
// It returns a cached cert for the SNI hostname, generating one if absent.
// Uses a read lock for the cache lookup and only acquires a write lock when a
// new cert must be stored, so concurrent TLS handshakes for already-cached
// hostnames never block each other.
func (c *CertCache) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	host := hello.ServerName
	if host == "" {
		host = "localhost"
	}

	// Fast path: cache hit under read lock.
	c.mu.RLock()
	cert, ok := c.cache[host]
	c.mu.RUnlock()
	if ok {
		return cert, nil
	}

	// Slow path: generate outside any lock so concurrent handshakes for
	// different hostnames don't serialize behind a single write lock.
	cert, err := c.generate([]string{host})
	if err != nil {
		return nil, err
	}

	// Store under write lock; double-check to avoid overwriting a cert that
	// another goroutine may have generated and stored between our RUnlock and
	// now.
	c.mu.Lock()
	if existing, ok := c.cache[host]; ok {
		c.mu.Unlock()
		return existing, nil
	}
	c.cache[host] = cert
	c.mu.Unlock()
	return cert, nil
}

// Generate returns a TLS certificate for the given hostnames, keyed by the
// first hostname. It populates the cache so that subsequent GetCertificate
// calls for the same hostname return immediately without re-generating.
// This makes pre-warming in Serve() effective.
func (c *CertCache) Generate(hosts []string) (*tls.Certificate, error) {
	if len(hosts) == 0 {
		return nil, nil
	}
	key := hosts[0]

	// Fast path: already cached.
	c.mu.RLock()
	cert, ok := c.cache[key]
	c.mu.RUnlock()
	if ok {
		return cert, nil
	}

	cert, err := c.generate(hosts)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	if existing, ok := c.cache[key]; ok {
		c.mu.Unlock()
		return existing, nil
	}
	c.cache[key] = cert
	c.mu.Unlock()
	return cert, nil
}

// generate creates a fresh signed leaf certificate. It is intentionally
// lock-free so callers can run it outside any mutex.
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
