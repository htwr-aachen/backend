package configurator

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/htwr-aachen/backend/pkg/config"
)

// generateCACert generates a self-signed CA certificate
func generateCACert(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	// Generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "Test CA",
			Organization: []string{"Test Organization"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	return certPEM, keyPEM
}

// generateClientCert generates a client certificate signed by the given CA
func generateClientCert(t *testing.T, caCertPEM, caKeyPEM []byte) (certPEM, keyPEM []byte) {
	t.Helper()

	// Parse CA certificate
	caCertBlock, _ := pem.Decode(caCertPEM)
	if caCertBlock == nil {
		t.Fatal("failed to decode CA certificate")
	}
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		t.Fatalf("failed to parse CA certificate: %v", err)
	}

	// Parse CA private key
	caKeyBlock, _ := pem.Decode(caKeyPEM)
	if caKeyBlock == nil {
		t.Fatal("failed to decode CA private key")
	}
	caKey, err := x509.ParseECPrivateKey(caKeyBlock.Bytes)
	if err != nil {
		t.Fatalf("failed to parse CA private key: %v", err)
	}

	// Generate client private key
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate client private key: %v", err)
	}

	// Create client certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "Test Client",
			Organization: []string{"Test Organization"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Create client certificate signed by CA
	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create client certificate: %v", err)
	}

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	clientKeyBytes, err := x509.MarshalECPrivateKey(clientKey)
	if err != nil {
		t.Fatalf("failed to marshal client private key: %v", err)
	}

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: clientKeyBytes,
	})

	return certPEM, keyPEM
}

func TestParseTLSVersion(t *testing.T) {
	tests := []struct {
		name        string
		versionStr  string
		expected    uint16
		expectError bool
	}{
		{
			name:        "TLS 1.2",
			versionStr:  TLSVersion12,
			expected:    tls.VersionTLS12,
			expectError: false,
		},
		{
			name:        "TLS 1.3",
			versionStr:  TLSVersion13,
			expected:    tls.VersionTLS13,
			expectError: false,
		},
		{
			name:        "Invalid version",
			versionStr:  "tls1.1",
			expected:    tls.VersionTLS12,
			expectError: true,
		},
		{
			name:        "Empty string",
			versionStr:  "",
			expected:    tls.VersionTLS12,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTLSVersion(tt.versionStr)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMergeFromConnConfig(t *testing.T) {
	t.Run("MinVersion set", func(t *testing.T) {
		cfg := &config.ConnectionTLSConfig{
			MinVersionStr: TLSVersion13,
			MinVersion:    tls.VersionTLS13,
		}
		tlsConfig := &tls.Config{}

		err := MergeFromConnConfig(cfg, tlsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tlsConfig.MinVersion != tls.VersionTLS13 {
			t.Errorf("expected MinVersion %v, got %v", tls.VersionTLS13, tlsConfig.MinVersion)
		}
	})

	t.Run("MaxVersion set", func(t *testing.T) {
		cfg := &config.ConnectionTLSConfig{
			MaxVersionStr: TLSVersion12,
			MaxVersion:    tls.VersionTLS12,
		}
		tlsConfig := &tls.Config{}

		err := MergeFromConnConfig(cfg, tlsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tlsConfig.MaxVersion != tls.VersionTLS12 {
			t.Errorf("expected MaxVersion %v, got %v", tls.VersionTLS12, tlsConfig.MaxVersion)
		}
	})

	t.Run("InsecureSkipVerify set", func(t *testing.T) {
		cfg := &config.ConnectionTLSConfig{
			InsecureSkipVerify: true,
		}
		tlsConfig := &tls.Config{}

		err := MergeFromConnConfig(cfg, tlsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !tlsConfig.InsecureSkipVerify {
			t.Error("expected InsecureSkipVerify to be true")
		}
	})

	t.Run("ServerName set", func(t *testing.T) {
		cfg := &config.ConnectionTLSConfig{
			ServerName: "example.com",
		}
		tlsConfig := &tls.Config{}

		err := MergeFromConnConfig(cfg, tlsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tlsConfig.ServerName != "example.com" {
			t.Errorf("expected ServerName 'example.com', got '%s'", tlsConfig.ServerName)
		}
	})

	t.Run("Custom CA certificate", func(t *testing.T) {
		tmpDir := t.TempDir()
		certFile := filepath.Join(tmpDir, "ca.crt")

		// Generate CA certificate
		caCertPEM, _ := generateCACert(t)
		if err := os.WriteFile(certFile, caCertPEM, 0644); err != nil {
			t.Fatalf("failed to write test cert: %v", err)
		}

		cfg := &config.ConnectionTLSConfig{
			ServerCert: certFile,
		}
		tlsConfig := &tls.Config{}

		err := MergeFromConnConfig(cfg, tlsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tlsConfig.RootCAs == nil {
			t.Error("expected RootCAs to be set")
		}
	})

	t.Run("Invalid CA certificate file", func(t *testing.T) {
		cfg := &config.ConnectionTLSConfig{
			ServerCert: "/nonexistent/path/ca.crt",
		}
		tlsConfig := &tls.Config{}

		err := MergeFromConnConfig(cfg, tlsConfig)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("Invalid CA certificate content", func(t *testing.T) {
		tmpDir := t.TempDir()
		certFile := filepath.Join(tmpDir, "bad.crt")
		if err := os.WriteFile(certFile, []byte("not a cert"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		cfg := &config.ConnectionTLSConfig{
			ServerCert: certFile,
		}
		tlsConfig := &tls.Config{}

		err := MergeFromConnConfig(cfg, tlsConfig)
		if err == nil {
			t.Error("expected error for invalid cert content")
		}
	})

	t.Run("Client certificate and key", func(t *testing.T) {
		tmpDir := t.TempDir()
		certFile := filepath.Join(tmpDir, "client.crt")
		keyFile := filepath.Join(tmpDir, "client.key")

		// Generate CA and client certificates
		caCertPEM, caKeyPEM := generateCACert(t)
		clientCertPEM, clientKeyPEM := generateClientCert(t, caCertPEM, caKeyPEM)

		if err := os.WriteFile(certFile, clientCertPEM, 0644); err != nil {
			t.Fatalf("failed to write test cert: %v", err)
		}
		if err := os.WriteFile(keyFile, clientKeyPEM, 0600); err != nil {
			t.Fatalf("failed to write test key: %v", err)
		}

		cfg := &config.ConnectionTLSConfig{
			ClientCert: certFile,
			ClientKey:  keyFile,
		}
		tlsConfig := &tls.Config{}

		err := MergeFromConnConfig(cfg, tlsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tlsConfig.Certificates) != 1 {
			t.Errorf("expected 1 certificate, got %d", len(tlsConfig.Certificates))
		}
	})

	t.Run("Client certificate without key", func(t *testing.T) {
		tmpDir := t.TempDir()
		certFile := filepath.Join(tmpDir, "client.crt")

		caCertPEM, caKeyPEM := generateCACert(t)
		clientCertPEM, _ := generateClientCert(t, caCertPEM, caKeyPEM)

		if err := os.WriteFile(certFile, clientCertPEM, 0644); err != nil {
			t.Fatalf("failed to write test cert: %v", err)
		}

		cfg := &config.ConnectionTLSConfig{
			ClientCert: certFile,
			ClientKey:  "/nonexistent/key",
		}
		tlsConfig := &tls.Config{}

		err := MergeFromConnConfig(cfg, tlsConfig)
		if err == nil {
			t.Error("expected error when key file doesn't exist")
		}
	})

	t.Run("Mismatched client certificate and key", func(t *testing.T) {
		tmpDir := t.TempDir()
		certFile := filepath.Join(tmpDir, "client.crt")
		keyFile := filepath.Join(tmpDir, "client.key")

		// Generate two different certificate/key pairs
		caCertPEM, caKeyPEM := generateCACert(t)
		clientCertPEM1, _ := generateClientCert(t, caCertPEM, caKeyPEM)
		_, clientKeyPEM2 := generateClientCert(t, caCertPEM, caKeyPEM)

		if err := os.WriteFile(certFile, clientCertPEM1, 0644); err != nil {
			t.Fatalf("failed to write test cert: %v", err)
		}
		if err := os.WriteFile(keyFile, clientKeyPEM2, 0600); err != nil {
			t.Fatalf("failed to write test key: %v", err)
		}

		cfg := &config.ConnectionTLSConfig{
			ClientCert: certFile,
			ClientKey:  keyFile,
		}
		tlsConfig := &tls.Config{}

		err := MergeFromConnConfig(cfg, tlsConfig)
		if err == nil {
			t.Error("expected error for mismatched cert and key")
		}
	})
}

func TestMergeFromGlobalConfig(t *testing.T) {
	t.Run("MinVersion and MaxVersion set", func(t *testing.T) {
		cfg := &config.GlobalTLSConfig{
			MinVersionStr: TLSVersion12,
			MinVersion:    tls.VersionTLS12,
			MaxVersionStr: TLSVersion13,
			MaxVersion:    tls.VersionTLS13,
		}
		tlsConfig := &tls.Config{}

		err := MergeFromGlobalConfig(cfg, tlsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tlsConfig.MinVersion != tls.VersionTLS12 {
			t.Errorf("expected MinVersion %v, got %v", tls.VersionTLS12, tlsConfig.MinVersion)
		}
		if tlsConfig.MaxVersion != tls.VersionTLS13 {
			t.Errorf("expected MaxVersion %v, got %v", tls.VersionTLS13, tlsConfig.MaxVersion)
		}
	})

	t.Run("Trust bundle set", func(t *testing.T) {
		tmpDir := t.TempDir()
		bundleFile := filepath.Join(tmpDir, "bundle.pem")

		// Generate CA certificate
		caCertPEM, _ := generateCACert(t)
		if err := os.WriteFile(bundleFile, caCertPEM, 0644); err != nil {
			t.Fatalf("failed to write test bundle: %v", err)
		}

		cfg := &config.GlobalTLSConfig{
			TrustBundle: []string{bundleFile},
		}
		tlsConfig := &tls.Config{}

		err := MergeFromGlobalConfig(cfg, tlsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tlsConfig.RootCAs == nil {
			t.Error("expected RootCAs to be set")
		}
	})

	t.Run("Trust bundle with multiple certificates", func(t *testing.T) {
		tmpDir := t.TempDir()
		bundleFile := filepath.Join(tmpDir, "bundle.pem")

		// Generate multiple CA certificates
		caCert1, _ := generateCACert(t)
		caCert2, _ := generateCACert(t)

		// Combine them into a single bundle
		bundle := append(caCert1, caCert2...)
		if err := os.WriteFile(bundleFile, bundle, 0644); err != nil {
			t.Fatalf("failed to write test bundle: %v", err)
		}

		cfg := &config.GlobalTLSConfig{
			TrustBundle: []string{bundleFile},
		}
		tlsConfig := &tls.Config{}

		err := MergeFromGlobalConfig(cfg, tlsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tlsConfig.RootCAs == nil {
			t.Error("expected RootCAs to be set")
		}
	})
	t.Run("Trust bundle with multiple certificate files", func(t *testing.T) {
		tmpDir := t.TempDir()
		bundle1File := filepath.Join(tmpDir, "bundle1.pem")
		bundle2File := filepath.Join(tmpDir, "bundle2.pem")

		// Generate multiple CA certificates
		caCert1, _ := generateCACert(t)
		caCert2, _ := generateCACert(t)

		// Combine them into a single bundle
		bundle1 := caCert1
		if err := os.WriteFile(bundle1File, bundle1, 0644); err != nil {
			t.Fatalf("failed to write test bundle: %v", err)
		}
		// Combine them into a single bundle
		bundle2 := caCert2
		if err := os.WriteFile(bundle2File, bundle2, 0644); err != nil {
			t.Fatalf("failed to write test bundle: %v", err)
		}

		cfg := &config.GlobalTLSConfig{
			TrustBundle: []string{bundle1File, bundle2File},
		}
		tlsConfig := &tls.Config{}

		err := MergeFromGlobalConfig(cfg, tlsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tlsConfig.RootCAs == nil {
			t.Error("expected RootCAs to be set")
		}
	})

	t.Run("Invalid trust bundle file", func(t *testing.T) {
		cfg := &config.GlobalTLSConfig{
			TrustBundle: []string{"/nonexistent/bundle.pem"},
		}
		tlsConfig := &tls.Config{}

		err := MergeFromGlobalConfig(cfg, tlsConfig)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("Invalid trust bundle content", func(t *testing.T) {
		tmpDir := t.TempDir()
		bundleFile := filepath.Join(tmpDir, "bad_bundle.pem")
		if err := os.WriteFile(bundleFile, []byte("invalid content"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		cfg := &config.GlobalTLSConfig{
			TrustBundle: []string{bundleFile},
		}
		tlsConfig := &tls.Config{}

		err := MergeFromGlobalConfig(cfg, tlsConfig)
		if err == nil {
			t.Error("expected error for invalid bundle content")
		}
	})

	t.Run("Nil trust bundle set", func(t *testing.T) {
		cfg := &config.GlobalTLSConfig{
			TrustBundle: nil,
		}
		tlsConfig := &tls.Config{}

		err := MergeFromGlobalConfig(cfg, tlsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("InsecureSkipVerify set", func(t *testing.T) {
		cfg := &config.GlobalTLSConfig{
			InsecureSkipVerify: true,
		}
		tlsConfig := &tls.Config{}

		err := MergeFromGlobalConfig(cfg, tlsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !tlsConfig.InsecureSkipVerify {
			t.Error("expected InsecureSkipVerify to be true")
		}
	})
}

func TestMergeFromConfig(t *testing.T) {
	t.Run("Empty config does not modify tls.Config", func(t *testing.T) {
		cfg := &config.TLSBaseConfig{}
		tlsConfig := &tls.Config{}

		mergeFromConfig(cfg, tlsConfig)

		if tlsConfig.MinVersion != 0 {
			t.Errorf("expected MinVersion to be 0, got %v", tlsConfig.MinVersion)
		}
		if tlsConfig.MaxVersion != 0 {
			t.Errorf("expected MaxVersion to be 0, got %v", tlsConfig.MaxVersion)
		}
		if tlsConfig.InsecureSkipVerify {
			t.Error("expected InsecureSkipVerify to be false")
		}
	})
}

func TestConnectionTLSConfig_MultipleClientCertificates(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "client.crt")
	keyFile := filepath.Join(tmpDir, "client.key")

	// Generate CA and client certificates
	caCertPEM, caKeyPEM := generateCACert(t)
	clientCertPEM, clientKeyPEM := generateClientCert(t, caCertPEM, caKeyPEM)

	if err := os.WriteFile(certFile, clientCertPEM, 0644); err != nil {
		t.Fatalf("failed to write test cert: %v", err)
	}
	if err := os.WriteFile(keyFile, clientKeyPEM, 0600); err != nil {
		t.Fatalf("failed to write test key: %v", err)
	}

	cfg := &config.ConnectionTLSConfig{
		ClientCert: certFile,
		ClientKey:  keyFile,
	}

	// Start with existing certificate
	existingCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatalf("failed to load existing cert: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{existingCert},
	}

	err = MergeFromConnConfig(cfg, tlsConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tlsConfig.Certificates) != 2 {
		t.Errorf("expected 2 certificates, got %d", len(tlsConfig.Certificates))
	}
}

func TestRootCAsAppendCertsFromPEM(t *testing.T) {
	t.Run("Valid certificate pool", func(t *testing.T) {
		caCertPEM, _ := generateCACert(t)

		pool := x509.NewCertPool()
		ok := pool.AppendCertsFromPEM(caCertPEM)
		if !ok {
			t.Error("failed to append valid certificate")
		}
	})

	t.Run("Invalid PEM data", func(t *testing.T) {
		pool := x509.NewCertPool()
		ok := pool.AppendCertsFromPEM([]byte("not a certificate"))
		if ok {
			t.Error("expected AppendCertsFromPEM to return false for invalid data")
		}
	})
}
