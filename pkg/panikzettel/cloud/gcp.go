package cloud

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/htwr-aachen/backend/pkg/panikzettel/config"
	"github.com/rs/zerolog/log"
	"gocloud.dev/blob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/gcp"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const (
	// GCS scopes required for bucket operations
	scopeStorageReadWrite = "https://www.googleapis.com/auth/devstorage.read_write"
	scopeStorageReadOnly  = "https://www.googleapis.com/auth/devstorage.read_only"
	scopeStorageFull      = "https://www.googleapis.com/auth/devstorage.full_control"
)

type GCPCloudClient struct {
	bucket *blob.Bucket
	signer *gcpSigner
}

func NewGCP(ctx context.Context, serviceCfg *config.Config) (*GCPCloudClient, error) {
	cfg := serviceCfg.CloudConfig

	transport, err := createTLSTransport(serviceCfg)
	if err != nil {
		log.Err(err).Msg("creating GCP TLS transport")
		return nil, fmt.Errorf("creating GCP TLS transport: %w", err)
	}

	creds, clientOpts, err := getGCPCredentials(ctx, cfg)
	if err != nil {
		log.Err(err).Msg("creating credentials to gcp")
		return nil, fmt.Errorf("creating GCP credentials: %w", err)
	}

	httpClient, err := gcp.NewHTTPClient(transport, gcp.CredentialsTokenSource(creds))
	if err != nil {
		log.Err(err).Msg("creating gcp http client")
		return nil, fmt.Errorf("creating gcp http client: %w", err)
	}

	bucketOpts := &gcsblob.Options{
		ClientOptions: clientOpts,
	}

	// Initialize signer only if signing credentials are provided
	var signer *gcpSigner
	if cfg.PrivateKey != "" && cfg.GoogleAccessId != "" {
		signer, err = newGCPSigner(cfg)
		if err != nil {
			log.Err(err).Msg("failed to create signer")
			return nil, fmt.Errorf("creating signer: %w", err)
		}
		log.Info().Str("email", cfg.GoogleAccessId).Msg("signer initialized")

		bucketOpts.SignBytes = signer.Sign
	}

	bucket, err := gcsblob.OpenBucket(ctx, httpClient, cfg.BucketName, bucketOpts)
	if err != nil {
		log.Err(err).Str("bucket", cfg.BucketName).Msg("opening gcs bucket")
		return nil, fmt.Errorf("opening gcs bucket: %w", err)
	}

	log.Info().Str("bucket", cfg.BucketName).Str("provider", cfg.Provider).Msg("successfully connected to cloud storage")

	cloudClient := &GCPCloudClient{
		bucket: bucket,
		signer: signer,
	}

	return cloudClient, nil
}

// createTLSTransport creates an HTTP transport with custom TLS configuration
func createTLSTransport(cfg *config.Config) (http.RoundTripper, error) {
	tlsConfig := &tls.Config{}

	configurator.MergeFromGlobalConfig(&cfg.GlobalConfig.TLS, tlsConfig)
	configurator.MergeFromConnConfig(&cfg.CloudConfig.TLSConfig, tlsConfig)

	// Create transport with custom TLS config
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return transport, nil
}

// getCredentials returns credentials based on the configured auth method
func getGCPCredentials(ctx context.Context, cfg config.CloudConfig) (*google.Credentials, []option.ClientOption, error) {
	var opts []option.ClientOption

	if cfg.ProjectId != "" {
		//opts = append(opts, option.WithQuotaProject(cfg.ProjectID))
	}

	switch cfg.AuthMethod {
	case "service_account":
		return getGCPServiceAccountCredentials(ctx, cfg, opts)
	case "api_key":
		return getGCPAPIKeyCredentials(ctx, cfg, opts)
	case "default", "":
		return getGCPDefaultCredentials(ctx, cfg, opts)
	default:
		return nil, nil, fmt.Errorf("unknown auth method: %s", cfg.AuthMethod)
	}
}

// getGCPDefaultCredentials uses Application Default Credentials (ADC)
// This checks: GOOGLE_APPLICATION_CREDENTIALS env var, gcloud CLI, GCE metadata
func getGCPDefaultCredentials(ctx context.Context, cfg config.CloudConfig, opts []option.ClientOption) (*google.Credentials, []option.ClientOption, error) {
	creds, err := gcp.DefaultCredentials(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("getting default gcp credentials: %w", err)
	}

	log.Info().Str("project", creds.ProjectID).Msg("using default credentials (ADC)")
	return creds, opts, nil
}

// getServiceAccountCredentials uses a service account JSON file or inline JSON
func getGCPServiceAccountCredentials(ctx context.Context, cfg config.CloudConfig, opts []option.ClientOption) (*google.Credentials, []option.ClientOption, error) {
	var credJSON []byte
	var err error

	if cfg.ServiceAccountJSON != "" {
		credJSON, err = base64.StdEncoding.DecodeString(cfg.ServiceAccountJSON)
		if err != nil {
			return nil, nil, fmt.Errorf("decoding gcp service account JSON: %w", err)
		}
		log.Info().Msg("using inline gcp service account credentials")
	} else if cfg.ServiceAccount != "" {
		credJSON, err = os.ReadFile(cfg.ServiceAccount)
		if err != nil {
			return nil, nil, fmt.Errorf("reading gcp service account file: %w", err)
		}
		log.Info().Str("path", cfg.ServiceAccount).Msg("using gcp service account from file")
	} else {
		return nil, nil, fmt.Errorf("service_account auth requires either service_account_path or service_account_json")
	}

	creds, err := google.CredentialsFromJSON(ctx, credJSON, scopeStorageReadOnly)
	if err != nil {
		return nil, nil, fmt.Errorf("creating credentials from gcp service account JSON: %w", err)
	}

	opts = append(opts, option.WithCredentials(creds))

	log.Info().Str("project", creds.ProjectID).Msg("service account credentials loaded successfully")

	return creds, opts, nil
}

// getGCPAPIKeyCredentials uses an API key for authentication
func getGCPAPIKeyCredentials(ctx context.Context, cfg config.CloudConfig, opts []option.ClientOption) (*google.Credentials, []option.ClientOption, error) {
	if cfg.APIKey == "" {
		return nil, nil, fmt.Errorf("api_key auth method requires api_key to be set")
	}

	opts = append(opts, option.WithAPIKey(cfg.APIKey))

	log.Warn().Msg("using API key authentication - this has limited GCS support and is not recommended for production")

	creds, err := gcp.DefaultCredentials(ctx)
	if err != nil {
		log.Warn().Msg("could not get default credentials, using API key only")
		return nil, opts, nil
	}

	return creds, opts, nil
}

func (c *GCPCloudClient) Bucket() *blob.Bucket {
	return c.bucket
}

func (c *GCPCloudClient) Close() {
	var err error
	if c.bucket != nil {
		err = c.bucket.Close()
		if err != nil {
			log.Err(err).Msg("closing gcp bucket")
		}
	}
}

// gcpSigner handles signing operations for GCP
type gcpSigner struct {
	privateKey *rsa.PrivateKey
	email      string
}

// Sign signs the provided data using RSA-SHA256
func (s *gcpSigner) Sign(data []byte) ([]byte, error) {
	if s.privateKey == nil {
		return nil, fmt.Errorf("private key not configured")
	}

	hash := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return nil, fmt.Errorf("signing data: %w", err)
	}

	return signature, nil
}

// newGCPSigner creates a new signer from the configuration
func newGCPSigner(cfg config.CloudConfig) (*gcpSigner, error) {
	// Read private key file
	keyData, err := os.ReadFile(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("reading private key from '%s': %w", cfg.PrivateKey, err)
	}

	// Parse private key (supports both PKCS1 and PKCS8 formats)
	privateKey, err := parsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	return &gcpSigner{
		privateKey: privateKey,
		email:      cfg.GoogleAccessId,
	}, nil
}

// parsePrivateKey parses RSA private key from PEM data
func parsePrivateKey(keyData []byte) (*rsa.PrivateKey, error) {
	// Try PKCS8 format first (used by GCP service accounts)
	key, err := x509.ParsePKCS8PrivateKey(keyData)
	if err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not an RSA private key")
		}
		return rsaKey, nil
	}

	// Try PKCS1 format
	rsaKey, err := x509.ParsePKCS1PrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key (tried PKCS8 and PKCS1): %w", err)
	}

	return rsaKey, nil
}
