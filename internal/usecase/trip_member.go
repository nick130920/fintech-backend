package usecase

import (
	"strings"
	"time"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
	"github.com/nick130920/fintech-backend/pkg/logger"
	"github.com/rs/zerolog"
)

const defaultInvitationDays = 7

// TripMemberUseCase contiene la lógica de gestión de miembros e invitaciones
type TripMemberUseCase struct {
	tripRepo       repo.TripRepo
	memberRepo     repo.TripMemberRepo
	invitationRepo repo.TripInvitationRepo
	userRepo       repo.UserRepo
	logger         zerolog.Logger
}

// NewTripMemberUseCase construye el use case
func NewTripMemberUseCase(
	tripRepo repo.TripRepo,
	memberRepo repo.TripMemberRepo,
	invitationRepo repo.TripInvitationRepo,
	userRepo repo.UserRepo,
) *TripMemberUseCase {
	return &TripMemberUseCase{
		tripRepo:       tripRepo,
		memberRepo:     memberRepo,
		invitationRepo: invitationRepo,
		userRepo:       userRepo,
		logger:         logger.Get().With().Str("usecase", "TripMember").Logger(),
	}
}

// ListMembers retorna los miembros del viaje
func (uc *TripMemberUseCase) ListMembers(userID, tripID uint) ([]dto.TripMemberResponse, error) {
	if err := uc.assertAccess(tripID, userID); err != nil {
		return nil, err
	}

	members, err := uc.memberRepo.GetByTrip(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	out := make([]dto.TripMemberResponse, 0, len(members))
	for _, m := range members {
		out = append(out, mapMember(m))
	}
	return out, nil
}

// AddGhostMember agrega un participante "fantasma" (no usuario registrado)
func (uc *TripMemberUseCase) AddGhostMember(userID, tripID uint, req *dto.AddTripMemberRequest) (*dto.TripMemberResponse, error) {
	if err := uc.assertCanManage(tripID, userID); err != nil {
		return nil, err
	}

	role := entity.TripMemberRoleMember
	if req.Role != "" {
		role = entity.TripMemberRole(req.Role)
	}

	member := &entity.TripMember{
		TripID:      tripID,
		DisplayName: req.DisplayName,
		Email:       req.Email,
		AvatarURL:   req.AvatarURL,
		Role:        role,
		IsGhost:     true,
		JoinedAt:    time.Now(),
	}

	if err := uc.memberRepo.Create(member); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	out := mapMember(member)
	return &out, nil
}

// UpdateMember modifica datos de un miembro existente
func (uc *TripMemberUseCase) UpdateMember(userID, tripID, memberID uint, req *dto.UpdateTripMemberRequest) (*dto.TripMemberResponse, error) {
	if err := uc.assertCanManage(tripID, userID); err != nil {
		return nil, err
	}

	member, err := uc.memberRepo.GetByTripAndID(tripID, memberID)
	if err != nil || member == nil {
		return nil, ErrTripMemberNotFound
	}

	if req.DisplayName != nil {
		member.DisplayName = *req.DisplayName
	}
	if req.AvatarURL != nil {
		member.AvatarURL = *req.AvatarURL
	}
	if req.Role != nil && member.Role != entity.TripMemberRoleOwner {
		member.Role = entity.TripMemberRole(*req.Role)
	}

	if err := uc.memberRepo.Update(member); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	out := mapMember(member)
	return &out, nil
}

// RemoveMember expulsa a un miembro si no tiene deudas pendientes
func (uc *TripMemberUseCase) RemoveMember(userID, tripID, memberID uint) error {
	if err := uc.assertCanManage(tripID, userID); err != nil {
		return err
	}

	member, err := uc.memberRepo.GetByTripAndID(tripID, memberID)
	if err != nil || member == nil {
		return ErrTripMemberNotFound
	}
	if member.IsOwner() {
		return apperrors.ErrConflict.WithDetails("no puedes eliminar al owner del viaje")
	}

	hasDebts, err := uc.memberRepo.HasPendingSplits(memberID)
	if err != nil {
		return apperrors.ErrInternal.WithInternal(err)
	}
	if hasDebts {
		return ErrTripMemberHasDebts
	}

	return uc.memberRepo.Delete(memberID)
}

// CreateInvitation genera un link de invitación (token+expiración)
func (uc *TripMemberUseCase) CreateInvitation(userID, tripID uint, req *dto.CreateInvitationRequest) (*dto.InvitationResponse, error) {
	if err := uc.assertCanManage(tripID, userID); err != nil {
		return nil, err
	}

	role := entity.TripMemberRoleMember
	if req.Role != "" {
		role = entity.TripMemberRole(req.Role)
	}

	days := req.ExpiresInDay
	if days <= 0 {
		days = defaultInvitationDays
	}

	token, err := generateInvitationToken()
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	invitation := &entity.TripInvitation{
		TripID:          tripID,
		Token:           token,
		Email:           strings.TrimSpace(req.Email),
		Role:            role,
		CreatedByUserID: userID,
		ExpiresAt:       time.Now().Add(time.Duration(days) * 24 * time.Hour),
	}
	if err := uc.invitationRepo.Create(invitation); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	return mapInvitation(invitation), nil
}

// ListInvitations lista invitaciones activas del viaje
func (uc *TripMemberUseCase) ListInvitations(userID, tripID uint) ([]*dto.InvitationResponse, error) {
	if err := uc.assertCanManage(tripID, userID); err != nil {
		return nil, err
	}
	invitations, err := uc.invitationRepo.GetByTrip(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	out := make([]*dto.InvitationResponse, 0, len(invitations))
	for _, inv := range invitations {
		out = append(out, mapInvitation(inv))
	}
	return out, nil
}

// AcceptInvitation procesa la aceptación de una invitación por un usuario real
func (uc *TripMemberUseCase) AcceptInvitation(userID uint, token string) (*dto.TripMemberResponse, error) {
	invitation, err := uc.invitationRepo.GetByToken(token)
	if err != nil || invitation == nil {
		return nil, ErrTripInvitationInvalid
	}
	if !invitation.IsValid() {
		return nil, ErrTripInvitationInvalid
	}

	user, err := uc.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return nil, apperrors.ErrUserNotFound
	}

	if existing, err := uc.memberRepo.GetByTripAndUser(invitation.TripID, userID); err == nil && existing != nil {
		return nil, apperrors.ErrConflict.WithDetails("ya eres miembro de este viaje")
	}

	member := &entity.TripMember{
		TripID:      invitation.TripID,
		UserID:      &userID,
		DisplayName: user.FullName(),
		Role:        invitation.Role,
		IsGhost:     false,
		JoinedAt:    time.Now(),
	}
	if user.Email != "" {
		member.Email = &user.Email
	}
	if err := uc.memberRepo.Create(member); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	invitation.MarkUsed(userID)
	if err := uc.invitationRepo.Update(invitation); err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	out := mapMember(member)
	return &out, nil
}

func (uc *TripMemberUseCase) assertAccess(tripID, userID uint) error {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return ErrTripNotFound
	}
	if trip.OwnerUserID == userID {
		return nil
	}
	member, err := uc.memberRepo.GetByTripAndUser(tripID, userID)
	if err != nil || member == nil {
		return ErrTripPermissionDenied
	}
	return nil
}

func (uc *TripMemberUseCase) assertCanManage(tripID, userID uint) error {
	trip, err := uc.tripRepo.GetByID(tripID)
	if err != nil || trip == nil {
		return ErrTripNotFound
	}
	if trip.OwnerUserID == userID {
		return nil
	}
	member, err := uc.memberRepo.GetByTripAndUser(tripID, userID)
	if err != nil || member == nil {
		return ErrTripPermissionDenied
	}
	if !member.CanManage() {
		return ErrTripPermissionDenied
	}
	return nil
}

func mapInvitation(inv *entity.TripInvitation) *dto.InvitationResponse {
	out := &dto.InvitationResponse{
		ID:        inv.ID,
		TripID:    inv.TripID,
		Token:     inv.Token,
		Email:     inv.Email,
		Role:      string(inv.Role),
		ExpiresAt: inv.ExpiresAt,
		UsedAt:    inv.UsedAt,
		CreatedAt: inv.CreatedAt,
	}
	return out
}
