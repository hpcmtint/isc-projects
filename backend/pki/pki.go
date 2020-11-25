package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Convert binary data to PEM format using provided block type.
func toPEM(blockType string, bytes []byte) []byte {
	b := pem.Block{Type: blockType, Bytes: bytes}
	certPEM := pem.EncodeToMemory(&b)
	return certPEM
}

// Generate RSA key and convert it to PEM format.
func genRSAKey() (*rsa.PrivateKey, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Fatalf("cannot generate RSA key: %v", err)
		return nil, nil, err
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
		return nil, nil, err
	}

	pem := toPEM("PRIVATE KEY", privBytes)

	return priv, pem, nil
}

// Create certificate based on template using parent cert, publick key and private parent key.
// Convert it to PEM format.
func createCert(template, parent *x509.Certificate, publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey) (*x509.Certificate, []byte, error) {
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent, publicKey, privateKey)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse certificate")
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to parse certificate")
	}

	certPEM := toPEM("CERTIFICATE", certBytes)

	return cert, certPEM, nil
}

// Generate root CA key and certifacte. Return them in PEM format.
func GenCACert(serialNumber int64) (*rsa.PrivateKey, []byte, *x509.Certificate, []byte, error) {
	rootTemplate := x509.Certificate{
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			Country:      []string{"US"},
			Organization: []string{"ISC Stork"},
			CommonName:   "Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(30, 0, 0), // 30 years of cert validity
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	privKey, privKeyPEM, err := genRSAKey()
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "problem with generating RSA key")
	}
	rootCert, rootPEM, err := createCert(&rootTemplate, &rootTemplate, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "problem with generating certificate")
	}
	return privKey, privKeyPEM, rootCert, rootPEM, nil
}

// Generate key and cerfication for provided DNS names and IP addresses, using provided serial number and CA key and cert.
// Return them in PEM format.
func GenKeyCert(name string, dnsNames []string, ipAddresses []net.IP, serialNumber int64, parentCert *x509.Certificate, parentKey *rsa.PrivateKey) ([]byte, []byte, error) {
	// generate a key pair
	privKey, privKeyPEM, err := genRSAKey()
	if err != nil {
		return nil, nil, err
	}

	// prepare cert template
	template := x509.Certificate{
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"ISC Stork"},
			OrganizationalUnit: []string{name},
			CommonName:         dnsNames[0],
		},
		NotBefore:      time.Now(),
		NotAfter:       time.Now().AddDate(30, 0, 0), // 30 years of cert validity
		IsCA:           true,
		MaxPathLenZero: true,
		IPAddresses:    ipAddresses,
		DNSNames:       dnsNames,
	}

	// prepare cert by signing template and public key using parent cert and parent priv key
	_, certPEM, err := createCert(&template, parentCert, &privKey.PublicKey, parentKey)
	if err != nil {
		return nil, nil, err
	}
	return certPEM, privKeyPEM, nil
}

// Generate CSR (Certificate Signing Request) for provided private key and DNS names and IP addresses.
func GenCSRUsingKey(name string, dnsNames []string, ipAddresses []net.IP, privKeyPEM []byte) ([]byte, [32]byte, error) {
	var fingerprint [32]byte

	// parse priv key
	pemBlock, _ := pem.Decode(privKeyPEM)
	privKeyIf, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, fingerprint, errors.Wrapf(err, "parsing priv key")
	}
	privKey := privKeyIf.(*rsa.PrivateKey)

	var commonName string
	if len(dnsNames) > 0 {
		commonName = dnsNames[0]
	} else {
		commonName = ipAddresses[0].String()
	}

	// generate a CSR template
	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"ISC Stork"},
			OrganizationalUnit: []string{name},
			CommonName:         commonName,
		},
		IPAddresses: ipAddresses,
		DNSNames:    dnsNames,
	}
	// generate the CSR request
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privKey)
	if err != nil {
		return nil, fingerprint, err
	}
	csrPEM := toPEM("CERTIFICATE REQUEST", csrBytes)

	fingerprint = sha256.Sum256(csrBytes)

	return csrPEM, fingerprint, nil
}

// Generate RSA key and CSR for it. Return them in PEM format with fingerprint.
func GenKeyAndCSR(name string, dnsNames []string, ipAddresses []net.IP) ([]byte, []byte, [32]byte, error) {
	var fingerprint [32]byte

	// generate a key pair
	_, privKeyPEM, err := genRSAKey()
	if err != nil {
		return nil, nil, fingerprint, err
	}

	// create CSR using priv key
	csrPEM, fingerprint, err := GenCSRUsingKey(name, dnsNames, ipAddresses, privKeyPEM)
	if err != nil {
		return nil, nil, fingerprint, err
	}

	return privKeyPEM, csrPEM, fingerprint, nil
}

// Parse certificate in PEM format.
func ParseCert(certPEM []byte) (*x509.Certificate, error) {
	pemBlock, _ := pem.Decode(certPEM)
	if pemBlock == nil {
		return nil, errors.New("decoding PEM with cert failed")
	}
	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing cert failed")
	}
	return cert, nil
}

// Sign cerificate for given CSR in PEM format using provided serial number and CA key and cert.
func SignCert(csrPEM []byte, serialNumber int64, parentCertPEM []byte, parentKeyPEM []byte) ([]byte, [32]byte, error, error) {
	var fingerprint [32]byte

	// parse and check CSR
	pemBlock, _ := pem.Decode(csrPEM)
	if pemBlock == nil {
		return nil, fingerprint, errors.New("decoding PEM with CSR failed"), nil
	}
	csr, err := x509.ParseCertificateRequest(pemBlock.Bytes)
	if err != nil {
		return nil, fingerprint, errors.Wrapf(err, "parsing CSR failed"), nil
	}
	if err = csr.CheckSignature(); err != nil {
		return nil, fingerprint, errors.Wrapf(err, "checking CSR signature failed"), nil
	}

	// parse CA cert and key
	pemBlock, _ = pem.Decode(parentKeyPEM)
	parentKeyIf, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, fingerprint, nil, errors.Wrapf(err, "parsing CA keys")
	}
	parentKey := parentKeyIf.(*rsa.PrivateKey)
	parentCert, err := ParseCert(parentCertPEM)
	if err != nil {
		return nil, fingerprint, nil, errors.Wrapf(err, "parsing CA cert")
	}

	// prepare certificate with information from CSR and from CA cert
	template := x509.Certificate{
		Signature:          csr.Signature,
		SignatureAlgorithm: csr.SignatureAlgorithm,
		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		PublicKey:          csr.PublicKey,

		SerialNumber: big.NewInt(serialNumber),
		Issuer:       parentCert.Subject,
		Subject:      csr.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(30, 0, 0), // 30 years of cert validity
		IPAddresses:  csr.IPAddresses,
		DNSNames:     csr.DNSNames,
	}

	cert, pem, err := createCert(&template, parentCert, csr.PublicKey.(*rsa.PublicKey), parentKey)
	if err != nil {
		return nil, fingerprint, nil, errors.Wrapf(err, "signing agent cert failed")
	}
	fingerprint = sha256.Sum256(cert.Raw)
	return pem, fingerprint, nil, nil
}
