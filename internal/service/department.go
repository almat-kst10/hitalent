package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/orgapi/internal/helper"
	"github.com/orgapi/internal/models"
	"github.com/orgapi/internal/repository"
)

type DepartmentService interface {
	Create(ctx context.Context, req *models.CreateDepartmentRequest) (*models.Department, error)
	GetDetail(ctx context.Context, id uint, depth int, includeEmployees bool) (*models.DepartmentDetailResponse, error)
	Update(ctx context.Context, id uint, req *models.UpdateDepartmentRequest) (*models.Department, error)
	Delete(ctx context.Context, id uint, mode string, reassignTo *uint) error
}

// ── Department ────────────────────────────────────────────────────────────────

type departmentService struct {
	repo repository.DepartmentRepository
	log  *slog.Logger
}

func NewDepartmentService(repo repository.DepartmentRepository, log *slog.Logger) DepartmentService {
	return &departmentService{repo: repo, log: log.With("service", "department")}
}

func (s *departmentService) Create(ctx context.Context, req *models.CreateDepartmentRequest) (*models.Department, error) {
	const op = "department-service/Create"

	req.Name = strings.TrimSpace(req.Name)
	if err := helper.ValidateField("name", req.Name); err != nil {
		s.log.WarnContext(ctx, "validation failed", "op", op, "err", err)
		return nil, err
	}

	dept, err := s.repo.Create(ctx, req)
	if err != nil {
		s.log.ErrorContext(ctx, "repo error", "op", op, "err", err)
		return nil, err
	}

	s.log.InfoContext(ctx, "department created", "op", op, "id", dept.ID)
	return dept, nil
}

func (s *departmentService) GetDetail(ctx context.Context, id uint, depth int, includeEmployees bool) (*models.DepartmentDetailResponse, error) {
	const op = "department-service/GetDetail"

	if depth < 0 {
		depth = 1
	}
	if depth > 5 {
		depth = 5
	}

	s.log.InfoContext(ctx, "get detail", "op", op, "id", id, "depth", depth, "include_employees", includeEmployees)

	result, err := s.repo.GetDetail(ctx, id, depth, includeEmployees)
	if err != nil {
		s.log.ErrorContext(ctx, "repo error", "op", op, "id", id, "err", err)
		return nil, err
	}
	return result, nil
}

func (s *departmentService) Update(ctx context.Context, id uint, req *models.UpdateDepartmentRequest) (*models.Department, error) {
	const op = "department-service/Update"

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if err := helper.ValidateField("name", trimmed); err != nil {
			s.log.WarnContext(ctx, "validation failed", "op", op, "err", err)
			return nil, err
		}
		req.Name = &trimmed
	}

	dept, err := s.repo.Update(ctx, id, req)
	if err != nil {
		s.log.ErrorContext(ctx, "repo error", "op", op, "id", id, "err", err)
		return nil, err
	}

	s.log.InfoContext(ctx, "department updated", "op", op, "id", id)
	return dept, nil
}

func (s *departmentService) Delete(ctx context.Context, id uint, mode string, reassignTo *uint) error {
	const op = "department-service/Delete"

	s.log.InfoContext(ctx, "delete department", "op", op, "id", id, "mode", mode)

	if err := s.repo.Delete(ctx, id, mode, reassignTo); err != nil {
		s.log.ErrorContext(ctx, "repo error", "op", op, "id", id, "err", err)
		return err
	}
	return nil
}
