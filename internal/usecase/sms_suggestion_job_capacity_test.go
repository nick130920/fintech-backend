package usecase

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"testing/synctest"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/webapi"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

func TestStartSMSBatchSuggestionJob_Accepts500WithoutPersistingMessages(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		uc, _, calls, messages, seen := newCompleteSuggestionBatchTestUseCase(t, 500)
		jobs := &suggestionProcessingJobRepo{}
		uc.suggestionJobRepo = jobs
		response, err := uc.StartSMSBatchSuggestionJob(7, messages)
		if err != nil || response == nil || response.JobID == "" || response.Status != entity.BudgetSuggestionJobPending {
			t.Fatalf("500-message submission failed: %v", err)
		}
		// The worker cannot run until this goroutine yields. Mutating the caller's
		// slice now proves the worker owns an independent, ephemeral copy.
		for i := range messages {
			messages[i].Body = "caller changed input"
		}
		synctest.Wait()
		if jobs.createCalls != 1 || jobs.createdMessagesJSON != "" || jobs.job.MessagesJSON != "" {
			t.Error("job must be created once with no persisted raw SMS")
		}
		if *calls != 20 || len(seen) != 500 {
			t.Errorf("calls = %d, distinct processed messages = %d; want 20 and 500", *calls, len(seen))
		}
		for _, count := range seen {
			if count != 1 {
				t.Error("in-process message processed more than once")
			}
		}
		if len(jobs.statuses) != 2 || jobs.statuses[0] != entity.BudgetSuggestionJobProcessing || jobs.statuses[1] != entity.BudgetSuggestionJobCompleted {
			t.Fatalf("saved statuses = %v; want processing then completed", jobs.statuses)
		}
		var result dto.AnalyzeSMSBatchResponse
		if json.Unmarshal([]byte(jobs.job.ResultJSON), &result) != nil || result.Suggestions.TotalExpense3m != 5000 || jobs.job.ErrorMessage != "" {
			t.Error("ephemeral worker must persist complete aggregated results only")
		}
	})
}

func TestStartSMSBatchSuggestionJob_Rejects501BeforeCreateOrWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		uc, _, calls := newSuggestionProcessingTestUseCase(t)
		jobs := &suggestionProcessingJobRepo{createErr: errors.New("Create must not run")}
		uc.suggestionJobRepo = jobs
		response, err := uc.StartSMSBatchSuggestionJob(7, syntheticSuggestionMessages(501))
		var appErr *apperrors.AppError
		if response != nil || !errors.As(err, &appErr) || appErr.Code != apperrors.ErrCodeInvalidRequest || appErr.StatusCode != http.StatusBadRequest || appErr.Internal != nil {
			t.Errorf("501-message submission must return a safe invalid-request 400; got %T", err)
		}
		synctest.Wait()
		if jobs.createCalls != 0 || *calls != 0 || len(jobs.statuses) != 0 {
			t.Errorf("rejected submission did work: creates=%d, provider calls=%d, status writes=%d", jobs.createCalls, *calls, len(jobs.statuses))
		}
	})
}

func TestStartSMSBatchSuggestionJob_EmptyInputCompletesWithoutWork(t *testing.T) {
	uc, _, calls := newSuggestionProcessingTestUseCase(t)
	jobs := &suggestionProcessingJobRepo{}
	uc.suggestionJobRepo = jobs
	response, err := uc.StartSMSBatchSuggestionJob(7, nil)
	if err != nil || response == nil || response.Status != entity.BudgetSuggestionJobCompleted || response.Suggestions == nil || len(response.Suggestions.ByCategory) != 0 {
		t.Fatalf("empty input should complete immediately: %v", err)
	}
	if jobs.createCalls != 0 || *calls != 0 {
		t.Error("empty input must not persist or call provider")
	}
}

func TestGetSMSBatchSuggestionJobStatus_OwnershipAndResults(t *testing.T) {
	uc, _, _ := newSuggestionProcessingTestUseCase(t)
	jobs := &suggestionProcessingJobRepo{job: &entity.BudgetSuggestionJob{ID: "synthetic-job", UserID: 7}}
	uc.suggestionJobRepo = jobs
	for _, status := range []string{entity.BudgetSuggestionJobPending, entity.BudgetSuggestionJobProcessing, entity.BudgetSuggestionJobCompleted, entity.BudgetSuggestionJobFailed} {
		t.Run(status, func(t *testing.T) {
			jobs.job.Status = status
			jobs.job.ErrorMessage = "safe failure"
			jobs.job.ResultJSON = `{"suggestions":{"total_expense_3m":10,"by_category":[]}}`
			response, err := uc.GetSMSBatchSuggestionJobStatus(8, "synthetic-job")
			if response != nil || !errors.Is(err, apperrors.ErrNotFound) {
				t.Error("non-owner must not see job status or results")
			}
			response, err = uc.GetSMSBatchSuggestionJobStatus(7, "synthetic-job")
			if err != nil || response == nil || response.Status != status {
				t.Fatalf("owner status lookup failed: %v", err)
			}
			if (response.Suggestions != nil) != (status == entity.BudgetSuggestionJobCompleted) {
				t.Error("only completed jobs expose suggestions")
			}
			if (response.Error != "") != (status == entity.BudgetSuggestionJobFailed) {
				t.Error("only failed jobs expose errors")
			}
		})
	}
}

func TestSMSBatchSuggestionJob_ZeroExpenseSuccess(t *testing.T) {
	uc, _, calls := newSuggestionProcessingTestUseCase(t, &webapi.BatchSMSBudgetResponse{})
	jobs := &suggestionProcessingJobRepo{job: &entity.BudgetSuggestionJob{Status: entity.BudgetSuggestionJobPending, ErrorMessage: "stale error"}}
	uc.suggestionJobRepo = jobs
	uc.runSMSBatchSuggestionJob("synthetic-job", 7, syntheticSuggestionMessages(1))
	if *calls != 1 || jobs.job.Status != entity.BudgetSuggestionJobCompleted || jobs.job.ErrorMessage != "" || jobs.job.ResultJSON == "" {
		t.Error("processed zero-expense chunk must complete and clear stale errors")
	}
}
