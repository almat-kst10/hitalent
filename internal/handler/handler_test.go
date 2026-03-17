package handler_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/orgapi/internal/handler"
	"github.com/orgapi/internal/models"
	"github.com/orgapi/internal/repository"
	"github.com/orgapi/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Department{}, &models.Employee{}); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	deptRepo := repository.NewDepartmentRepo(db, log)
	empRepo := repository.NewEmployeeRepo(db, log)
	return handler.New(
		service.NewDepartmentService(deptRepo, log),
		service.NewEmployeeService(empRepo, log),
		log,
	).Routes()
}

func post(t *testing.T, h http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func patch(t *testing.T, h http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func del(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestCreateDepartment_OK(t *testing.T) {
	h := newTestHandler(t)
	w := post(t, h, "/departments/", map[string]any{"name": "Engineering"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var dept models.Department
	json.NewDecoder(w.Body).Decode(&dept)
	if dept.Name != "Engineering" {
		t.Errorf("expected name 'Engineering', got '%s'", dept.Name)
	}
}

func TestCreateDepartment_EmptyName(t *testing.T) {
	h := newTestHandler(t)
	w := post(t, h, "/departments/", map[string]any{"name": "   "})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateDepartment_DuplicateName(t *testing.T) {
	h := newTestHandler(t)
	post(t, h, "/departments/", map[string]any{"name": "Backend"})
	w := post(t, h, "/departments/", map[string]any{"name": "Backend"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate, got %d", w.Code)
	}
}

func TestCreateEmployee_DeptNotFound(t *testing.T) {
	h := newTestHandler(t)
	w := post(t, h, "/departments/999/employees/", map[string]any{
		"full_name": "Alice", "position": "Dev",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetDepartment_WithChildren(t *testing.T) {
	h := newTestHandler(t)

	w := post(t, h, "/departments/", map[string]any{"name": "Root"})
	var parent models.Department
	json.NewDecoder(w.Body).Decode(&parent)

	post(t, h, "/departments/", map[string]any{"name": "Child", "parent_id": parent.ID})

	w = get(t, h, "/departments/1?depth=2")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var detail models.DepartmentDetailResponse
	json.NewDecoder(w.Body).Decode(&detail)
	if len(detail.Children) == 0 {
		t.Error("expected children in response")
	}
}

func TestDeleteDepartment_Cascade(t *testing.T) {
	h := newTestHandler(t)
	post(t, h, "/departments/", map[string]any{"name": "ToDelete"})
	w := del(t, h, "/departments/1?mode=cascade")
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestDeleteDepartment_MissingMode(t *testing.T) {
	h := newTestHandler(t)
	post(t, h, "/departments/", map[string]any{"name": "X"})
	w := del(t, h, "/departments/1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCycleDetection(t *testing.T) {
	h := newTestHandler(t)
	post(t, h, "/departments/", map[string]any{"name": "A"})
	post(t, h, "/departments/", map[string]any{"name": "B", "parent_id": 1})

	// пытаемся сделать A дочерним B — цикл
	w := patch(t, h, "/departments/1", map[string]any{"parent_id": 2})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for cycle, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSelfParent(t *testing.T) {
	h := newTestHandler(t)
	post(t, h, "/departments/", map[string]any{"name": "A"})
	w := patch(t, h, "/departments/1", map[string]any{"parent_id": 1})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for self-parent, got %d", w.Code)
	}
}
