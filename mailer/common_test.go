//go:build test
// +build test

package mailer

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createTestCA() (*os.File, *os.File, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			Country:      []string{"JP"},
			CommonName:   "Test CA",
		},
		BasicConstraintsValid: true,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		IsCA:      true,
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(0, 0, 7),
	}
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.CreateCertificate(rand.Reader, ca, ca, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}
	caPem := new(bytes.Buffer)
	key := new(bytes.Buffer)
	err = pem.Encode(caPem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
	if err != nil {
		return nil, nil, err
	}
	err = pem.Encode(key, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
	if err != nil {
		return nil, nil, err
	}

	caFile, err := os.CreateTemp("/tmp", "ca.crt")
	if err != nil {
		return nil, nil, err
	}
	keyFile, err := os.CreateTemp("/tmp", "ca.key")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(caFile.Name(), caPem.Bytes(), 0644); err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(keyFile.Name(), key.Bytes(), 0640); err != nil {
		return nil, nil, err
	}
	return caFile, keyFile, nil
}

func createServerCert(caCert, caKey *os.File) (*os.File, *os.File, error) {
	caPem, err := os.ReadFile(caCert.Name())
	if err != nil {
		return nil, nil, err
	}
	keyPem, err := os.ReadFile(caKey.Name())
	if err != nil {
		return nil, nil, err
	}
	block, _ := pem.Decode(caPem)
	ca, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	block, _ = pem.Decode(keyPem)
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	certPriv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			Country:      []string{"JP"},
			CommonName:   "Test Server",
		},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:    []string{"localhost"},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, 1),
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		KeyUsage: x509.KeyUsageDigitalSignature,
	}
	cert, err := x509.CreateCertificate(rand.Reader, certTemplate, ca, &certPriv.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	serverCertPem := new(bytes.Buffer)
	serverKeyPem := new(bytes.Buffer)
	err = pem.Encode(serverCertPem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
	if err != nil {
		return nil, nil, err
	}
	err = pem.Encode(serverKeyPem, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPriv),
	})
	if err != nil {
		return nil, nil, err
	}
	serverCertFile, err := os.CreateTemp("/tmp", "server.crt")
	if err != nil {
		return nil, nil, err
	}
	serverKeyFile, err := os.CreateTemp("/tmp", "server.key")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(serverCertFile.Name(), serverCertPem.Bytes(), 0644); err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(serverKeyFile.Name(), serverKeyPem.Bytes(), 0640); err != nil {
		return nil, nil, err
	}
	return serverCertFile, serverKeyFile, nil
}

func TestLoadCACert(t *testing.T) {
	caCert, key, err := createTestCA()
	defer os.Remove(caCert.Name())
	defer os.Remove(key.Name())
	assert.NoError(t, err)
	pool, err := LoadCACert(caCert.Name())
	assert.NoError(t, err)
	assert.NotNil(t, pool)
	nilPool, err := LoadCACert("hoge")
	assert.Error(t, err)
	assert.Nil(t, nilPool)
}
