package configs

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config representa toda la configuración de la aplicación
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	JWT      JWTConfig      `json:"jwt"`
	CORS     CORSConfig     `json:"cors"`
	Upload   UploadConfig   `json:"upload"`
	Email    EmailConfig    `json:"email"`
	External ExternalConfig `json:"external"`
	Features FeatureConfig  `json:"features"`
	Logger   LoggerConfig   `json:"logger"`
}

// ServerConfig representa la configuración del servidor
type ServerConfig struct {
	Port     string `json:"port"`
	Host     string `json:"host"`
	Mode     string `json:"mode"`
	LogLevel string `json:"log_level"`
}

// DatabaseConfig representa la configuración de la base de datos
type DatabaseConfig struct {
	URL             string `json:"-"`
	databaseURLSet  bool
	Host            string `json:"host"`
	Port            string `json:"port"`
	User            string `json:"user"`
	Password        string `json:"password"`
	DBName          string `json:"db_name"`
	SSLMode         string `json:"ssl_mode"`
	TimeZone        string `json:"timezone"`
	LogLevel        string `json:"log_level"`
	AutoMigrate     bool   `json:"auto_migrate"`
	MigrationEngine string `json:"migration_engine"` // gorm | golang-migrate
	MigrationPath   string `json:"migration_path"`   // e.g. file://migrations
	MaxIdleConns    int    `json:"max_idle_conns"`
	MaxOpenConns    int    `json:"max_open_conns"`
	ConnMaxLifetime int    `json:"conn_max_lifetime"`
}

// JWTConfig representa la configuración JWT
type JWTConfig struct {
	SecretKey             string        `json:"-"` // No exponer en JSON
	ExpiresIn             time.Duration `json:"expires_in"`
	RefreshTokenExpiresIn time.Duration `json:"refresh_token_expires_in"`
}

// CORSConfig representa la configuración CORS
type CORSConfig struct {
	AllowedOrigins []string `json:"allowed_origins"`
	AllowedMethods []string `json:"allowed_methods"`
	AllowedHeaders []string `json:"allowed_headers"`
}

// UploadConfig representa la configuración de subida de archivos
type UploadConfig struct {
	MaxSize      int64    `json:"max_size"`
	AllowedTypes []string `json:"allowed_types"`
	Path         string   `json:"path"`
}

// EmailConfig representa la configuración de email
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"-"` // No exponer en JSON
	FromAddress  string `json:"from_address"`
	Enabled      bool   `json:"enabled"`
}

// ExternalConfig representa configuraciones de servicios externos
type ExternalConfig struct {
	PlaidClientID    string           `json:"-"` // No exponer en JSON
	PlaidSecret      string           `json:"-"` // No exponer en JSON
	PlaidEnvironment string           `json:"plaid_environment"`
	SentryDSN        string           `json:"-"` // No exponer en JSON
	SentryTestSecret string           `json:"-"` // Secreto para POST /debug/sentry-test (solo desarrollo)
	WebhookSecret    string           `json:"-"` // Secreto para autenticación de webhooks
	OCRAPIKey        string           `json:"-"` // API key del proveedor OCR
	OCRProviderURL   string           `json:"ocr_provider_url"`
	Gmail            GmailOAuthConfig `json:"-"`
}

// GmailOAuthConfig credenciales y secretos para Gmail API (OAuth2).
type GmailOAuthConfig struct {
	ClientID           string `json:"-"`
	ClientSecret       string `json:"-"`
	RedirectURL        string `json:"-"`
	TokenEncryptionKey string `json:"-"` // hex o base64 (32 bytes raw) para AES-GCM
	OAuthStateSecret   string `json:"-"` // HMAC del parámetro state
	// GmailQueryQuery query por defecto para listar mensajes (ej. newer_than:90d)
	DefaultListQuery string `json:"-"`
}

// FeatureConfig representa configuraciones de características
type FeatureConfig struct {
	EnableSwagger       bool `json:"enable_swagger"`
	EnableMetrics       bool `json:"enable_metrics"`
	EnableProfiler      bool `json:"enable_profiler"`
	EnableDebugRoutes   bool `json:"enable_debug_routes"`
	EnableNotifications bool `json:"enable_notifications"`
}

// LoggerConfig representa la configuración del logger
type LoggerConfig struct {
	Level string `json:"level"`
}

// globalConfig almacena la configuración global
var globalConfig *Config

// Load carga la configuración desde variables de entorno
func Load() *Config {
	if globalConfig != nil {
		return globalConfig
	}

	config := &Config{
		Server: ServerConfig{
			Port:     getEnv("PORT", "8080"),        // Railway usa PORT por defecto
			Host:     getEnv("HOST", "0.0.0.0"),     // Railway necesita 0.0.0.0
			Mode:     getEnv("GIN_MODE", "release"), // release en producción; en Railway Dashboard verificar que no esté "debug"
			LogLevel: getEnv("LOG_LEVEL", "info"),
		},
		Database: loadDatabaseConfig(getEnv("GIN_MODE", "release")),
		JWT: JWTConfig{
			SecretKey:             getEnv("JWT_SECRET_KEY", "default-secret-key-change-in-production"),
			ExpiresIn:             time.Duration(getEnvAsInt("JWT_EXPIRES_IN", 86400)) * time.Second,             // 24 horas por defecto
			RefreshTokenExpiresIn: time.Duration(getEnvAsInt("REFRESH_TOKEN_EXPIRES_IN", 2592000)) * time.Second, // 30 días
		},
		CORS: CORSConfig{
			AllowedOrigins: getEnvAsStringSlice("CORS_ALLOWED_ORIGINS", []string{"*"}), // Railway permite todo por defecto, configurar en producción
			AllowedMethods: getEnvAsStringSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			AllowedHeaders: getEnvAsStringSlice("CORS_ALLOWED_HEADERS", []string{"Origin", "Content-Type", "Accept", "Authorization"}),
		},
		Upload: UploadConfig{
			MaxSize:      getEnvAsInt64("UPLOAD_MAX_SIZE", 10*1024*1024), // 10MB
			AllowedTypes: getEnvAsStringSlice("UPLOAD_ALLOWED_TYPES", []string{"image/jpeg", "image/png", "image/gif", "application/pdf"}),
			Path:         getEnv("UPLOAD_PATH", "./uploads"),
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:     getEnvAsInt("SMTP_PORT", 587),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromAddress:  getEnv("SMTP_FROM", "noreply@fintech.com"),
			Enabled:      getEnvAsBool("ENABLE_EMAIL_NOTIFICATIONS", false),
		},
		External: ExternalConfig{
			PlaidClientID:    getEnv("PLAID_CLIENT_ID", ""),
			PlaidSecret:      getEnv("PLAID_SECRET", ""),
			PlaidEnvironment: getEnv("PLAID_ENVIRONMENT", "sandbox"),
			SentryDSN:        getEnv("SENTRY_DSN", ""),
			SentryTestSecret: getEnv("SENTRY_TEST_SECRET", ""),
			WebhookSecret:    getEnv("WEBHOOK_SECRET", ""),
			OCRAPIKey:        getEnv("OCR_API_KEY", ""),
			OCRProviderURL:   getEnv("OCR_PROVIDER_URL", "https://api.ocr.space/parse/image"),
			Gmail: GmailOAuthConfig{
				ClientID:           getEnv("GMAIL_CLIENT_ID", ""),
				ClientSecret:       getEnv("GMAIL_CLIENT_SECRET", ""),
				RedirectURL:        getEnv("GMAIL_REDIRECT_URL", ""),
				TokenEncryptionKey: getEnv("TOKEN_ENCRYPTION_KEY", ""),
				OAuthStateSecret:   getEnv("OAUTH_STATE_SECRET", ""),
				DefaultListQuery:   getEnv("GMAIL_DEFAULT_LIST_QUERY", "newer_than:90d"),
			},
		},
		Features: FeatureConfig{
			EnableSwagger:       getEnvAsBool("ENABLE_SWAGGER", true),
			EnableMetrics:       getEnvAsBool("ENABLE_METRICS", true),
			EnableProfiler:      getEnvAsBool("ENABLE_PROFILER", true),
			EnableDebugRoutes:   getEnvAsBool("ENABLE_DEBUG_ROUTES", true),
			EnableNotifications: getEnvAsBool("ENABLE_NOTIFICATIONS", true),
		},
		Logger: LoggerConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	globalConfig = config
	return config
}

// Get retorna la configuración global
func Get() *Config {
	if globalConfig == nil {
		return Load()
	}
	return globalConfig
}

// IsProduction verifica si estamos en modo producción
func (c *Config) IsProduction() bool {
	return c.Server.Mode == "release"
}

// IsDevelopment verifica si estamos en modo desarrollo
func (c *Config) IsDevelopment() bool {
	return c.Server.Mode == "debug"
}

// Validate valida la configuración
func (c *Config) Validate() error {
	if c.IsProduction() {
		jwtSecret := strings.TrimSpace(c.JWT.SecretKey)
		if jwtSecret == "" || jwtSecret == "default-secret-key-change-in-production" || len([]byte(jwtSecret)) < 32 {
			return fmt.Errorf("JWT_SECRET_KEY must be configured with at least 32 bytes in release mode")
		}

		if c.Database.databaseURLSet || c.Database.URL != "" {
			if !isValidProductionDatabaseURL(c.Database.URL) {
				return fmt.Errorf("DATABASE_URL must be a complete PostgreSQL connection URL in release mode")
			}
		} else if err := validateProductionDatabaseConfig(c.Database); err != nil {
			return err
		}
	}

	g := c.External.Gmail
	if g.ClientID != "" && g.ClientSecret != "" && g.RedirectURL != "" {
		if strings.TrimSpace(g.TokenEncryptionKey) == "" {
			return fmt.Errorf(
				"TOKEN_ENCRYPTION_KEY es obligatoria cuando Gmail OAuth está configurado (GMAIL_CLIENT_ID, GMAIL_CLIENT_SECRET, GMAIL_REDIRECT_URL); no uses el JWT como sustituto",
			)
		}
		if strings.TrimSpace(g.OAuthStateSecret) == "" {
			return fmt.Errorf(
				"OAUTH_STATE_SECRET es obligatoria cuando Gmail OAuth está configurado; no uses el JWT como sustituto",
			)
		}
	}

	return nil
}

// loadDatabaseConfig carga la configuración de base de datos
// Soporta tanto DATABASE_URL (Railway, Heroku) como variables individuales
func loadDatabaseConfig(mode string) DatabaseConfig {
	// DATABASE_URL is the sole database source whenever it is supplied, including whitespace-only values.
	if rawDatabaseURL, databaseURLSet := os.LookupEnv("DATABASE_URL"); databaseURLSet {
		databaseURL := strings.TrimSpace(rawDatabaseURL)
		if config, err := parseDatabaseURL(databaseURL); err == nil {
			config.URL = databaseURL
			config.databaseURLSet = true
			return config
		}
		return DatabaseConfig{URL: databaseURL, databaseURLSet: true}
	}

	hostDefault, portDefault, userDefault, passwordDefault, nameDefault := "", "", "", "", ""
	if mode != "release" {
		hostDefault, portDefault, userDefault, passwordDefault, nameDefault = "localhost", "5432", "postgres", "postgres", "fintech_db"
	}

	// Use local defaults only outside release when DATABASE_URL is absent.
	return DatabaseConfig{
		Host:            getEnv("DB_HOST", hostDefault),
		Port:            getEnv("DB_PORT", portDefault),
		User:            getEnv("DB_USER", userDefault),
		Password:        getEnv("DB_PASSWORD", passwordDefault),
		DBName:          getEnv("DB_NAME", nameDefault),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		TimeZone:        getEnv("DB_TIMEZONE", "America/Bogota"),
		LogLevel:        getEnv("DB_LOG_LEVEL", "error"),
		AutoMigrate:     getEnvAsBool("DB_AUTO_MIGRATE", true),
		MigrationEngine: getEnv("DB_MIGRATION_ENGINE", "gorm"),
		MigrationPath:   getEnv("DB_MIGRATION_PATH", "file://migrations"),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
		ConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 60),
	}
}

// parseDatabaseURL parsea una URL de base de datos PostgreSQL
// Formato: postgres://user:password@host:port/dbname?sslmode=require
func parseDatabaseURL(databaseURL string) (DatabaseConfig, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return DatabaseConfig{}, err
	}

	config := DatabaseConfig{
		Host:            u.Hostname(),
		Port:            u.Port(),
		DBName:          strings.TrimPrefix(u.Path, "/"),
		TimeZone:        getEnv("DB_TIMEZONE", "America/Bogota"),
		LogLevel:        getEnv("DB_LOG_LEVEL", "error"),
		AutoMigrate:     getEnvAsBool("DB_AUTO_MIGRATE", true),
		MigrationEngine: getEnv("DB_MIGRATION_ENGINE", "gorm"),
		MigrationPath:   getEnv("DB_MIGRATION_PATH", "file://migrations"),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
		ConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 60),
	}

	// Extraer usuario y contraseña
	if u.User != nil {
		config.User = u.User.Username()
		if password, hasPassword := u.User.Password(); hasPassword {
			config.Password = password
		}
	}

	// Configurar puerto por defecto si no está especificado
	if config.Port == "" {
		config.Port = "5432"
	}

	// Parsear parámetros de query
	queryParams := u.Query()

	// SSL Mode
	if sslMode := queryParams.Get("sslmode"); sslMode != "" {
		config.SSLMode = sslMode
	} else {
		// Railway y la mayoría de providers usan SSL por defecto
		config.SSLMode = "require"
	}

	return config, nil
}

func isValidProductionDatabaseURL(databaseURL string) bool {
	u, err := url.Parse(strings.TrimSpace(databaseURL))
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Hostname() == "" {
		return false
	}
	if u.User == nil || strings.TrimSpace(u.User.Username()) == "" {
		return false
	}
	password, hasPassword := u.User.Password()
	password = strings.TrimSpace(password)
	return hasPassword && password != "" && password != "postgres" && strings.TrimSpace(strings.Trim(u.Path, "/")) != ""
}

func validateProductionDatabaseConfig(database DatabaseConfig) error {
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "DB_HOST", value: database.Host},
		{name: "DB_PORT", value: database.Port},
		{name: "DB_USER", value: database.User},
		{name: "DB_PASSWORD", value: database.Password},
		{name: "DB_NAME", value: database.DBName},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s must be configured in release mode", field.name)
		}
	}
	if strings.TrimSpace(database.Password) == "postgres" {
		return fmt.Errorf("DB_PASSWORD must not use the default value in release mode")
	}
	return nil
}

// Funciones auxiliares para obtener variables de entorno

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
