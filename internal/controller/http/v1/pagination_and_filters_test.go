package v1

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParsePaginationParams_Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/items", nil)

	page, perPage, offset, hasPagination := ParsePaginationParams(c, 20)
	if hasPagination {
		t.Fatalf("expected hasPagination false")
	}
	if page != 1 || perPage != 20 || offset != 0 {
		t.Fatalf("unexpected defaults: page=%d perPage=%d offset=%d", page, perPage, offset)
	}
}

func TestParsePaginationParams_CustomValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/items?page=3&per_page=15", nil)

	page, perPage, offset, hasPagination := ParsePaginationParams(c, 20)
	if !hasPagination {
		t.Fatalf("expected hasPagination true")
	}
	if page != 3 || perPage != 15 || offset != 30 {
		t.Fatalf("unexpected values: page=%d perPage=%d offset=%d", page, perPage, offset)
	}
}

func TestBuildFilterFromQuery_AdvancedFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(
		"GET",
		"/transactions?category_ids=1,2,3&from_date=2026-01-01&to_date=2026-01-31&amount_min=10.5&amount_max=99.9",
		nil,
	)

	h := &TransactionHandler{}
	filter := h.buildFilterFromQuery(c)

	if len(filter.CategoryIDs) != 3 {
		t.Fatalf("expected 3 category IDs, got %d", len(filter.CategoryIDs))
	}
	if filter.FromDate == nil || filter.ToDate == nil {
		t.Fatalf("expected date range to be parsed")
	}
	if filter.MinAmount == nil || filter.MaxAmount == nil {
		t.Fatalf("expected amount range to be parsed")
	}
}
