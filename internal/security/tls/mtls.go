package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
)

// MTLSConfig holds mTLS configuration
type MTLSConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	CertFile   string `mapstructure:"cert_file"`
	KeyFile    string `mapstructure:"key_file"`
	CAFile     string `mapstructure:"ca_file"`
	ServerName string `mapstructure:"server_name"`
}

// MTLSProvider provides mTLS credentials for gRPC
type MTLSProvider struct {
	config *MTLSConfig
}

// NewMTLSProvider creates a new mTLS provider
func NewMTLSProvider(config *MTLSConfig) *MTLSProvider {
	return &MTLSProvider{config: config}
}

// ServerCredentials returns gRPC server credentials with mTLS
func (p *MTLSProvider) ServerCredentials() (credentials.TransportCredentials, error) {
	if !p.config.Enabled {
		return nil, nil
	}

	// Load server certificate and key
	cert, err := tls.LoadX509KeyPair(p.config.CertFile, p.config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// Load CA certificate for client verification
	caCert, err := os.ReadFile(p.config.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS13,
	}

	return credentials.NewTLS(tlsConfig), nil
}

// ClientCredentials returns gRPC client credentials with mTLS
func (p *MTLSProvider) ClientCredentials() (credentials.TransportCredentials, error) {
	if !p.config.Enabled {
		return nil, nil
	}

	// Load client certificate and key
	cert, err := tls.LoadX509KeyPair(p.config.CertFile, p.config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate for server verification
	caCert, err := os.ReadFile(p.config.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		ServerName:   p.config.ServerName,
		MinVersion:   tls.VersionTLS13,
	}

	return credentials.NewTLS(tlsConfig), nil
}

// IsEnabled returns whether mTLS is enabled
func (p *MTLSProvider) IsEnabled() bool {
	return p.config.Enabled
}
