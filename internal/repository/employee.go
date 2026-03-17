package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/orgapi/internal/models"
	"gorm.io/gorm"
)

type EmployeeRepository interface {
	Create(ctx context.Context, departmentID uint, req *models.CreateEmployeeRequest) (*models.Employee, error)
}

type employeeRepo struct {
	db  *gorm.DB
	log *slog.Logger
}

func NewEmployeeRepo(db *gorm.DB, log *slog.Logger) EmployeeRepository {
	return &employeeRepo{db: db, log: log.With("repo", "employee")}
}

func (r *employeeRepo) Create(ctx context.Context, departmentID uint, req *models.CreateEmployeeRequest) (*models.Employee, error) {
	const op = "employee-repo/Create"

	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Department{}).
		Where("id = ?", departmentID).Count(&count).Error; err != nil {
		r.log.ErrorContext(ctx, "check department exists", "op", op, "err", err)
		return nil, err
	}
	if count == 0 {
		return nil, fmt.Errorf("%w: department %d", models.ErrNotFound, departmentID)
	}

	emp := &models.Employee{
		DepartmentID: departmentID,
		FullName:     req.FullName,
		Position:     req.Position,
		HiredAt:      req.HiredAt,
	}
	if err := r.db.WithContext(ctx).Create(emp).Error; err != nil {
		r.log.ErrorContext(ctx, "create employee", "op", op, "err", err)
		return nil, err
	}

	r.log.InfoContext(ctx, "employee created", "op", op, "id", emp.ID, "department_id", departmentID)
	return emp, nil
}
