package repo

import (
	"time"

	"github.com/nick130920/fintech-backend/internal/entity"
)

// TripRepo define la interfaz de persistencia para viajes
type TripRepo interface {
	Create(trip *entity.Trip) error
	GetByID(id uint) (*entity.Trip, error)
	GetByIDDeep(id uint) (*entity.Trip, error)
	GetByUser(userID uint, status string) ([]*entity.Trip, error)
	Update(trip *entity.Trip) error
	Delete(id uint) error
	UpdateTotals(tripID uint, estimatedTotal, spentTotal float64) error
	GetActiveOverlappingDates(userID uint, from, to time.Time) ([]*entity.Trip, error)
}

// TripMemberRepo define la interfaz de persistencia para miembros de viajes
type TripMemberRepo interface {
	Create(member *entity.TripMember) error
	GetByID(id uint) (*entity.TripMember, error)
	GetByTrip(tripID uint) ([]*entity.TripMember, error)
	GetByTripAndUser(tripID, userID uint) (*entity.TripMember, error)
	GetByTripAndID(tripID, memberID uint) (*entity.TripMember, error)
	Update(member *entity.TripMember) error
	Delete(id uint) error
	HasPendingSplits(memberID uint) (bool, error)
}

// TripInvitationRepo define la interfaz de persistencia para invitaciones a viajes
type TripInvitationRepo interface {
	Create(invitation *entity.TripInvitation) error
	GetByToken(token string) (*entity.TripInvitation, error)
	GetByTrip(tripID uint) ([]*entity.TripInvitation, error)
	Update(invitation *entity.TripInvitation) error
	Delete(id uint) error
}

// TripBudgetAllocationRepo define la interfaz de persistencia para asignaciones del presupuesto de viaje
type TripBudgetAllocationRepo interface {
	Create(allocation *entity.TripBudgetAllocation) error
	Upsert(allocation *entity.TripBudgetAllocation) error
	GetByID(id uint) (*entity.TripBudgetAllocation, error)
	GetByTrip(tripID uint) ([]*entity.TripBudgetAllocation, error)
	GetByTripAndCategory(tripID, categoryID uint) (*entity.TripBudgetAllocation, error)
	Update(allocation *entity.TripBudgetAllocation) error
	UpdateSpent(allocationID uint, spent float64) error
	Delete(id uint) error
}

// ExpenseSplitRepo define la interfaz de persistencia para splits de gastos
type ExpenseSplitRepo interface {
	Create(split *entity.ExpenseSplit) error
	GetByID(id uint) (*entity.ExpenseSplit, error)
	GetByExpense(expenseID uint) ([]*entity.ExpenseSplit, error)
	GetByMember(memberID uint) ([]*entity.ExpenseSplit, error)
	GetByTrip(tripID uint) ([]*entity.ExpenseSplit, error)
	Update(split *entity.ExpenseSplit) error
	DeleteByExpense(expenseID uint) error
	BatchCreate(splits []*entity.ExpenseSplit) error
}

// SettlementRepo define la interfaz de persistencia para settlements entre miembros
type SettlementRepo interface {
	Create(settlement *entity.Settlement) error
	GetByID(id uint) (*entity.Settlement, error)
	GetByTrip(tripID uint) ([]*entity.Settlement, error)
	Delete(id uint) error
}

// TripItineraryRepo define la interfaz de persistencia para items del itinerario
type TripItineraryRepo interface {
	Create(item *entity.TripItineraryItem) error
	GetByID(id uint) (*entity.TripItineraryItem, error)
	GetByTrip(tripID uint) ([]*entity.TripItineraryItem, error)
	Update(item *entity.TripItineraryItem) error
	Delete(id uint) error
	LinkExpense(itemID, expenseID uint) error
}
