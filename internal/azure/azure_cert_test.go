package azure

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_extractCertificateDomains_PEM(t *testing.T) {
	_, pemData := newTestCertificate(t, "example.com", []string{"www.example.com", "api.example.com"})

	domains, err := extractCertificateDomains(pemData)

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"www.example.com", "api.example.com", "example.com"}, domains)
}

func Test_extractCertificateDomains_Base64(t *testing.T) {
	der, _ := newTestCertificate(t, "example.com", []string{"example.com", "alt.example.com"})
	encoded := base64.StdEncoding.EncodeToString(der)

	domains, err := extractCertificateDomains(encoded)

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"example.com", "alt.example.com"}, domains)
}

func Test_extractCertificateDomains_Empty(t *testing.T) {
	domains, err := extractCertificateDomains("")

	assert.NoError(t, err)
	assert.Empty(t, domains)
}

func Test_extractCertificateDomains_Invalid(t *testing.T) {
	_, err := extractCertificateDomains("not-base64")

	assert.Error(t, err)
}

func newTestCertificate(t *testing.T, commonName string, dnsNames []string) ([]byte, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		DNSNames:  dnsNames,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if pemBytes == nil {
		t.Fatal("failed to encode PEM certificate")
	}

	return derBytes, string(pemBytes)
}
