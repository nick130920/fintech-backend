package usecase

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/nick130920/proyecto-fintech/internal/controller/http/v1/dto"
	"github.com/nick130920/proyecto-fintech/internal/entity"
	"github.com/nick130920/proyecto-fintech/internal/usecase/repo"
	"github.com/nick130920/proyecto-fintech/pkg/auth"
)

// UserUseCase contiene la lógica de negocio para usuarios
type UserUseCase struct {
	userRepo   repo.UserRepo
	jwtManager *auth.JWTManager
}

// NewUserUseCase crea una nueva instancia de UserUseCase
func NewUserUseCase(userRepo repo.UserRepo, jwtManager *auth.JWTManager) *UserUseCase {
	return &UserUseCase{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

// Register registra un nuevo usuario
func (uc *UserUseCase) Register(req *dto.CreateUserRequest) (*dto.LoginResponse, error) {
	// Verificar si el usuario ya existe
	_, err := uc.userRepo.GetByEmail(req.Email)
	if err == nil {
		return nil, errors.New("user already exists")
	}

	// Hash de la contraseña
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Crear la entidad usuario
	newUser := &entity.User{
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Email:      req.Email,
		Phone:      req.Phone,
		Password:   string(hashedPassword),
		IsActive:   true,
		IsVerified: false,
	}

	// Parsear fecha de nacimiento si se proporciona
	if req.DateOfBirth != "" {
		if dob, err := time.Parse("2006-01-02", req.DateOfBirth); err == nil {
			newUser.DateOfBirth = &dob
		}
	}

	// Guardar usuario
	if err := uc.userRepo.Create(newUser); err != nil {
		return nil, err
	}

	// Generar tokens
	accessToken, refreshToken, err := uc.generateTokens(newUser)
	if err != nil {
		return nil, err
	}

	// Actualizar última conexión
	uc.userRepo.UpdateLastLogin(newUser.ID)

	return &dto.LoginResponse{
		User:         newUser.ToPublic(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600, // 1 hora
	}, nil
}

// Login autentica un usuario
func (uc *UserUseCase) Login(req *dto.LoginRequest) (*dto.LoginResponse, error) {
	// Buscar usuario por email
	user, err := uc.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Verificar contraseña
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Verificar que el usuario esté activo
	if !user.IsAccountActive() {
		return nil, errors.New("account disabled")
	}

	// Generar tokens
	accessToken, refreshToken, err := uc.generateTokens(user)
	if err != nil {
		return nil, err
	}

	// Actualizar última conexión
	uc.userRepo.UpdateLastLogin(user.ID)

	return &dto.LoginResponse{
		User:         user.ToPublic(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600, // 1 hora
	}, nil
}

// GetByID obtiene un usuario por ID
func (uc *UserUseCase) GetByID(id uint) (*entity.User, error) {
	return uc.userRepo.GetByID(id)
}

// Update actualiza un usuario
func (uc *UserUseCase) Update(id uint, req *dto.UpdateUserRequest) (*entity.User, error) {
	user, err := uc.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Actualizar campos si se proporcionan
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Locale != "" {
		user.Locale = req.Locale
	}
	if req.Timezone != "" {
		user.Timezone = req.Timezone
	}

	// Parsear fecha de nacimiento si se proporciona
	if req.DateOfBirth != "" {
		if dob, err := time.Parse("2006-01-02", req.DateOfBirth); err == nil {
			user.DateOfBirth = &dob
		}
	}

	if err := uc.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

// RefreshToken renueva un token de acceso
func (uc *UserUseCase) RefreshToken(refreshToken string) (*dto.TokenResponse, error) {
	// Validar el refresh token
	userID, email, err := uc.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Buscar el usuario
	user, err := uc.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Verificar que el email coincida
	if user.Email != email {
		return nil, errors.New("invalid refresh token")
	}

	// Verificar que el usuario esté activo
	if !user.IsAccountActive() {
		return nil, errors.New("account disabled")
	}

	// Generar nuevo access token
	accessToken, err := uc.jwtManager.GenerateToken(user.ID, user.Email, user.FirstName, user.LastName)
	if err != nil {
		return nil, err
	}

	// Generar nuevo refresh token
	newRefreshToken, err := uc.jwtManager.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    3600, // 1 hora
	}, nil
}

// Deactivate desactiva una cuenta de usuario
func (uc *UserUseCase) Deactivate(id uint) error {
	return uc.userRepo.SetActive(id, false)
}

// Activate activa una cuenta de usuario
func (uc *UserUseCase) Activate(id uint) error {
	return uc.userRepo.SetActive(id, true)
}

// VerifyAccount marca una cuenta como verificada
func (uc *UserUseCase) VerifyAccount(id uint) error {
	return uc.userRepo.SetVerified(id, true)
}

// generateTokens genera tokens JWT para el usuario
func (uc *UserUseCase) generateTokens(user *entity.User) (string, string, error) {
	// Generar access token
	accessToken, err := uc.jwtManager.GenerateToken(user.ID, user.Email, user.FirstName, user.LastName)
	if err != nil {
		return "", "", err
	}

	// Generar refresh token
	refreshToken, err := uc.jwtManager.GenerateRefreshToken(user.ID, user.Email)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}
