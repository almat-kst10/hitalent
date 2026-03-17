package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/orgapi/internal/helper"
	"github.com/orgapi/internal/models"
	"github.com/orgapi/internal/repository"
)

type EmployeeService interface {
	Create(ctx context.Context, departmentID uint, req *models.CreateEmployeeRequest) (*models.Employee, error)
}

type employeeService struct {
	repo repository.EmployeeRepository
	log  *slog.Logger
}

func NewEmployeeService(repo repository.EmployeeRepository, log *slog.Logger) EmployeeService {
	return &employeeService{repo: repo, log: log.With("service", "employee")}
}

func (s *employeeService) Create(ctx context.Context, departmentID uint, req *models.CreateEmployeeRequest) (*models.Employee, error) {
	const op = "employee-service/Create"

	req.FullName = strings.TrimSpace(req.FullName)
	req.Position = strings.TrimSpace(req.Position)

	if err := helper.ValidateField("full_name", req.FullName); err != nil {
		s.log.WarnContext(ctx, "validation failed", "op", op, "err", err)
		return nil, err
	}
	if err := helper.ValidateField("position", req.Position); err != nil {
		s.log.WarnContext(ctx, "validation failed", "op", op, "err", err)
		return nil, err
	}

	emp, err := s.repo.Create(ctx, departmentID, req)
	if err != nil {
		s.log.ErrorContext(ctx, "repo error", "op", op, "err", err)
		return nil, err
	}

	s.log.InfoContext(ctx, "employee created", "op", op, "id", emp.ID, "department_id", departmentID)
	return emp, nil
}
