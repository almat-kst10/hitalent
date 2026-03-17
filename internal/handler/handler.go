package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/orgapi/internal/models"
	"github.com/orgapi/internal/service"
)

type Handler struct {
	deptSvc service.DepartmentService
	empSvc  service.EmployeeService
	log     *slog.Logger
}

func New(deptSvc service.DepartmentService, empSvc service.EmployeeService, log *slog.Logger) *Handler {
	return &Handler{deptSvc: deptSvc, empSvc: empSvc, log: log}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /departments/", h.createDepartment)
	mux.HandleFunc("GET /departments/{id}", h.getDepartment)
	mux.HandleFunc("PATCH /departments/{id}", h.updateDepartment)
	mux.HandleFunc("DELETE /departments/{id}", h.deleteDepartment)
	mux.HandleFunc("POST /departments/{id}/employees/", h.createEmployee)
	return mux
}

// ── Department handlers ───────────────────────────────────────────────────────

func (h *Handler) createDepartment(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	dept, err := h.deptSvc.Create(r.Context(), &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dept)
}

func (h *Handler) getDepartment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	depth := queryInt(r, "depth", 1)
	includeEmployees := queryBool(r, "include_employees", true)

	dept, err := h.deptSvc.GetDetail(r.Context(), id, depth, includeEmployees)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dept)
}

func (h *Handler) updateDepartment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req models.UpdateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	dept, err := h.deptSvc.Update(r.Context(), id, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dept)
}

func (h *Handler) deleteDepartment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	mode := r.URL.Query().Get("mode")
	if mode == "" {
		writeError(w, http.StatusBadRequest, "mode is required: cascade or reassign")
		return
	}

	var reassignTo *uint
	if raw := r.URL.Query().Get("reassign_to_department_id"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid reassign_to_department_id")
			return
		}
		uid := uint(v)
		reassignTo = &uid
	}

	if err := h.deptSvc.Delete(r.Context(), id, mode, reassignTo); err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Employee handlers ─────────────────────────────────────────────────────────

func (h *Handler) createEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req models.CreateEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	emp, err := h.empSvc.Create(r.Context(), id, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, emp)
}

// ── error handling ────────────────────────────────────────────────────────────

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	h.log.Error("handler error", "err", err)
	switch {
	case errors.Is(err, models.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, models.ErrConflict),
		errors.Is(err, models.ErrCycleDetected):
		writeError(w, http.StatusConflict, err.Error())
	case strings.Contains(err.Error(), "must not"):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parseID(r *http.Request) (uint, error) {
	v, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}

func queryInt(r *http.Request, key string, def int) int {
	if raw := r.URL.Query().Get(key); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			return v
		}
	}
	return def
}

func queryBool(r *http.Request, key string, def bool) bool {
	if raw := r.URL.Query().Get(key); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			return v
		}
	}
	return def
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, models.ResponseError{Error: msg})
}
