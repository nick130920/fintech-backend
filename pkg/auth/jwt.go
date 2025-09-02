package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager maneja la generación y validación de tokens JWT
type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
}

// UserClaims representa los claims personalizados para nuestros tokens
type UserClaims struct {
	UserID    uint   `json:"user_id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	jwt.RegisteredClaims
}

// NewJWTManager crea una nueva instancia de JWTManager
func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     secretKey,
		tokenDuration: tokenDuration,
	}
}

// GenerateToken genera un nuevo token JWT para un usuario
func (manager *JWTManager) GenerateToken(userID uint, email, firstName, lastName string) (string, error) {
	claims := UserClaims{
		UserID:    userID,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(manager.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "fintech-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(manager.secretKey))
}

// ValidateToken valida un token JWT y retorna los claims
func (manager *JWTManager) ValidateToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Verificar que el método de firma sea HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("método de firma inválido")
			}
			return []byte(manager.secretKey), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token inválido")
	}

	return claims, nil
}

// GenerateRefreshToken genera un refresh token con mayor duración
func (manager *JWTManager) GenerateRefreshToken(userID uint, email string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   email,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour * 7)), // 7 días
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Issuer:    "fintech-api",
		ID:        string(rune(userID)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(manager.secretKey))
}

// ValidateRefreshToken valida un refresh token
func (manager *JWTManager) ValidateRefreshToken(tokenString string) (uint, string, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("método de firma inválido")
			}
			return []byte(manager.secretKey), nil
		},
	)

	if err != nil {
		return 0, "", err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return 0, "", errors.New("refresh token inválido")
	}

	// Convertir ID de usuario desde string
	userID := uint(claims.ID[0]) // Simplificado, en producción usar mejor conversión
	email := claims.Subject

	return userID, email, nil
}

// ExtractTokenFromHeader extrae el token del header Authorization
func ExtractTokenFromHeader(authHeader string) (string, error) {
	const bearerPrefix = "Bearer "

	if len(authHeader) < len(bearerPrefix) {
		return "", errors.New("header de autorización inválido")
	}

	if authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", errors.New("header de autorización debe empezar con 'Bearer '")
	}

	return authHeader[len(bearerPrefix):], nil
}

// TokenResponse representa la respuesta con tokens
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// CreateTokenResponse crea una respuesta de tokens estándar
func (manager *JWTManager) CreateTokenResponse(accessToken, refreshToken string) *TokenResponse {
	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(manager.tokenDuration.Seconds()),
		TokenType:    "Bearer",
	}
}
