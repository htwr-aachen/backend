package configurator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestReadSecretFile(t *testing.T) {
	t.Run("Read valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")
		content := "my-secret-password"

		if err := os.WriteFile(secretFile, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		result, err := ReadSecretFile(secretFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != content {
			t.Errorf("expected '%s', got '%s'", content, result)
		}
	})

	t.Run("Read file with whitespace", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")
		content := "  my-secret-password\n\n"
		expected := "my-secret-password"

		if err := os.WriteFile(secretFile, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		result, err := ReadSecretFile(secretFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("Read file with only newlines", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")
		content := "\n\n\n"

		if err := os.WriteFile(secretFile, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		result, err := ReadSecretFile(secretFile)
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
		result, err := ReadSecretFile("/nonexistent/path/secret.txt")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
		if result != "" {
			t.Errorf("expected empty result on error, got '%s'", result)
		}
	})

	t.Run("Read empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "empty.txt")

		if err := os.WriteFile(secretFile, []byte(""), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		result, err := ReadSecretFile(secretFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})

	t.Run("Read file with special characters", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")
		content := "p@ssw0rd!#$%^&*()"

		if err := os.WriteFile(secretFile, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		result, err := ReadSecretFile(secretFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != content {
			t.Errorf("expected '%s', got '%s'", content, result)
		}
	})
}

func TestGetStringOrFile(t *testing.T) {
	t.Run("Direct value when no file specified", func(t *testing.T) {
		v := viper.New()
		v.Set("password", "direct-password")

		result, err := GetStringOrFile(v, "password")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "direct-password" {
			t.Errorf("expected 'direct-password', got '%s'", result)
		}
	})

	t.Run("File value takes precedence", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")
		if err := os.WriteFile(secretFile, []byte("file-password"), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		v := viper.New()
		v.Set("password", "direct-password")
		v.Set("password_file", secretFile)

		result, err := GetStringOrFile(v, "password")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "file-password" {
			t.Errorf("expected 'file-password', got '%s'", result)
		}
	})

	t.Run("Empty file falls back to direct value", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "empty.txt")
		if err := os.WriteFile(secretFile, []byte(""), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		v := viper.New()
		v.Set("password", "direct-password")
		v.Set("password_file", secretFile)

		result, err := GetStringOrFile(v, "password")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "direct-password" {
			t.Errorf("expected 'direct-password', got '%s'", result)
		}
	})

	t.Run("Nonexistent file returns error", func(t *testing.T) {
		v := viper.New()
		v.Set("password", "direct-password")
		v.Set("password_file", "/nonexistent/secret.txt")

		result, err := GetStringOrFile(v, "password")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
		if result != "" {
			t.Errorf("expected empty result on error, got '%s'", result)
		}
	})

	t.Run("No value or file returns empty", func(t *testing.T) {
		v := viper.New()

		result, err := GetStringOrFile(v, "password")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})

	t.Run("File with whitespace is trimmed", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")
		if err := os.WriteFile(secretFile, []byte("  password123\n"), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		v := viper.New()
		v.Set("password_file", secretFile)

		result, err := GetStringOrFile(v, "password")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "password123" {
			t.Errorf("expected 'password123', got '%s'", result)
		}
	})
}

func TestResolveFiles(t *testing.T) {
	t.Run("Simple struct with file field", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")
		if err := os.WriteFile(secretFile, []byte("secret-value"), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		type Config struct {
			Password     string `mapstructure:"password"`
			PasswordFile string `mapstructure:"password_file"`
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

	t.Run("Multiple file fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		passwordFile := filepath.Join(tmpDir, "password.txt")
		tokenFile := filepath.Join(tmpDir, "token.txt")

		if err := os.WriteFile(passwordFile, []byte("my-password"), 0600); err != nil {
			t.Fatalf("failed to write password file: %v", err)
		}
		if err := os.WriteFile(tokenFile, []byte("my-token"), 0600); err != nil {
			t.Fatalf("failed to write token file: %v", err)
		}

		type Config struct {
			Password     string `mapstructure:"password"`
			PasswordFile string `mapstructure:"password_file"`
			Token        string `mapstructure:"token"`
			TokenFile    string `mapstructure:"token_file"`
		}

		config := &Config{
			PasswordFile: passwordFile,
			TokenFile:    tokenFile,
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Password != "my-password" {
			t.Errorf("expected 'my-password', got '%s'", config.Password)
		}
		if config.Token != "my-token" {
			t.Errorf("expected 'my-token', got '%s'", config.Token)
		}
	})

	t.Run("Nested struct with file fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")
		if err := os.WriteFile(secretFile, []byte("nested-secret"), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		type DatabaseConfig struct {
			Password     string `mapstructure:"password"`
			PasswordFile string `mapstructure:"password_file"`
		}

		type Config struct {
			Database DatabaseConfig `mapstructure:"database"`
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
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")
		if err := os.WriteFile(secretFile, []byte("pointer-secret"), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		type DatabaseConfig struct {
			Password     string `mapstructure:"password"`
			PasswordFile string `mapstructure:"password_file"`
		}

		type Config struct {
			Database *DatabaseConfig `mapstructure:"database"`
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

	t.Run("Nil nested pointer struct", func(t *testing.T) {
		type DatabaseConfig struct {
			Password     string `mapstructure:"password"`
			PasswordFile string `mapstructure:"password_file"`
		}

		type Config struct {
			Database *DatabaseConfig `mapstructure:"database"`
		}

		config := &Config{
			Database: nil,
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Skip field with tag", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")
		if err := os.WriteFile(secretFile, []byte("secret-value"), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		type Config struct {
			CertPath     string `mapstructure:"cert_path" fileconfig:"skip"`
			CertPathFile string `mapstructure:"cert_path_file"`
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

	t.Run("Direct value is not overwritten if file field is empty", func(t *testing.T) {
		type Config struct {
			Password     string `mapstructure:"password"`
			PasswordFile string `mapstructure:"password_file"`
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
			Password     string `mapstructure:"password"`
			PasswordFile string `mapstructure:"password_file"`
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
		tmpDir := t.TempDir()
		secret1File := filepath.Join(tmpDir, "secret1.txt")
		secret2File := filepath.Join(tmpDir, "secret2.txt")

		if err := os.WriteFile(secret1File, []byte("secret-one"), 0600); err != nil {
			t.Fatalf("failed to write secret1 file: %v", err)
		}
		if err := os.WriteFile(secret2File, []byte("secret-two"), 0600); err != nil {
			t.Fatalf("failed to write secret2 file: %v", err)
		}

		type ProviderConfig struct {
			APIKey     string `mapstructure:"api_key"`
			APIKeyFile string `mapstructure:"api_key_file"`
		}

		type Config struct {
			Providers map[string]ProviderConfig `mapstructure:"providers"`
		}

		config := &Config{
			Providers: map[string]ProviderConfig{
				"provider1": {
					APIKeyFile: secret1File,
				},
				"provider2": {
					APIKeyFile: secret2File,
				},
			},
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Providers["provider1"].APIKey != "secret-one" {
			t.Errorf("expected 'secret-one', got '%s'", config.Providers["provider1"].APIKey)
		}
		if config.Providers["provider2"].APIKey != "secret-two" {
			t.Errorf("expected 'secret-two', got '%s'", config.Providers["provider2"].APIKey)
		}
	})

	t.Run("Map of pointer structs", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")

		if err := os.WriteFile(secretFile, []byte("map-pointer-secret"), 0600); err != nil {
			t.Fatalf("failed to write secret file: %v", err)
		}

		type ProviderConfig struct {
			APIKey     string `mapstructure:"api_key"`
			APIKeyFile string `mapstructure:"api_key_file"`
		}

		type Config struct {
			Providers map[string]*ProviderConfig `mapstructure:"providers"`
		}

		config := &Config{
			Providers: map[string]*ProviderConfig{
				"provider1": {
					APIKeyFile: secretFile,
				},
			},
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Providers["provider1"].APIKey != "map-pointer-secret" {
			t.Errorf("expected 'map-pointer-secret', got '%s'", config.Providers["provider1"].APIKey)
		}
	})

	t.Run("Empty map", func(t *testing.T) {
		type ProviderConfig struct {
			APIKey     string `mapstructure:"api_key"`
			APIKeyFile string `mapstructure:"api_key_file"`
		}

		type Config struct {
			Providers map[string]ProviderConfig `mapstructure:"providers"`
		}

		config := &Config{
			Providers: map[string]ProviderConfig{},
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Nil map", func(t *testing.T) {
		type ProviderConfig struct {
			APIKey     string `mapstructure:"api_key"`
			APIKeyFile string `mapstructure:"api_key_file"`
		}

		type Config struct {
			Providers map[string]ProviderConfig `mapstructure:"providers"`
		}

		config := &Config{
			Providers: nil,
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Non-string file field is ignored", func(t *testing.T) {
		type Config struct {
			Port     int `mapstructure:"port"`
			PortFile int `mapstructure:"port_file"`
		}

		config := &Config{
			PortFile: 8080,
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Port != 0 {
			t.Errorf("expected Port to remain 0, got %d", config.Port)
		}
	})

	t.Run("File with only whitespace does not overwrite", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "whitespace.txt")
		if err := os.WriteFile(secretFile, []byte("   \n\n  "), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		type Config struct {
			Password     string `mapstructure:"password"`
			PasswordFile string `mapstructure:"password_file"`
		}

		config := &Config{
			Password:     "existing-value",
			PasswordFile: secretFile,
		}

		err := ResolveFiles(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Empty content should not overwrite existing value
		if config.Password != "existing-value" {
			t.Errorf("expected 'existing-value' to be preserved, got '%s'", config.Password)
		}
	})
}

func TestUnmarshalWithFileResolution(t *testing.T) {
	t.Run("Unmarshal and resolve files", func(t *testing.T) {
		tmpDir := t.TempDir()
		secretFile := filepath.Join(tmpDir, "secret.txt")
		if err := os.WriteFile(secretFile, []byte("file-secret"), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		type Config struct {
			Username     string `mapstructure:"username"`
			Password     string `mapstructure:"password"`
			PasswordFile string `mapstructure:"password_file"`
		}

		v := viper.New()
		v.Set("username", "admin")
		v.Set("password", "direct-password")
		v.Set("password_file", secretFile)

		var config Config
		err := UnmarshalWithFileResolution(v, &config)
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

	t.Run("Nested config with file resolution", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPasswordFile := filepath.Join(tmpDir, "db_password.txt")
		apiKeyFile := filepath.Join(tmpDir, "api_key.txt")

		if err := os.WriteFile(dbPasswordFile, []byte("db-secret"), 0600); err != nil {
			t.Fatalf("failed to write db password file: %v", err)
		}
		if err := os.WriteFile(apiKeyFile, []byte("api-secret"), 0600); err != nil {
			t.Fatalf("failed to write api key file: %v", err)
		}

		type DatabaseConfig struct {
			Host         string `mapstructure:"host"`
			Password     string `mapstructure:"password"`
			PasswordFile string `mapstructure:"password_file"`
		}

		type APIConfig struct {
			Key     string `mapstructure:"key"`
			KeyFile string `mapstructure:"key_file"`
		}

		type Config struct {
			Database DatabaseConfig `mapstructure:"database"`
			API      APIConfig      `mapstructure:"api"`
		}

		v := viper.New()
		v.Set("database.host", "localhost")
		v.Set("database.password_file", dbPasswordFile)
		v.Set("api.key_file", apiKeyFile)

		var config Config
		err := UnmarshalWithFileResolution(v, &config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Database.Host != "localhost" {
			t.Errorf("expected host 'localhost', got '%s'", config.Database.Host)
		}
		if config.Database.Password != "db-secret" {
			t.Errorf("expected database password 'db-secret', got '%s'", config.Database.Password)
		}
		if config.API.Key != "api-secret" {
			t.Errorf("expected API key 'api-secret', got '%s'", config.API.Key)
		}
	})

	t.Run("Unmarshal error is propagated", func(t *testing.T) {
		type Config struct {
			Port int `mapstructure:"port"`
		}

		v := viper.New()
		v.Set("port", "not-a-number")

		var config Config
		err := UnmarshalWithFileResolution(v, &config)
		if err == nil {
			t.Error("expected unmarshal error")
		}
	})

	t.Run("File resolution error is propagated", func(t *testing.T) {
		type Config struct {
			Password     string `mapstructure:"password"`
			PasswordFile string `mapstructure:"password_file"`
		}

		v := viper.New()
		v.Set("password_file", "/nonexistent/file.txt")

		var config Config
		err := UnmarshalWithFileResolution(v, &config)
		if err == nil {
			t.Error("expected file resolution error")
		}
	})

	t.Run("Complete workflow from YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		passwordFile := filepath.Join(tmpDir, "password.txt")
		tokenFile := filepath.Join(tmpDir, "token.txt")

		if err := os.WriteFile(passwordFile, []byte("yaml-password"), 0600); err != nil {
			t.Fatalf("failed to write password file: %v", err)
		}
		if err := os.WriteFile(tokenFile, []byte("yaml-token"), 0600); err != nil {
			t.Fatalf("failed to write token file: %v", err)
		}

		yamlContent := `
username: testuser
password_file: ` + passwordFile + `
api:
  endpoint: https://api.example.com
  token_file: ` + tokenFile + `
`
		if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		type APIConfig struct {
			Endpoint  string `mapstructure:"endpoint"`
			Token     string `mapstructure:"token"`
			TokenFile string `mapstructure:"token_file"`
		}

		type Config struct {
			Username     string    `mapstructure:"username"`
			Password     string    `mapstructure:"password"`
			PasswordFile string    `mapstructure:"password_file"`
			API          APIConfig `mapstructure:"api"`
		}

		v := viper.New()
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			t.Fatalf("failed to read config: %v", err)
		}

		var config Config
		err := UnmarshalWithFileResolution(v, &config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.Username != "testuser" {
			t.Errorf("expected username 'testuser', got '%s'", config.Username)
		}
		if config.Password != "yaml-password" {
			t.Errorf("expected password 'yaml-password', got '%s'", config.Password)
		}
		if config.API.Endpoint != "https://api.example.com" {
			t.Errorf("expected endpoint 'https://api.example.com', got '%s'", config.API.Endpoint)
		}
		if config.API.Token != "yaml-token" {
			t.Errorf("expected token 'yaml-token', got '%s'", config.API.Token)
		}
	})
}
