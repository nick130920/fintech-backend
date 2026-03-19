package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

const smsBatchJobProcessTimeout = 12 * time.Minute

// StartSMSBatchSuggestionJob encola análisis por lotes; respuesta inmediata (evita conexión HTTP larga).
func (uc *BankNotificationPatternUseCase) StartSMSBatchSuggestionJob(userID uint, messages []dto.SMSMessageForAnalysis) (*dto.StartSMSBatchJobResponse, error) {
	if uc.suggestionJobRepo == nil {
		return nil, fmt.Errorf("suggestion job repository not configured")
	}

	if len(messages) == 0 {
		return &dto.StartSMSBatchJobResponse{
			Status: entity.BudgetSuggestionJobCompleted,
			Suggestions: &dto.BudgetSuggestions{
				ByCategory: []dto.BudgetSuggestionCategory{},
			},
		}, nil
	}

	raw, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("marshal messages: %w", err)
	}

	jobID := uuid.NewString()
	job := &entity.BudgetSuggestionJob{
		ID:           jobID,
		UserID:       userID,
		Status:       entity.BudgetSuggestionJobPending,
		MessagesJSON: string(raw),
	}
	if err := uc.suggestionJobRepo.Create(job); err != nil {
		return nil, fmt.Errorf("create suggestion job: %w", err)
	}

	msgsCopy := make([]dto.SMSMessageForAnalysis, len(messages))
	copy(msgsCopy, messages)
	go uc.runSMSBatchSuggestionJob(jobID, userID, msgsCopy)

	return &dto.StartSMSBatchJobResponse{
		JobID:  jobID,
		Status: entity.BudgetSuggestionJobPending,
	}, nil
}

func (uc *BankNotificationPatternUseCase) runSMSBatchSuggestionJob(jobID string, userID uint, messages []dto.SMSMessageForAnalysis) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("budget SMS batch job panic job_id=%s: %v", jobID, r)
			uc.markSMSBatchJobFailed(jobID, "error interno al procesar")
		}
	}()

	job, err := uc.suggestionJobRepo.FindByID(jobID)
	if err != nil || job == nil {
		return
	}
	job.Status = entity.BudgetSuggestionJobProcessing
	if err := uc.suggestionJobRepo.Save(job); err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), smsBatchJobProcessTimeout)
	defer cancel()

	result, procErr := uc.ProcessSMSBatchForSuggestions(ctx, userID, messages)

	job, err = uc.suggestionJobRepo.FindByID(jobID)
	if err != nil || job == nil {
		return
	}

	if procErr != nil {
		job.Status = entity.BudgetSuggestionJobFailed
		job.ErrorMessage = procErr.Error()
		job.ResultJSON = ""
		_ = uc.suggestionJobRepo.Save(job)
		return
	}

	b, err := json.Marshal(result)
	if err != nil {
		uc.markSMSBatchJobFailed(jobID, "error al serializar resultado")
		return
	}
	job.Status = entity.BudgetSuggestionJobCompleted
	job.ResultJSON = string(b)
	job.ErrorMessage = ""
	_ = uc.suggestionJobRepo.Save(job)
}

func (uc *BankNotificationPatternUseCase) markSMSBatchJobFailed(jobID, msg string) {
	job, err := uc.suggestionJobRepo.FindByID(jobID)
	if err != nil || job == nil {
		return
	}
	job.Status = entity.BudgetSuggestionJobFailed
	job.ErrorMessage = msg
	_ = uc.suggestionJobRepo.Save(job)
}

// GetSMSBatchSuggestionJobStatus devuelve estado para polling (solo el dueño del job).
func (uc *BankNotificationPatternUseCase) GetSMSBatchSuggestionJobStatus(userID uint, jobID string) (*dto.SMSBatchJobStatusResponse, error) {
	if uc.suggestionJobRepo == nil {
		return nil, fmt.Errorf("suggestion job repository not configured")
	}
	job, err := uc.suggestionJobRepo.FindByIDAndUser(jobID, userID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, apperrors.ErrNotFound
	}

	resp := &dto.SMSBatchJobStatusResponse{Status: job.Status}
	switch job.Status {
	case entity.BudgetSuggestionJobCompleted:
		var out dto.AnalyzeSMSBatchResponse
		if job.ResultJSON != "" {
			if err := json.Unmarshal([]byte(job.ResultJSON), &out); err != nil {
				resp.Status = entity.BudgetSuggestionJobFailed
				resp.Error = "resultado corrupto"
				return resp, nil
			}
		}
		resp.Suggestions = &out.Suggestions
	case entity.BudgetSuggestionJobFailed:
		resp.Error = job.ErrorMessage
	}
	return resp, nil
}
