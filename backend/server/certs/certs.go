package certs

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"math/rand"
	"net"

	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"isc.org/stork/pki"
	dbmodel "isc.org/stork/server/database/model"
)

// Generate server token and store it in database.
// It is used during manual agent registration.
func GenerateServerToken(db *pg.DB) ([]byte, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	serverToken := make([]byte, 32)
	for i := range serverToken {
		serverToken[i] = chars[rand.Intn(len(chars))] // #nosec G404
	}
	err := dbmodel.SetSecret(db, dbmodel.SecretServerToken, serverToken)
	if err != nil {
		return nil, err
	}
	return serverToken, nil
}

// Check if root CA key and certs are present in db. If not generate them
// and store in database.
func setupRootKeyAndCert(db *pg.DB) (*ecdsa.PrivateKey, *x509.Certificate, []byte, error) {
	// check root key and root cert, if missing then generate them
	rootKeyPEM, err := dbmodel.GetSecret(db, dbmodel.SecretCAKey)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "problem with getting CA key from database")
	}
	rootCertPEM, err := dbmodel.GetSecret(db, dbmodel.SecretCACert)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "problem with getting CA cert from database")
	}

	var rootKey *ecdsa.PrivateKey
	var rootCert *x509.Certificate

	// no root key or no root cert so generate
	if rootKeyPEM == nil || rootCertPEM == nil {
		certSerialNumber, err := dbmodel.GetNewCertSerialNumber(db)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "cannot get new cert S/N")
		}
		rootKey, rootKeyPEM, rootCert, rootCertPEM, err = pki.GenCACert(certSerialNumber)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "cannot generate root CA cert")
		}
		err = dbmodel.SetSecret(db, dbmodel.SecretCAKey, rootKeyPEM)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "cannot store root CA key in database")
		}
		err = dbmodel.SetSecret(db, dbmodel.SecretCACert, rootCertPEM)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "cannot store root CA key in database")
		}
		log.Printf("generated root CA key and cert")
	} else {
		rootKeyPEMBlock, _ := pem.Decode(rootKeyPEM)
		rootKeyIf, err := x509.ParsePKCS8PrivateKey(rootKeyPEMBlock.Bytes)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "cannot parse root CA key")
		}
		rootKey = rootKeyIf.(*ecdsa.PrivateKey)

		rootCertPEMBlock, _ := pem.Decode(rootCertPEM)
		rootCert, err = x509.ParseCertificate(rootCertPEMBlock.Bytes)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "cannot parse root CA cert")
		}
		log.Printf("root CA key and cert ok")
	}

	return rootKey, rootCert, rootCertPEM, nil
}

// Check if server key and certs are present in db. If not generate them
// and store in database.
func setupServerKeyAndCert(db *pg.DB, rootKey *ecdsa.PrivateKey, rootCert *x509.Certificate) ([]byte, []byte, error) {
	// check server cert, if missing then generate it
	serverKeyPEM, err := dbmodel.GetSecret(db, dbmodel.SecretServerKey)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot get server key from database")
	}
	serverCertPEM, err := dbmodel.GetSecret(db, dbmodel.SecretServerCert)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot get server cert from database")
	}

	if serverKeyPEM == nil || serverCertPEM == nil {
		// get list of all host IP addresses that will be put to server cert
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot get interface addresses")
		}
		var srvIPs []net.IP
		var srvNames []string
		for _, addr := range addrs {
			ipAddr, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			srvIPs = append(srvIPs, ipAddr)
			names, err := net.LookupAddr(ipAddr.String())
			if err == nil {
				srvNames = append(srvNames, names...)
			}
		}
		if len(srvIPs) == 0 || len(srvNames) == 0 {
			return nil, nil, errors.Errorf("cannot find IP addresses on this host")
		}

		certSerialNumber, err := dbmodel.GetNewCertSerialNumber(db)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot get new cert S/N")
		}
		serverCertPEM, serverKeyPEM, err = pki.GenKeyCert("server", srvNames, srvIPs, certSerialNumber, rootCert, rootKey)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot generate key and cert for server")
		}
		err = dbmodel.SetSecret(db, dbmodel.SecretServerKey, serverKeyPEM)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot store server key in database")
		}
		err = dbmodel.SetSecret(db, dbmodel.SecretServerCert, serverCertPEM)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot store server cert in database")
		}
		log.Printf("generated server key and cert")
	} else {
		// check server key
		serverKeyPEMBlock, _ := pem.Decode(serverKeyPEM)
		_, err := x509.ParsePKCS8PrivateKey(serverKeyPEMBlock.Bytes)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot parse server key")
		}

		serverCertPEMBlock, _ := pem.Decode(serverCertPEM)
		_, err = x509.ParseCertificate(serverCertPEMBlock.Bytes)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "cannot parse server cert")
		}
		log.Printf("server key and cert ok")
	}

	return serverKeyPEM, serverCertPEM, nil
}

// Prepare root CA and server keys and certs, and server token.
func SetupServerCerts(db *pg.DB) ([]byte, []byte, []byte, error) {
	log.Printf("preparing certs, it may take up to several minutes")

	// setup root CA key and cert
	rootKey, rootCert, rootCertPEM, err := setupRootKeyAndCert(db)
	if err != nil {
		return nil, nil, nil, err
	}

	// setup server key and cert using root CA key and cert
	serverKeyPEM, serverCertPEM, err := setupServerKeyAndCert(db, rootKey, rootCert)
	if err != nil {
		return nil, nil, nil, err
	}

	// check server access token; if missing generate it
	serverToken, err := dbmodel.GetSecret(db, dbmodel.SecretServerToken)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "cannot get server token from database")
	}
	if serverToken == nil {
		_, err = GenerateServerToken(db)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "cannot generate server token")
		}
	}

	return rootCertPEM, serverCertPEM, serverKeyPEM, nil
}
