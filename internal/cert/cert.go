// Check if cert and key exists, if not, generates self-signed cert using ECDSA (P-256)
package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// EnsureCert checks if cert and key exist, otherwise generates a self-signed certificate.
func EnsureCert(certPath, keyPath string) (string, string, error) {
	if certPath != "" && keyPath != "" {
		if fileExists(certPath) && fileExists(keyPath) {
			return certPath, keyPath, nil
		}
	}

	// Set default paths if not provided
	if certPath == "" {
		certPath = filepath.Join("certs", "selfsigned.crt")
	}
	if keyPath == "" {
		keyPath = filepath.Join("certs", "selfsigned.key")
	}

	if err := os.MkdirAll(filepath.Dir(certPath), 0700); err != nil {
		return "", "", err
	}

	// Generate self-signed cert
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Revproxy"},
		},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"*"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", "", err
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return "", "", err
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyOut, err := os.Create(keyPath)
	if err != nil {
		return "", "", err
	}
	defer keyOut.Close()
	b, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return "", "", err
	}
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b})

	return certPath, keyPath, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
