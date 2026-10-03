package configs

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestLoadUploadConfigLeavesObjectStorageOptional(t *testing.T) {
	cfg := loadUploadConfigForTest(t, "debug", nil)

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v; want nil when object storage is absent", err)
	}
	if cfg.Upload.Path != "./uploads" {
		t.Errorf("Upload.Path = %q; want legacy default %q", cfg.Upload.Path, "./uploads")
	}
	if cfg.Upload.ObjectStorageUseTLS != true {
		t.Errorf("ObjectStorageUseTLS = %t; want true", cfg.Upload.ObjectStorageUseTLS)
	}
	if cfg.Upload.ObjectStorageForcePathStyle != true {
		t.Errorf("ObjectStorageForcePathStyle = %t; want true", cfg.Upload.ObjectStorageForcePathStyle)
	}
	if cfg.Upload.SignedURLTTLSeconds != 900 {
		t.Errorf("SignedURLTTLSeconds = %d; want 900", cfg.Upload.SignedURLTTLSeconds)
	}
}

func TestLoadUploadConfigExplicitObjectStorageSettingsRequireCompleteConfiguration(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "TLS disabled alone",
			env:  map[string]string{"OBJECT_STORAGE_USE_TLS": "false"},
		},
		{
			name: "path style enabled alone",
			env:  map[string]string{"OBJECT_STORAGE_FORCE_PATH_STYLE": "true"},
		},
		{
			name: "default TTL alone",
			env:  map[string]string{"UPLOAD_SIGNED_URL_TTL_SECONDS": "900"},
		},
		{
			name: "empty bucket",
			env:  map[string]string{"OBJECT_STORAGE_BUCKET": ""},
		},
		{
			name: "whitespace bucket",
			env:  map[string]string{"OBJECT_STORAGE_BUCKET": " \t "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadUploadConfigForTest(t, "debug", tt.env)
			assertSafeValidationError(t, cfg.Validate(), "OBJECT_STORAGE_ENDPOINT", "")
		})
	}
}

func TestLoadUploadConfigAcceptsCompleteObjectStorageConfiguration(t *testing.T) {
	cfg := loadUploadConfigForTest(t, "debug", completeObjectStorageEnv())

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v; want nil", err)
	}
	if cfg.Upload.ObjectStorageEndpoint != "https://objects.example.test" {
		t.Errorf("ObjectStorageEndpoint = %q; want configured endpoint", cfg.Upload.ObjectStorageEndpoint)
	}

	encoded, err := json.Marshal(cfg.Upload)
	if err != nil {
		t.Fatalf("json.Marshal(Upload) error = %v", err)
	}
	for _, secret := range []string{"fixture-access-key", "fixture-secret-key"} {
		if strings.Contains(string(encoded), secret) {
			t.Errorf("json.Marshal(Upload) = %s; must not include secret %q", encoded, secret)
		}
	}
}

func TestConfigValidateRejectsProgrammaticInvalidUploadValues(t *testing.T) {
	tests := []struct {
		name string
		edit func(*UploadConfig)
		want string
	}{
		{
			name: "zero max size",
			edit: func(upload *UploadConfig) { upload.MaxSize = 0 },
			want: "UPLOAD_MAX_SIZE",
		},
		{
			name: "negative max size",
			edit: func(upload *UploadConfig) { upload.MaxSize = -1 },
			want: "UPLOAD_MAX_SIZE",
		},
		{
			name: "empty allowed types",
			edit: func(upload *UploadConfig) { upload.AllowedTypes = []string{} },
			want: "UPLOAD_ALLOWED_TYPES",
		},
		{
			name: "blank allowed type",
			edit: func(upload *UploadConfig) { upload.AllowedTypes = []string{" "} },
			want: "UPLOAD_ALLOWED_TYPES",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validReleaseConfig()
			cfg.Upload = validUploadConfig()
			tt.edit(&cfg.Upload)

			assertSafeValidationError(t, cfg.Validate(), tt.want, "")
		})
	}
}

func TestConfigValidateReleaseRejectsProgrammaticInsecureObjectStorage(t *testing.T) {
	tests := []struct {
		name string
		edit func(*UploadConfig)
		want string
	}{
		{
			name: "HTTP endpoint",
			edit: func(upload *UploadConfig) { upload.ObjectStorageEndpoint = "http://objects.example.test" },
			want: "OBJECT_STORAGE_ENDPOINT",
		},
		{
			name: "TLS disabled",
			edit: func(upload *UploadConfig) { upload.ObjectStorageUseTLS = false },
			want: "OBJECT_STORAGE_USE_TLS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validReleaseConfig()
			cfg.Upload = validObjectStorageUploadConfig()
			tt.edit(&cfg.Upload)

			assertSafeValidationError(t, cfg.Validate(), tt.want, "")
		})
	}
}

func TestLoadUploadConfigRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name string
		mode string
		env  map[string]string
		want string
	}{
		{
			name: "partial object storage configuration",
			mode: "debug",
			env:  map[string]string{"OBJECT_STORAGE_BUCKET": "uploads"},
			want: "OBJECT_STORAGE_ENDPOINT",
		},
		{
			name: "malformed endpoint",
			mode: "debug",
			env: mergeObjectStorageEnv(map[string]string{
				"OBJECT_STORAGE_ENDPOINT": "https://",
			}),
			want: "OBJECT_STORAGE_ENDPOINT",
		},
		{
			name: "partial credentials",
			mode: "debug",
			env: mergeObjectStorageEnv(map[string]string{
				"OBJECT_STORAGE_SECRET_ACCESS_KEY": "",
			}),
			want: "OBJECT_STORAGE_SECRET_ACCESS_KEY",
		},
		{
			name: "non-positive signed URL TTL",
			mode: "debug",
			env:  map[string]string{"UPLOAD_SIGNED_URL_TTL_SECONDS": "0"},
			want: "UPLOAD_SIGNED_URL_TTL_SECONDS",
		},
		{
			name: "signed URL TTL exceeds maximum",
			mode: "debug",
			env:  map[string]string{"UPLOAD_SIGNED_URL_TTL_SECONDS": "3601"},
			want: "UPLOAD_SIGNED_URL_TTL_SECONDS",
		},
		{
			name: "non-positive upload size",
			mode: "debug",
			env:  map[string]string{"UPLOAD_MAX_SIZE": "0"},
			want: "UPLOAD_MAX_SIZE",
		},
		{
			name: "empty MIME entry",
			mode: "debug",
			env:  map[string]string{"UPLOAD_ALLOWED_TYPES": "image/jpeg, "},
			want: "UPLOAD_ALLOWED_TYPES",
		},
		{
			name: "release requires HTTPS endpoint",
			mode: "release",
			env: mergeObjectStorageEnv(map[string]string{
				"OBJECT_STORAGE_ENDPOINT": "http://objects.example.test",
			}),
			want: "OBJECT_STORAGE_ENDPOINT",
		},
		{
			name: "release requires TLS",
			mode: "release",
			env: mergeObjectStorageEnv(map[string]string{
				"OBJECT_STORAGE_USE_TLS": "false",
			}),
			want: "OBJECT_STORAGE_USE_TLS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadUploadConfigForTest(t, tt.mode, tt.env)
			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() error = nil; want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Validate() error = %q; want it to name %s", err, tt.want)
			}
		})
	}
}

func loadUploadConfigForTest(t *testing.T, mode string, env map[string]string) *Config {
	t.Helper()
	previous := globalConfig
	globalConfig = nil
	t.Cleanup(func() { globalConfig = previous })

	for _, key := range []string{
		"OBJECT_STORAGE_ENDPOINT",
		"OBJECT_STORAGE_REGION",
		"OBJECT_STORAGE_BUCKET",
		"OBJECT_STORAGE_ACCESS_KEY_ID",
		"OBJECT_STORAGE_SECRET_ACCESS_KEY",
		"OBJECT_STORAGE_USE_TLS",
		"OBJECT_STORAGE_FORCE_PATH_STYLE",
		"UPLOAD_SIGNED_URL_TTL_SECONDS",
		"UPLOAD_MAX_SIZE",
		"UPLOAD_ALLOWED_TYPES",
		"UPLOAD_PATH",
	} {
		unsetUploadEnv(t, key)
	}
	for key, value := range env {
		t.Setenv(key, value)
	}

	t.Setenv("GIN_MODE", mode)
	if mode == "release" {
		t.Setenv("JWT_SECRET_KEY", validJWTSecret)
		t.Setenv("DATABASE_URL", fixtureURL)
	}

	return Load()
}

func unsetUploadEnv(t *testing.T, key string) {
	t.Helper()
	value, wasSet := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%q): %v", key, err)
	}
	t.Cleanup(func() {
		if wasSet {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	})
}

func validUploadConfig() UploadConfig {
	return UploadConfig{
		MaxSize:             10 * 1024 * 1024,
		AllowedTypes:        []string{"image/jpeg"},
		Path:                "./uploads",
		ObjectStorageUseTLS: true,
		SignedURLTTLSeconds: 900,
	}
}

func validObjectStorageUploadConfig() UploadConfig {
	config := validUploadConfig()
	config.ObjectStorageEndpoint = "https://objects.example.test"
	config.ObjectStorageRegion = "us-east-1"
	config.ObjectStorageBucket = "uploads"
	config.ObjectStorageAccessKeyID = "fixture-access-key"
	config.ObjectStorageSecretAccessKey = "fixture-secret-key"
	config.ObjectStorageForcePathStyle = true
	return config
}

func completeObjectStorageEnv() map[string]string {
	return map[string]string{
		"OBJECT_STORAGE_ENDPOINT":          "https://objects.example.test",
		"OBJECT_STORAGE_REGION":            "us-east-1",
		"OBJECT_STORAGE_BUCKET":            "uploads",
		"OBJECT_STORAGE_ACCESS_KEY_ID":     "fixture-access-key",
		"OBJECT_STORAGE_SECRET_ACCESS_KEY": "fixture-secret-key",
		"OBJECT_STORAGE_USE_TLS":           "true",
		"OBJECT_STORAGE_FORCE_PATH_STYLE":  "true",
		"UPLOAD_SIGNED_URL_TTL_SECONDS":    "900",
	}
}

func mergeObjectStorageEnv(overrides map[string]string) map[string]string {
	env := completeObjectStorageEnv()
	for key, value := range overrides {
		env[key] = value
	}
	return env
}
