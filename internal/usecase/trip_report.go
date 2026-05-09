package usecase

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"time"

	"github.com/jung-kurt/gofpdf"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

// ExportFormat indica el formato de exportación del reporte
type ExportFormat string

const (
	ExportFormatCSV ExportFormat = "csv"
	ExportFormatPDF ExportFormat = "pdf"
)

// TripReportUseCase consolida toda la información de un viaje en un reporte
type TripReportUseCase struct {
	tripRepo       repo.TripRepo
	memberRepo     repo.TripMemberRepo
	allocationRepo repo.TripBudgetAllocationRepo
	expenseRepo    repo.ExpenseRepo
	splitRepo      repo.ExpenseSplitRepo
	settlementRepo repo.SettlementRepo
	itineraryRepo  repo.TripItineraryRepo
}

// NewTripReportUseCase construye el use case de reportes
func NewTripReportUseCase(
	tripRepo repo.TripRepo,
	memberRepo repo.TripMemberRepo,
	allocationRepo repo.TripBudgetAllocationRepo,
	expenseRepo repo.ExpenseRepo,
	splitRepo repo.ExpenseSplitRepo,
	settlementRepo repo.SettlementRepo,
	itineraryRepo repo.TripItineraryRepo,
) *TripReportUseCase {
	return &TripReportUseCase{
		tripRepo:       tripRepo,
		memberRepo:     memberRepo,
		allocationRepo: allocationRepo,
		expenseRepo:    expenseRepo,
		splitRepo:      splitRepo,
		settlementRepo: settlementRepo,
		itineraryRepo:  itineraryRepo,
	}
}

// Build genera la estructura completa del reporte para el cliente
func (uc *TripReportUseCase) Build(userID, tripID uint) (*dto.TripReportResponse, error) {
	trip, err := uc.tripRepo.GetByIDDeep(tripID)
	if err != nil || trip == nil {
		return nil, ErrTripNotFound
	}
	if !uc.hasAccess(trip, userID) {
		return nil, ErrTripPermissionDenied
	}

	allocations, err := uc.allocationRepo.GetByTrip(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	expenses, err := uc.expenseRepo.GetByTripID(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	settlements, err := uc.settlementRepo.GetByTrip(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}
	itinerary, err := uc.itineraryRepo.GetByTrip(tripID)
	if err != nil {
		return nil, apperrors.ErrInternal.WithInternal(err)
	}

	report := &dto.TripReportResponse{
		Trip:        *uc.tripSummary(trip),
		Settlements: make([]dto.SettlementResponse, 0, len(settlements)),
		GeneratedAt: time.Now(),
	}

	report.TotalsByCategory = uc.totalsByCategory(allocations, expenses, trip.PrimaryCurrency)
	report.TotalsByMember = uc.totalsByMember(trip, expenses)
	report.EstimatedVsReal = uc.estimatedVsReal(allocations, expenses, trip.PrimaryCurrency)
	report.ItineraryProgress = uc.itineraryProgress(itinerary)
	for _, settlement := range settlements {
		report.Settlements = append(report.Settlements, *mapSettlement(settlement))
	}

	return report, nil
}

// Export devuelve los bytes del reporte en el formato solicitado
func (uc *TripReportUseCase) Export(userID, tripID uint, format ExportFormat) ([]byte, string, string, error) {
	report, err := uc.Build(userID, tripID)
	if err != nil {
		return nil, "", "", err
	}

	switch format {
	case ExportFormatCSV:
		data := uc.exportCSV(report)
		filename := fmt.Sprintf("trip-%d-report.csv", tripID)
		return data, filename, "text/csv", nil
	case ExportFormatPDF:
		data, err := uc.exportPDF(report)
		if err != nil {
			return nil, "", "", apperrors.ErrInternal.WithInternal(err)
		}
		filename := fmt.Sprintf("trip-%d-report.pdf", tripID)
		return data, filename, "application/pdf", nil
	default:
		return nil, "", "", apperrors.ErrInvalidRequest.WithDetails("formato inválido (csv|pdf)")
	}
}

func (uc *TripReportUseCase) hasAccess(trip *entity.Trip, userID uint) bool {
	if trip.OwnerUserID == userID {
		return true
	}
	member, err := uc.memberRepo.GetByTripAndUser(trip.ID, userID)
	return err == nil && member != nil
}

func (uc *TripReportUseCase) tripSummary(trip *entity.Trip) *dto.TripResponse {
	resp := &dto.TripResponse{
		ID:              trip.ID,
		OwnerUserID:     trip.OwnerUserID,
		Name:            trip.Name,
		Destination:     trip.Destination,
		CountryCode:     trip.CountryCode,
		StartDate:       trip.StartDate,
		EndDate:         trip.EndDate,
		PrimaryCurrency: trip.PrimaryCurrency,
		Status:          string(trip.Status),
		CoverImageURL:   trip.CoverImageURL,
		EstimatedTotal:  trip.EstimatedTotal,
		SpentTotal:      trip.SpentTotal,
		RemainingAmount: trip.RemainingAmount(),
		ProgressPercent: trip.ProgressPercentage(),
		DaysTotal:       trip.DaysTotal(),
		DaysRemaining:   trip.DaysRemaining(),
		IsActiveNow:     trip.IsActiveNow(),
		Notes:           trip.Notes,
		CreatedAt:       trip.CreatedAt,
		UpdatedAt:       trip.UpdatedAt,
	}
	for _, m := range trip.Members {
		resp.Members = append(resp.Members, mapMember(&m))
	}
	return resp
}

func (uc *TripReportUseCase) totalsByCategory(allocations []*entity.TripBudgetAllocation, expenses []*entity.Expense, primary string) []dto.TripReportCategoryTotal {
	type accumulator struct {
		categoryName string
		estimated    float64
		spent        float64
	}
	totals := map[uint]*accumulator{}

	for _, allocation := range allocations {
		acc := totals[allocation.CategoryID]
		if acc == nil {
			acc = &accumulator{}
			totals[allocation.CategoryID] = acc
		}
		acc.estimated += allocation.EstimatedAmount
		if allocation.Category != nil {
			acc.categoryName = allocation.Category.Name
		}
	}

	for _, expense := range expenses {
		acc := totals[expense.CategoryID]
		if acc == nil {
			acc = &accumulator{}
			totals[expense.CategoryID] = acc
		}
		amount := expense.Amount
		if expense.Currency != primary && expense.ExchangeRate > 0 {
			amount = expense.Amount * expense.ExchangeRate
		}
		acc.spent += amount
		if acc.categoryName == "" {
			acc.categoryName = expense.Category.Name
		}
	}

	out := make([]dto.TripReportCategoryTotal, 0, len(totals))
	for categoryID, acc := range totals {
		out = append(out, dto.TripReportCategoryTotal{
			CategoryID:      categoryID,
			CategoryName:    acc.categoryName,
			EstimatedAmount: acc.estimated,
			SpentAmount:     acc.spent,
			Variance:        acc.spent - acc.estimated,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CategoryName < out[j].CategoryName
	})
	return out
}

func (uc *TripReportUseCase) totalsByMember(trip *entity.Trip, expenses []*entity.Expense) []dto.TripReportMemberTotal {
	memberByID := map[uint]*entity.TripMember{}
	for i := range trip.Members {
		m := trip.Members[i]
		memberByID[m.ID] = &m
	}

	paid := map[uint]float64{}
	owed := map[uint]float64{}
	for _, expense := range expenses {
		amount := expense.Amount
		if expense.Currency != trip.PrimaryCurrency && expense.ExchangeRate > 0 {
			amount = expense.Amount * expense.ExchangeRate
		}
		if expense.PaidByMemberID != nil {
			paid[*expense.PaidByMemberID] += amount
		}
		for _, split := range expense.Splits {
			share := split.ShareAmount
			if expense.Currency != trip.PrimaryCurrency && expense.ExchangeRate > 0 {
				share = split.ShareAmount * expense.ExchangeRate
			}
			owed[split.MemberID] += share
		}
	}

	out := make([]dto.TripReportMemberTotal, 0, len(memberByID))
	for memberID, member := range memberByID {
		entry := dto.TripReportMemberTotal{
			MemberID:   memberID,
			MemberName: member.DisplayName,
			Paid:       paid[memberID],
			Owed:       owed[memberID],
			Net:        paid[memberID] - owed[memberID],
		}
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].MemberName < out[j].MemberName })
	return out
}

func (uc *TripReportUseCase) estimatedVsReal(allocations []*entity.TripBudgetAllocation, expenses []*entity.Expense, primary string) dto.TripReportEstimateVsReal {
	estimated := 0.0
	for _, allocation := range allocations {
		estimated += allocation.EstimatedAmount
	}
	spent := 0.0
	for _, expense := range expenses {
		amount := expense.Amount
		if expense.Currency != primary && expense.ExchangeRate > 0 {
			amount = expense.Amount * expense.ExchangeRate
		}
		spent += amount
	}
	return dto.TripReportEstimateVsReal{
		EstimatedTotal: estimated,
		SpentTotal:     spent,
		Variance:       spent - estimated,
		OverBudget:     spent > estimated && estimated > 0,
	}
}

func (uc *TripReportUseCase) itineraryProgress(items []*entity.TripItineraryItem) []dto.TripReportItineraryProgress {
	out := make([]dto.TripReportItineraryProgress, 0, len(items))
	for _, item := range items {
		entry := dto.TripReportItineraryProgress{
			ItemID:        item.ID,
			Title:         item.Title,
			Day:           item.Day,
			EstimatedCost: item.EstimatedCost,
		}
		if item.Expense != nil {
			entry.ActualCost = item.Expense.Amount
			entry.Variance = item.Variance(item.Expense.Amount)
		}
		out = append(out, entry)
	}
	return out
}

// exportCSV exporta el reporte como CSV multi-sección
func (uc *TripReportUseCase) exportCSV(report *dto.TripReportResponse) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	w.Write([]string{"# Resumen del viaje"})
	w.Write([]string{"Nombre", report.Trip.Name})
	w.Write([]string{"Destino", report.Trip.Destination})
	w.Write([]string{"Estado", report.Trip.Status})
	w.Write([]string{"Inicio", report.Trip.StartDate.Format("2006-01-02")})
	w.Write([]string{"Fin", report.Trip.EndDate.Format("2006-01-02")})
	w.Write([]string{"Moneda", report.Trip.PrimaryCurrency})
	w.Write([]string{"Estimado total", fmt.Sprintf("%.2f", report.EstimatedVsReal.EstimatedTotal)})
	w.Write([]string{"Gastado total", fmt.Sprintf("%.2f", report.EstimatedVsReal.SpentTotal)})
	w.Write([]string{"Variación", fmt.Sprintf("%.2f", report.EstimatedVsReal.Variance)})
	w.Write([]string{})

	w.Write([]string{"# Totales por categoría"})
	w.Write([]string{"Categoría", "Estimado", "Gastado", "Variación"})
	for _, c := range report.TotalsByCategory {
		w.Write([]string{
			c.CategoryName,
			fmt.Sprintf("%.2f", c.EstimatedAmount),
			fmt.Sprintf("%.2f", c.SpentAmount),
			fmt.Sprintf("%.2f", c.Variance),
		})
	}
	w.Write([]string{})

	w.Write([]string{"# Totales por miembro"})
	w.Write([]string{"Miembro", "Pagó", "Debe", "Neto"})
	for _, m := range report.TotalsByMember {
		w.Write([]string{
			m.MemberName,
			fmt.Sprintf("%.2f", m.Paid),
			fmt.Sprintf("%.2f", m.Owed),
			fmt.Sprintf("%.2f", m.Net),
		})
	}
	w.Write([]string{})

	w.Write([]string{"# Settlements"})
	w.Write([]string{"De", "Para", "Monto", "Moneda", "Fecha"})
	for _, s := range report.Settlements {
		w.Write([]string{
			s.FromName,
			s.ToName,
			fmt.Sprintf("%.2f", s.Amount),
			s.Currency,
			s.PaidAt.Format("2006-01-02"),
		})
	}

	w.Flush()
	return buf.Bytes()
}

// exportPDF genera un PDF sencillo con las secciones principales del reporte
func (uc *TripReportUseCase) exportPDF(report *dto.TripReportResponse) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 20, 15)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 18)
	pdf.Cell(0, 10, fmt.Sprintf("Reporte de viaje: %s", report.Trip.Name))
	pdf.Ln(12)

	pdf.SetFont("Helvetica", "", 11)
	pdf.Cell(0, 6, fmt.Sprintf("Destino: %s", report.Trip.Destination))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Estado: %s", report.Trip.Status))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Periodo: %s - %s",
		report.Trip.StartDate.Format("2006-01-02"),
		report.Trip.EndDate.Format("2006-01-02")))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Moneda: %s", report.Trip.PrimaryCurrency))
	pdf.Ln(10)

	pdf.SetFont("Helvetica", "B", 13)
	pdf.Cell(0, 8, "Resumen estimado vs real")
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 11)
	pdf.Cell(0, 6, fmt.Sprintf("Estimado: %.2f %s", report.EstimatedVsReal.EstimatedTotal, report.Trip.PrimaryCurrency))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Gastado:  %.2f %s", report.EstimatedVsReal.SpentTotal, report.Trip.PrimaryCurrency))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Variación: %.2f %s", report.EstimatedVsReal.Variance, report.Trip.PrimaryCurrency))
	pdf.Ln(10)

	uc.pdfTable(pdf, "Totales por categoría",
		[]string{"Categoría", "Estimado", "Gastado", "Variación"},
		func() [][]string {
			rows := make([][]string, 0, len(report.TotalsByCategory))
			for _, c := range report.TotalsByCategory {
				rows = append(rows, []string{
					c.CategoryName,
					fmt.Sprintf("%.2f", c.EstimatedAmount),
					fmt.Sprintf("%.2f", c.SpentAmount),
					fmt.Sprintf("%.2f", c.Variance),
				})
			}
			return rows
		}(),
	)

	uc.pdfTable(pdf, "Totales por miembro",
		[]string{"Miembro", "Pagó", "Debe", "Neto"},
		func() [][]string {
			rows := make([][]string, 0, len(report.TotalsByMember))
			for _, m := range report.TotalsByMember {
				rows = append(rows, []string{
					m.MemberName,
					fmt.Sprintf("%.2f", m.Paid),
					fmt.Sprintf("%.2f", m.Owed),
					fmt.Sprintf("%.2f", m.Net),
				})
			}
			return rows
		}(),
	)

	if len(report.Settlements) > 0 {
		uc.pdfTable(pdf, "Pagos entre miembros",
			[]string{"De", "Para", "Monto", "Moneda", "Fecha"},
			func() [][]string {
				rows := make([][]string, 0, len(report.Settlements))
				for _, s := range report.Settlements {
					rows = append(rows, []string{
						s.FromName, s.ToName,
						fmt.Sprintf("%.2f", s.Amount),
						s.Currency,
						s.PaidAt.Format("2006-01-02"),
					})
				}
				return rows
			}(),
		)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (uc *TripReportUseCase) pdfTable(pdf *gofpdf.Fpdf, title string, headers []string, rows [][]string) {
	pdf.SetFont("Helvetica", "B", 13)
	pdf.Cell(0, 8, title)
	pdf.Ln(8)

	colWidths := uc.computeColWidths(pdf, len(headers))
	pdf.SetFont("Helvetica", "B", 10)
	for i, header := range headers {
		pdf.CellFormat(colWidths[i], 7, header, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Helvetica", "", 10)
	for _, row := range rows {
		for i, cell := range row {
			align := "L"
			if i > 0 {
				align = "R"
			}
			pdf.CellFormat(colWidths[i], 6, cell, "1", 0, align, false, 0, "")
		}
		pdf.Ln(-1)
	}
	pdf.Ln(4)
}

func (uc *TripReportUseCase) computeColWidths(pdf *gofpdf.Fpdf, n int) []float64 {
	pageWidth, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	available := pageWidth - left - right
	if n == 0 {
		return nil
	}
	widths := make([]float64, n)
	first := available * 0.35
	rest := (available - first) / float64(n-1)
	for i := range widths {
		if i == 0 {
			widths[i] = first
		} else {
			widths[i] = rest
		}
	}
	if n == 1 {
		widths[0] = available
	}
	return widths
}
