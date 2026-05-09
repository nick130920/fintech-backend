package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/pkg/validator"
)

// TripHandler agrupa endpoints HTTP relacionados con el módulo de viajes
type TripHandler struct {
	tripUC      *usecase.TripUseCase
	memberUC    *usecase.TripMemberUseCase
	budgetUC    *usecase.TripBudgetUseCase
	expenseUC   *usecase.TripExpenseUseCase
	balanceUC   *usecase.TripBalanceUseCase
	settleUC    *usecase.SettlementUseCase
	itineraryUC *usecase.TripItineraryUseCase
	importUC    *usecase.TripImportUseCase
	reportUC    *usecase.TripReportUseCase
	validator   *validator.Validator
	logger      zerolog.Logger
}

// NewTripHandler construye el handler con sus dependencias
func NewTripHandler(
	tripUC *usecase.TripUseCase,
	memberUC *usecase.TripMemberUseCase,
	budgetUC *usecase.TripBudgetUseCase,
	expenseUC *usecase.TripExpenseUseCase,
	balanceUC *usecase.TripBalanceUseCase,
	settleUC *usecase.SettlementUseCase,
	itineraryUC *usecase.TripItineraryUseCase,
	importUC *usecase.TripImportUseCase,
	reportUC *usecase.TripReportUseCase,
	logger zerolog.Logger,
) *TripHandler {
	return &TripHandler{
		tripUC:      tripUC,
		memberUC:    memberUC,
		budgetUC:    budgetUC,
		expenseUC:   expenseUC,
		balanceUC:   balanceUC,
		settleUC:    settleUC,
		itineraryUC: itineraryUC,
		importUC:    importUC,
		reportUC:    reportUC,
		validator:   validator.New(),
		logger:      logger,
	}
}

// ----- Helpers HTTP -----

func (h *TripHandler) bindAndValidate(c *gin.Context, target interface{}) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Datos de entrada inválidos",
			Details: err.Error(),
		})
		return false
	}
	if err := h.validator.Validate(target); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "Error de validación",
			Details: err.Error(),
		})
		return false
	}
	return true
}

func (h *TripHandler) parseUintParam(c *gin.Context, key string) (uint, bool) {
	raw := c.Param(key)
	id, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "ID inválido",
			Details: err.Error(),
		})
		return 0, false
	}
	return uint(id), true
}

// ===== Endpoints del Trip =====

// CreateTrip godoc
// @Summary  Crear viaje
// @Tags     trips
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    request body dto.CreateTripRequest true "Datos del viaje"
// @Success  201 {object} dto.Response{data=dto.TripResponse}
// @Router   /api/v1/trips [post]
func (h *TripHandler) CreateTrip(c *gin.Context) {
	var req dto.CreateTripRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	userID := MustGetUserIDFromContext(c)
	resp, err := h.tripUC.CreateTrip(userID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.Response{Code: "SUCCESS", Message: "Viaje creado", Data: resp})
}

// ListTrips godoc
// @Summary  Listar viajes del usuario
// @Tags     trips
// @Produce  json
// @Security BearerAuth
// @Param    status query string false "Filtro por estado"
// @Success  200 {object} dto.Response{data=[]dto.TripResponse}
// @Router   /api/v1/trips [get]
func (h *TripHandler) ListTrips(c *gin.Context) {
	userID := MustGetUserIDFromContext(c)
	status := c.Query("status")
	trips, err := h.tripUC.ListTrips(userID, status)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: trips})
}

// GetTrip godoc
// @Summary  Obtener viaje por ID
// @Tags     trips
// @Produce  json
// @Security BearerAuth
// @Param    id path int true "Trip ID"
// @Success  200 {object} dto.Response{data=dto.TripResponse}
// @Router   /api/v1/trips/{id} [get]
func (h *TripHandler) GetTrip(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	resp, err := h.tripUC.GetTrip(MustGetUserIDFromContext(c), tripID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// UpdateTrip godoc
// @Summary  Actualizar viaje
// @Tags     trips
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id path int true "Trip ID"
// @Param    request body dto.UpdateTripRequest true "Cambios"
// @Success  200 {object} dto.Response{data=dto.TripResponse}
// @Router   /api/v1/trips/{id} [put]
func (h *TripHandler) UpdateTrip(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateTripRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	resp, err := h.tripUC.UpdateTrip(MustGetUserIDFromContext(c), tripID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// DeleteTrip godoc
// @Summary  Eliminar viaje
// @Tags     trips
// @Produce  json
// @Security BearerAuth
// @Param    id path int true "Trip ID"
// @Success  204 "No Content"
// @Router   /api/v1/trips/{id} [delete]
func (h *TripHandler) DeleteTrip(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := h.tripUC.DeleteTrip(MustGetUserIDFromContext(c), tripID); err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TripHandler) changeStatus(c *gin.Context, status entity.TripStatus) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	resp, err := h.tripUC.ChangeStatus(MustGetUserIDFromContext(c), tripID, status)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// StartTrip transiciona a estado active
func (h *TripHandler) StartTrip(c *gin.Context)    { h.changeStatus(c, entity.TripStatusActive) }
// CompleteTrip transiciona a estado completed
func (h *TripHandler) CompleteTrip(c *gin.Context) { h.changeStatus(c, entity.TripStatusCompleted) }
// CancelTrip transiciona a estado cancelled
func (h *TripHandler) CancelTrip(c *gin.Context)   { h.changeStatus(c, entity.TripStatusCancelled) }

// ===== Endpoints de miembros =====

// ListMembers lista miembros del viaje
func (h *TripHandler) ListMembers(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	members, err := h.memberUC.ListMembers(MustGetUserIDFromContext(c), tripID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: members})
}

// AddMember agrega un miembro fantasma
func (h *TripHandler) AddMember(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.AddTripMemberRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	resp, err := h.memberUC.AddGhostMember(MustGetUserIDFromContext(c), tripID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.Response{Code: "SUCCESS", Data: resp})
}

// UpdateMember actualiza datos de un miembro
func (h *TripHandler) UpdateMember(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	memberID, ok := h.parseUintParam(c, "memberId")
	if !ok {
		return
	}
	var req dto.UpdateTripMemberRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	resp, err := h.memberUC.UpdateMember(MustGetUserIDFromContext(c), tripID, memberID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// RemoveMember elimina un miembro
func (h *TripHandler) RemoveMember(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	memberID, ok := h.parseUintParam(c, "memberId")
	if !ok {
		return
	}
	if err := h.memberUC.RemoveMember(MustGetUserIDFromContext(c), tripID, memberID); err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CreateInvitation genera link de invitación
func (h *TripHandler) CreateInvitation(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.CreateInvitationRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	resp, err := h.memberUC.CreateInvitation(MustGetUserIDFromContext(c), tripID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.Response{Code: "SUCCESS", Data: resp})
}

// ListInvitations lista invitaciones generadas
func (h *TripHandler) ListInvitations(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	invitations, err := h.memberUC.ListInvitations(MustGetUserIDFromContext(c), tripID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: invitations})
}

// AcceptInvitation procesa la aceptación
func (h *TripHandler) AcceptInvitation(c *gin.Context) {
	var req dto.AcceptInvitationRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	resp, err := h.memberUC.AcceptInvitation(MustGetUserIDFromContext(c), req.Token)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// ===== Endpoints de presupuesto =====

// GetBudget devuelve las asignaciones del viaje
func (h *TripHandler) GetBudget(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	allocations, err := h.budgetUC.GetAllocations(MustGetUserIDFromContext(c), tripID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: allocations})
}

// UpsertBudget reemplaza las asignaciones
func (h *TripHandler) UpsertBudget(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpsertTripBudgetRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	allocations, err := h.budgetUC.UpsertBudget(MustGetUserIDFromContext(c), tripID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: allocations})
}

// ===== Endpoints de gastos =====

// ListExpenses devuelve gastos del viaje
func (h *TripHandler) ListExpenses(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	expenses, err := h.expenseUC.ListExpenses(MustGetUserIDFromContext(c), tripID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: expenses})
}

// CreateExpense crea un gasto del viaje
func (h *TripHandler) CreateExpense(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.CreateTripExpenseRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	resp, err := h.expenseUC.CreateExpense(MustGetUserIDFromContext(c), tripID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.Response{Code: "SUCCESS", Data: resp})
}

// UpdateExpense actualiza un gasto del viaje
func (h *TripHandler) UpdateExpense(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	expenseID, ok := h.parseUintParam(c, "expenseId")
	if !ok {
		return
	}
	var req dto.UpdateTripExpenseRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	resp, err := h.expenseUC.UpdateExpense(MustGetUserIDFromContext(c), tripID, expenseID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// DeleteExpense elimina un gasto
func (h *TripHandler) DeleteExpense(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	expenseID, ok := h.parseUintParam(c, "expenseId")
	if !ok {
		return
	}
	if err := h.expenseUC.DeleteExpense(MustGetUserIDFromContext(c), tripID, expenseID); err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ===== Endpoints de balance / settlements =====

// GetBalance devuelve balance + transferencias sugeridas
func (h *TripHandler) GetBalance(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	resp, err := h.balanceUC.Compute(MustGetUserIDFromContext(c), tripID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// ListSettlements devuelve los settlements del viaje
func (h *TripHandler) ListSettlements(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	resp, err := h.settleUC.List(MustGetUserIDFromContext(c), tripID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// CreateSettlement registra un pago entre miembros
func (h *TripHandler) CreateSettlement(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.CreateSettlementRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	resp, err := h.settleUC.Create(MustGetUserIDFromContext(c), tripID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.Response{Code: "SUCCESS", Data: resp})
}

// DeleteSettlement elimina un settlement
func (h *TripHandler) DeleteSettlement(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	settlementID, ok := h.parseUintParam(c, "settlementId")
	if !ok {
		return
	}
	if err := h.settleUC.Delete(MustGetUserIDFromContext(c), tripID, settlementID); err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ===== Endpoints de itinerario =====

// ListItinerary lista items del itinerario
func (h *TripHandler) ListItinerary(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	resp, err := h.itineraryUC.List(MustGetUserIDFromContext(c), tripID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// CreateItinerary crea un item de itinerario
func (h *TripHandler) CreateItinerary(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.CreateItineraryItemRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	resp, err := h.itineraryUC.Create(MustGetUserIDFromContext(c), tripID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.Response{Code: "SUCCESS", Data: resp})
}

// UpdateItinerary actualiza un item
func (h *TripHandler) UpdateItinerary(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	itemID, ok := h.parseUintParam(c, "itemId")
	if !ok {
		return
	}
	var req dto.UpdateItineraryItemRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	resp, err := h.itineraryUC.Update(MustGetUserIDFromContext(c), tripID, itemID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// DeleteItinerary elimina un item del itinerario
func (h *TripHandler) DeleteItinerary(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	itemID, ok := h.parseUintParam(c, "itemId")
	if !ok {
		return
	}
	if err := h.itineraryUC.Delete(MustGetUserIDFromContext(c), tripID, itemID); err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// LinkItineraryExpense vincula un gasto real al item
func (h *TripHandler) LinkItineraryExpense(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	itemID, ok := h.parseUintParam(c, "itemId")
	if !ok {
		return
	}
	var req dto.LinkItineraryExpenseRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	resp, err := h.itineraryUC.LinkExpense(MustGetUserIDFromContext(c), tripID, itemID, req.ExpenseID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// ===== Reportes y export =====

// GetReport devuelve el reporte estructurado
func (h *TripHandler) GetReport(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	resp, err := h.reportUC.Build(MustGetUserIDFromContext(c), tripID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// ExportReport descarga el reporte como CSV o PDF
func (h *TripHandler) ExportReport(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}

	format := usecase.ExportFormat(c.DefaultQuery("format", "pdf"))
	data, filename, contentType, err := h.reportUC.Export(MustGetUserIDFromContext(c), tripID, format)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, contentType, data)
}

// ===== Importación =====

// SuggestImport devuelve gastos candidatos para asignar al viaje
func (h *TripHandler) SuggestImport(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	resp, err := h.importUC.Suggest(MustGetUserIDFromContext(c), tripID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{Code: "SUCCESS", Data: resp})
}

// AssignImport asigna gastos al viaje en lote
func (h *TripHandler) AssignImport(c *gin.Context) {
	tripID, ok := h.parseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.AssignImportRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	count, err := h.importUC.Assign(MustGetUserIDFromContext(c), tripID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Gastos asignados al viaje",
		Data:    gin.H{"assigned": count},
	})
}
