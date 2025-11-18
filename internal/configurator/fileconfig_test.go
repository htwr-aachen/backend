package configurator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

// --- Helper Functions ---

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	fPath := filepath.Join(tmpDir, "test_secret.txt")
	if err := os.WriteFile(fPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	return fPath
}

// --- Tests ---

func TestReadSecretFile(t *testing.T) {
	t.Run("Read valid file", func(t *testing.T) {
		content := "my-secret-password"
		path := createTempFile(t, content)

		result, err := ReadSecretFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != content {
			t.Errorf("expected '%s', got '%s'", content, result)
		}
	})

	t.Run("Read file with whitespace (Trim)", func(t *testing.T) {
		content := "  my-secret-password\n\n"
		expected := "my-secret-password"
		path := createTempFile(t, content)

		result, err := ReadSecretFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("Read file with only newlines", func(t *testing.T) {
		path := createTempFile(t, "\n\n\n")
		result, err := ReadSecretFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})

	t.Run("Empty path returns empty string", func(t *testing.T) {
		result, err := ReadSecretFile("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})

	t.Run("Nonexistent file returns error", func(t *testing.T) {
		_, err := ReadSecretFile("/nonexistent/path/secret.txt")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestReadFile(t *testing.T) {
	// Test for ReadFile (Raw bytes, no trim)
	t.Run("Read raw bytes without trim", func(t *testing.T) {
		content := "  data  "
		path := createTempFile(t, content)

		result, err := ReadFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result) != content {
			t.Errorf("expected '%s' (untrimmed), got '%s'", content, string(result))
		}
	})
}

func TestGetStringOrFile(t *testing.T) {
	t.Run("Direct value when no file specified", func(t *testing.T) {
		k := koanf.New(".")
		k.Load(confmap.Provider(map[string]any{
			"password": "direct-password",
		}, "."), nil)

		result, err := GetStringOrFile(k, "password")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "direct-password" {
			t.Errorf("expected 'direct-password', got '%s'", result)
		}
	})

	t.Run("File value takes precedence", func(t *testing.T) {
		secretFile := createTempFile(t, "file-password")
		k := koanf.New(".")
		k.Load(confmap.Provider(map[string]any{
			"password":      "direct-password",
			"password_file": secretFile,
		}, "."), nil)

		result, err := GetStringOrFile(k, "password")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "file-password" {
			t.Errorf("expected 'file-password', got '%s'", result)
		}
	})

	t.Run("File with whitespace is trimmed", func(t *testing.T) {
		secretFile := createTempFile(t, "  password123\n")
		k := koanf.New(".")
		k.Load(confmap.Provider(map[string]any{
			"password_file": secretFile,
		}, "."), nil)

		result, err := GetStringOrFile(k, "password")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "password123" {
			t.Errorf("expected 'password123', got '%s'", result)
		}
	})
}

func TestGetBytesOrFile(t *testing.T) {
	t.Run("Reads bytes from file without trimming", func(t *testing.T) {
		content := "  raw-data  "
		secretFile := createTempFile(t, content)
		k := koanf.New(".")
		k.Load(confmap.Provider(map[string]any{
			"cert_file": secretFile,
		}, "."), nil)

		result, err := GetBytesOrFile(k, "cert")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result) != content {
			t.Errorf("expected '%s', got '%s'", content, string(result))
		}
	})
}

func TestResolveFiles(t *testing.T) {
	t.Run("Simple struct with file field", func(t *testing.T) {
		secretFile := createTempFile(t, "secret-value")

		type Config struct {
			Password     string
			PasswordFile string
		}

		config := &Config{
			PasswordFile: secretFile,
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Password != "secret-value" {
			t.Errorf("expected 'secret-value', got '%s'", config.Password)
		}
	})

	t.Run("Resolves []byte fields (raw content)", func(t *testing.T) {
		content := "  binary-data  "
		secretFile := createTempFile(t, content)

		type Config struct {
			Cert     []byte
			CertFile string
		}

		config := &Config{
			CertFile: secretFile,
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if string(config.Cert) != content {
			t.Errorf("expected raw content '%s', got '%s'", content, string(config.Cert))
		}
	})

	t.Run("Nested struct with file fields", func(t *testing.T) {
		secretFile := createTempFile(t, "nested-secret")

		type DatabaseConfig struct {
			Password     string
			PasswordFile string
		}

		type Config struct {
			Database DatabaseConfig
		}

		config := &Config{
			Database: DatabaseConfig{
				PasswordFile: secretFile,
			},
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Database.Password != "nested-secret" {
			t.Errorf("expected 'nested-secret', got '%s'", config.Database.Password)
		}
	})

	t.Run("Nested pointer struct", func(t *testing.T) {
		secretFile := createTempFile(t, "pointer-secret")

		type DatabaseConfig struct {
			Password     string
			PasswordFile string
		}

		type Config struct {
			Database *DatabaseConfig
		}

		config := &Config{
			Database: &DatabaseConfig{
				PasswordFile: secretFile,
			},
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Database.Password != "pointer-secret" {
			t.Errorf("expected 'pointer-secret', got '%s'", config.Database.Password)
		}
	})

	t.Run("Nil nested pointer struct is safe", func(t *testing.T) {
		type DatabaseConfig struct {
			PasswordFile string
		}
		type Config struct {
			Database *DatabaseConfig
		}
		config := &Config{Database: nil}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Skip field with tag", func(t *testing.T) {
		secretFile := createTempFile(t, "secret-value")

		type Config struct {
			CertPath     string
			CertPathFile string `fileconfig:"skip"`
		}

		config := &Config{
			CertPathFile: secretFile,
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should NOT resolve because of skip tag
		if config.CertPath != "" {
			t.Errorf("expected empty CertPath due to skip tag, got '%s'", config.CertPath)
		}
	})

	t.Run("Direct value is preserved if file field is empty", func(t *testing.T) {
		type Config struct {
			Password     string
			PasswordFile string
		}

		config := &Config{
			Password:     "direct-value",
			PasswordFile: "",
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Password != "direct-value" {
			t.Errorf("expected 'direct-value' to be preserved, got '%s'", config.Password)
		}
	})

	t.Run("Nonexistent file returns error", func(t *testing.T) {
		type Config struct {
			Password     string
			PasswordFile string
		}

		config := &Config{
			PasswordFile: "/nonexistent/secret.txt",
		}

		err := ResolveFiles(config)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("Map of structs", func(t *testing.T) {
		secret1File := createTempFile(t, "secret-one")
		secret2File := createTempFile(t, "secret-two")

		type ProviderConfig struct {
			APIKey     string
			APIKeyFile string
		}

		type Config struct {
			Providers map[string]ProviderConfig
		}

		config := &Config{
			Providers: map[string]ProviderConfig{
				"provider1": {APIKeyFile: secret1File},
				"provider2": {APIKeyFile: secret2File},
			},
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Providers["provider1"].APIKey != "secret-one" {
			t.Errorf("provider1: expected 'secret-one', got '%s'", config.Providers["provider1"].APIKey)
		}
		if config.Providers["provider2"].APIKey != "secret-two" {
			t.Errorf("provider2: expected 'secret-two', got '%s'", config.Providers["provider2"].APIKey)
		}
	})
}

func TestUnmarshalWithFileResolution(t *testing.T) {
	t.Run("Unmarshal and resolve combined", func(t *testing.T) {
		secretFile := createTempFile(t, "file-secret")

		type Config struct {
			Username     string `koanf:"username"`
			Password     string `koanf:"password"`
			PasswordFile string `koanf:"password_file"`
		}

		k := koanf.New(".")
		k.Load(confmap.Provider(map[string]any{
			"username":      "admin",
			"password":      "direct-password", // Should be overwritten by file
			"password_file": secretFile,
		}, "."), nil)

		var config Config
		err := UnmarshalWithFileResolution(k, &config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Username != "admin" {
			t.Errorf("expected username 'admin', got '%s'", config.Username)
		}
		if config.Password != "file-secret" {
			t.Errorf("expected password 'file-secret', got '%s'", config.Password)
		}
	})

	t.Run("Unmarshal error propagation", func(t *testing.T) {
		type Config struct {
			Port int `koanf:"port"`
		}
		k := koanf.New(".")
		k.Load(confmap.Provider(map[string]any{"port": "not-a-number"}, "."), nil)

		var config Config
		err := UnmarshalWithFileResolution(k, &config)
		if err == nil {
			t.Error("expected unmarshal error")
		}
	})
}
