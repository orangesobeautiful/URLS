package common

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
)

const InternalTLSServerName = "urlssrvc"

// GenInternalTLSConfig 生成內部服務的 tls config
func GenInternalTLSConfig(serverName string) (tlsConfig *tls.Config, err error) {
	certsDir := filepath.Join("server-data", "certs")
	// Load the service certificate and its key
	srvcCert, err := tls.LoadX509KeyPair(
		filepath.Join(certsDir, "srvc.crt"),
		filepath.Join(certsDir, "srvc.key"))
	if err != nil {
		return
	}

	// Load the CA certificate
	caCert, err := os.ReadFile(filepath.Join(certsDir, "ca.crt"))
	if err != nil {
		return
	}

	// Put the CA certificate to certificate pool
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return
	}

	// Create the TLS configuration
	tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{srvcCert},
		RootCAs:      certPool,
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS12,
		ServerName:   serverName,
	}

	return tlsConfig, nil
}
