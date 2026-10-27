package usecase

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

type suggestionProcessingJobRepo struct {
	repo.BudgetSuggestionJobRepo
	job                 *entity.BudgetSuggestionJob
	statuses            []string
	createCalls         int
	createErr           error
	createdMessagesJSON string
}

func (r *suggestionProcessingJobRepo) Create(job *entity.BudgetSuggestionJob) error {
	r.createCalls++
	r.createdMessagesJSON = job.MessagesJSON
	r.job = job
	return r.createErr
}

func (r *suggestionProcessingJobRepo) FindByIDAndUser(id string, userID uint) (*entity.BudgetSuggestionJob, error) {
	if r.job != nil && r.job.ID == id && r.job.UserID == userID {
		return r.job, nil
	}
	return nil, nil
}

func (r *suggestionProcessingJobRepo) FindByID(string) (*entity.BudgetSuggestionJob, error) {
	return r.job, nil
}

func (r *suggestionProcessingJobRepo) Save(job *entity.BudgetSuggestionJob) error {
	r.statuses = append(r.statuses, job.Status)
	return nil
}

func TestSMSBatchSuggestionJob_ContextFailureMarksJobFailed(t *testing.T) {
	for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(sentinel.Error(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				providerErr := fmt.Errorf("synthetic private provider detail: %w", sentinel)
				uc, _, calls := newSuggestionChunkErrorTestUseCase(t, providerErr, providerErr)
				jobs := &suggestionProcessingJobRepo{job: &entity.BudgetSuggestionJob{
					Status: entity.BudgetSuggestionJobPending,
				}}
				uc.suggestionJobRepo = jobs

				uc.runSMSBatchSuggestionJob("synthetic-job", 1, syntheticSuggestionMessages(26))

				if *calls != 8 || len(jobs.statuses) != 2 || jobs.statuses[0] != entity.BudgetSuggestionJobProcessing || jobs.statuses[1] != entity.BudgetSuggestionJobFailed {
					t.Fatalf("calls = %d, saved statuses = %v; want processing then failed", *calls, jobs.statuses)
				}
				want := "all attempted AI chunks failed for SMS budget suggestions: " + sentinel.Error()
				if jobs.job.ResultJSON != "" || jobs.job.ErrorMessage != want {
					t.Errorf("result = %q, error = %q; want no result and sanitized error %q", jobs.job.ResultJSON, jobs.job.ErrorMessage, want)
				}
			})
		})
	}
}

func TestSMSBatchSuggestionJob_TotalAIFailureMarksJobFailed(t *testing.T) {
	uc, _, calls := newSuggestionProcessingTestUseCase(t, nil, nil)
	jobs := &suggestionProcessingJobRepo{job: &entity.BudgetSuggestionJob{
		Status: entity.BudgetSuggestionJobPending,
	}}
	uc.suggestionJobRepo = jobs

	// Run synchronously to assert the terminal status without timing or goroutine races.
	uc.runSMSBatchSuggestionJob("synthetic-job", 1, syntheticSuggestionMessages(26))

	if *calls != 2 || len(jobs.statuses) != 2 || jobs.statuses[0] != entity.BudgetSuggestionJobProcessing || jobs.statuses[1] != entity.BudgetSuggestionJobFailed {
		t.Fatalf("calls = %d, saved statuses = %v; want processing then failed", *calls, jobs.statuses)
	}
	if jobs.job.ResultJSON != "" || !strings.Contains(jobs.job.ErrorMessage, "all attempted AI chunks failed") {
		t.Errorf("result = %q, error = %q; want failed job with no successful result", jobs.job.ResultJSON, jobs.job.ErrorMessage)
	}
	if strings.Contains(jobs.job.ErrorMessage, "synthetic provider failure") {
		t.Error("job error must not expose raw provider response data")
	}
}
