package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/nick130920/proyecto-fintech/internal/entity"
	"github.com/nick130920/proyecto-fintech/internal/usecase/repo"
)

// UserPostgres implementa UserRepo usando PostgreSQL
type UserPostgres struct {
	db *gorm.DB
}

// NewUserPostgres crea una nueva instancia de UserPostgres
func NewUserPostgres(db *gorm.DB) repo.UserRepo {
	return &UserPostgres{db: db}
}

// Create crea un nuevo usuario
func (r *UserPostgres) Create(user *entity.User) error {
	return r.db.Create(user).Error
}

// GetByID obtiene un usuario por ID
func (r *UserPostgres) GetByID(id uint) (*entity.User, error) {
	var user entity.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail obtiene un usuario por email
func (r *UserPostgres) GetByEmail(email string) (*entity.User, error) {
	var user entity.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update actualiza un usuario
func (r *UserPostgres) Update(user *entity.User) error {
	return r.db.Save(user).Error
}

// Delete elimina un usuario (soft delete)
func (r *UserPostgres) Delete(id uint) error {
	return r.db.Delete(&entity.User{}, id).Error
}

// GetAll obtiene todos los usuarios con paginación
func (r *UserPostgres) GetAll(limit, offset int) ([]*entity.User, error) {
	var users []*entity.User
	err := r.db.Limit(limit).Offset(offset).Find(&users).Error
	return users, err
}

// Count obtiene el número total de usuarios
func (r *UserPostgres) Count() (int64, error) {
	var count int64
	err := r.db.Model(&entity.User{}).Count(&count).Error
	return count, err
}

// Search busca usuarios por término
func (r *UserPostgres) Search(term string, limit, offset int) ([]*entity.User, error) {
	var users []*entity.User
	searchPattern := "%" + term + "%"

	err := r.db.Where(
		"first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ?",
		searchPattern, searchPattern, searchPattern,
	).Limit(limit).Offset(offset).Find(&users).Error

	return users, err
}

// Exists verifica si un usuario existe
func (r *UserPostgres) Exists(id uint) (bool, error) {
	var count int64
	err := r.db.Model(&entity.User{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByEmail verifica si un email ya está registrado
func (r *UserPostgres) ExistsByEmail(email string) (bool, error) {
	var count int64
	err := r.db.Model(&entity.User{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SetActive activa/desactiva un usuario
func (r *UserPostgres) SetActive(id uint, active bool) error {
	return r.db.Model(&entity.User{}).Where("id = ?", id).Update("is_active", active).Error
}

// SetVerified marca un usuario como verificado/no verificado
func (r *UserPostgres) SetVerified(id uint, verified bool) error {
	return r.db.Model(&entity.User{}).Where("id = ?", id).Update("is_verified", verified).Error
}

// UpdateLastLogin actualiza la fecha de último login
func (r *UserPostgres) UpdateLastLogin(id uint) error {
	now := time.Now()
	return r.db.Model(&entity.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_login_at": &now,
		"login_count":   gorm.Expr("login_count + 1"),
	}).Error
}

// UpdatePassword actualiza la contraseña de un usuario
func (r *UserPostgres) UpdatePassword(id uint, hashedPassword string) error {
	return r.db.Model(&entity.User{}).Where("id = ?", id).Update("password", hashedPassword).Error
}

// GetActiveUsersWithPagination obtiene usuarios activos con paginación
func (r *UserPostgres) GetActiveUsersWithPagination(limit, offset int) ([]*entity.User, error) {
	var users []*entity.User
	err := r.db.Where("is_active = ?", true).Limit(limit).Offset(offset).Find(&users).Error
	return users, err
}

// GetVerifiedUsers obtiene usuarios verificados
func (r *UserPostgres) GetVerifiedUsers(limit, offset int) ([]*entity.User, error) {
	var users []*entity.User
	err := r.db.Where("is_verified = ?", true).Limit(limit).Offset(offset).Find(&users).Error
	return users, err
}

// GetUsersByDateRange obtiene usuarios creados en un rango de fechas
func (r *UserPostgres) GetUsersByDateRange(fromDate, toDate time.Time, limit, offset int) ([]*entity.User, error) {
	var users []*entity.User
	err := r.db.Where("created_at BETWEEN ? AND ?", fromDate, toDate).
		Limit(limit).Offset(offset).Find(&users).Error
	return users, err
}

// GetRecentUsers obtiene usuarios registrados recientemente
func (r *UserPostgres) GetRecentUsers(limit int) ([]*entity.User, error) {
	var users []*entity.User
	err := r.db.Order("created_at DESC").Limit(limit).Find(&users).Error
	return users, err
}

// CountActive obtiene el número de usuarios activos
func (r *UserPostgres) CountActive() (int64, error) {
	var count int64
	err := r.db.Model(&entity.User{}).Where("is_active = ?", true).Count(&count).Error
	return count, err
}

// CountVerified obtiene el número de usuarios verificados
func (r *UserPostgres) CountVerified() (int64, error) {
	var count int64
	err := r.db.Model(&entity.User{}).Where("is_verified = ?", true).Count(&count).Error
	return count, err
}

// GetUsersWithFilters obtiene usuarios con múltiples filtros
func (r *UserPostgres) GetUsersWithFilters(filters map[string]interface{}, limit, offset int) ([]*entity.User, error) {
	var users []*entity.User
	query := r.db.Model(&entity.User{})

	// Aplicar filtros dinámicamente
	for key, value := range filters {
		switch key {
		case "is_active":
			query = query.Where("is_active = ?", value)
		case "is_verified":
			query = query.Where("is_verified = ?", value)
		case "locale":
			query = query.Where("locale = ?", value)
		case "timezone":
			query = query.Where("timezone = ?", value)
		case "created_after":
			query = query.Where("created_at > ?", value)
		case "created_before":
			query = query.Where("created_at < ?", value)
		case "search":
			if searchTerm, ok := value.(string); ok && searchTerm != "" {
				searchPattern := "%" + searchTerm + "%"
				query = query.Where(
					"first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ?",
					searchPattern, searchPattern, searchPattern,
				)
			}
		}
	}

	err := query.Limit(limit).Offset(offset).Find(&users).Error
	return users, err
}

// BulkUpdateActive actualiza el estado activo de múltiples usuarios
func (r *UserPostgres) BulkUpdateActive(userIDs []uint, active bool) error {
	return r.db.Model(&entity.User{}).Where("id IN ?", userIDs).Update("is_active", active).Error
}

// BulkDelete elimina múltiples usuarios (soft delete)
func (r *UserPostgres) BulkDelete(userIDs []uint) error {
	return r.db.Where("id IN ?", userIDs).Delete(&entity.User{}).Error
}

// GetUserStats obtiene estadísticas básicas de un usuario
func (r *UserPostgres) GetUserStats(userID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Obtener usuario
	user, err := r.GetByID(userID)
	if err != nil {
		return nil, err
	}

	stats["account_created_days"] = int(time.Since(user.CreatedAt).Hours() / 24)
	stats["login_count"] = user.LoginCount
	stats["is_active"] = user.IsActive
	stats["is_verified"] = user.IsVerified

	if user.LastLoginAt != nil {
		stats["days_since_last_login"] = int(time.Since(*user.LastLoginAt).Hours() / 24)
	}

	// Aquí se pueden agregar más estadísticas relacionadas con presupuestos, gastos, etc.
	// TODO: Implementar cuando tengamos los repositorios de Budget y Expense

	return stats, nil
}

// CountUsers obtiene el número total de usuarios (implementación de interfaz)
func (r *UserPostgres) CountUsers() (int64, error) {
	return r.Count()
}

// GetActiveUsers obtiene usuarios activos (implementación de interfaz)
func (r *UserPostgres) GetActiveUsers() ([]*entity.User, error) {
	var users []*entity.User
	err := r.db.Where("is_active = ?", true).Find(&users).Error
	return users, err
}

// GetUsersRegisteredAfter obtiene usuarios registrados después de una fecha
func (r *UserPostgres) GetUsersRegisteredAfter(date time.Time) ([]*entity.User, error) {
	var users []*entity.User
	err := r.db.Where("created_at > ?", date).Find(&users).Error
	return users, err
}

// Cleanup elimina usuarios que nunca se verificaron después de X días
func (r *UserPostgres) Cleanup(daysOld int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysOld)
	return r.db.Unscoped().Where(
		"is_verified = ? AND created_at < ?",
		false, cutoffDate,
	).Delete(&entity.User{}).Error
}
