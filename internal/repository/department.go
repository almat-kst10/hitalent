package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/orgapi/internal/models"
	"gorm.io/gorm"
)

type DepartmentRepository interface {
	Create(ctx context.Context, req *models.CreateDepartmentRequest) (*models.Department, error)
	GetByID(ctx context.Context, id uint) (*models.Department, error)
	GetDetail(ctx context.Context, id uint, depth int, includeEmployees bool) (*models.DepartmentDetailResponse, error)
	Update(ctx context.Context, id uint, req *models.UpdateDepartmentRequest) (*models.Department, error)
	Delete(ctx context.Context, id uint, mode string, reassignTo *uint) error
}

type departmentRepo struct {
	db  *gorm.DB
	log *slog.Logger
}

func NewDepartmentRepo(db *gorm.DB, log *slog.Logger) DepartmentRepository {
	return &departmentRepo{db: db, log: log.With("repo", "department")}
}

func (r *departmentRepo) Create(ctx context.Context, req *models.CreateDepartmentRequest) (*models.Department, error) {
	const op = "department-repo/Create"

	if req.ParentID != nil {
		var count int64
		if err := r.db.WithContext(ctx).Model(&models.Department{}).
			Where("id = ?", *req.ParentID).Count(&count).Error; err != nil {
			r.log.ErrorContext(ctx, "check parent exists", "op", op, "err", err)
			return nil, err
		}
		if count == 0 {
			return nil, fmt.Errorf("%w: parent department", models.ErrNotFound)
		}
	}

	if err := r.checkNameUnique(ctx, req.Name, req.ParentID, nil); err != nil {
		return nil, err
	}

	dept := &models.Department{Name: req.Name, ParentID: req.ParentID}
	if err := r.db.WithContext(ctx).Create(dept).Error; err != nil {
		r.log.ErrorContext(ctx, "create department", "op", op, "err", err)
		return nil, err
	}

	r.log.InfoContext(ctx, "department created", "op", op, "id", dept.ID, "name", dept.Name)
	return dept, nil
}

func (r *departmentRepo) GetByID(ctx context.Context, id uint) (*models.Department, error) {
	const op = "department-repo/GetByID"

	var dept models.Department
	err := r.db.WithContext(ctx).First(&dept, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: department %d", models.ErrNotFound, id)
	}
	if err != nil {
		r.log.ErrorContext(ctx, "get department by id", "op", op, "id", id, "err", err)
		return nil, err
	}
	return &dept, nil
}

func (r *departmentRepo) GetDetail(ctx context.Context, id uint, depth int, includeEmployees bool) (*models.DepartmentDetailResponse, error) {
	const op = "department-repo/GetDetail"

	dept, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	r.log.InfoContext(ctx, "building department detail", "op", op, "id", id, "depth", depth)
	return r.buildDetail(ctx, dept, depth, includeEmployees)
}

func (r *departmentRepo) buildDetail(ctx context.Context, dept *models.Department, depth int, includeEmployees bool) (*models.DepartmentDetailResponse, error) {
	const op = "department-repo/buildDetail"

	resp := &models.DepartmentDetailResponse{
		ID:        dept.ID,
		Name:      dept.Name,
		ParentID:  dept.ParentID,
		CreatedAt: dept.CreatedAt,
	}

	if includeEmployees {
		var employees []models.Employee
		if err := r.db.WithContext(ctx).
			Where("department_id = ?", dept.ID).
			Order("created_at asc").
			Find(&employees).Error; err != nil {
			r.log.ErrorContext(ctx, "load employees", "op", op, "department_id", dept.ID, "err", err)
			return nil, err
		}
		resp.Employees = employees
	}

	if depth > 0 {
		var children []models.Department
		if err := r.db.WithContext(ctx).
			Where("parent_id = ?", dept.ID).
			Find(&children).Error; err != nil {
			r.log.ErrorContext(ctx, "load children", "op", op, "department_id", dept.ID, "err", err)
			return nil, err
		}
		for i := range children {
			child, err := r.buildDetail(ctx, &children[i], depth-1, includeEmployees)
			if err != nil {
				return nil, err
			}
			resp.Children = append(resp.Children, *child)
		}
	}

	return resp, nil
}

func (r *departmentRepo) Update(ctx context.Context, id uint, req *models.UpdateDepartmentRequest) (*models.Department, error) {
	const op = "department-repo/Update"

	dept, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		parentID := dept.ParentID
		if req.ParentID != nil {
			parentID = req.ParentID
		}
		if req.ClearParent {
			parentID = nil
		}
		if err := r.checkNameUnique(ctx, *req.Name, parentID, &id); err != nil {
			return nil, err
		}
		dept.Name = *req.Name
	}

	if req.ClearParent {
		if err := r.db.WithContext(ctx).Model(dept).Update("parent_id", nil).Error; err != nil {
			r.log.ErrorContext(ctx, "clear parent", "op", op, "id", id, "err", err)
			return nil, err
		}
		dept.ParentID = nil
	} else if req.ParentID != nil {
		if *req.ParentID == id {
			return nil, fmt.Errorf("%w: department cannot be its own parent", models.ErrConflict)
		}
		if err := r.checkCycle(ctx, id, *req.ParentID); err != nil {
			return nil, err
		}
		var count int64
		if err := r.db.WithContext(ctx).Model(&models.Department{}).
			Where("id = ?", *req.ParentID).Count(&count).Error; err != nil {
			r.log.ErrorContext(ctx, "check new parent exists", "op", op, "err", err)
			return nil, err
		}
		if count == 0 {
			return nil, fmt.Errorf("%w: parent department", models.ErrNotFound)
		}
		dept.ParentID = req.ParentID
	}

	if err := r.db.WithContext(ctx).Save(dept).Error; err != nil {
		r.log.ErrorContext(ctx, "save department", "op", op, "id", id, "err", err)
		return nil, err
	}

	r.log.InfoContext(ctx, "department updated", "op", op, "id", id)
	return dept, nil
}

func (r *departmentRepo) Delete(ctx context.Context, id uint, mode string, reassignTo *uint) error {
	const op = "department-repo/Delete"

	if _, err := r.GetByID(ctx, id); err != nil {
		return err
	}

	switch mode {
	case "cascade":
		r.log.InfoContext(ctx, "cascade delete", "op", op, "id", id)
		return r.deleteCascade(ctx, id)

	case "reassign":
		r.log.InfoContext(ctx, "reasign delete", "op", op, "id", id)
		return r.deleteReasign(ctx, id, reassignTo)

	default:
		return fmt.Errorf("%w: mode must be 'cascade' or 'reassign'", models.ErrConflict)
	}
}

func (r *departmentRepo) deleteReasign(ctx context.Context, id uint, reassignTo *uint) error {
	const op = "department-repo/deleteReasign"
	if reassignTo == nil {
		return fmt.Errorf("%w: reassign_to_department_id is required", models.ErrConflict)
	}
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Department{}).
		Where("id = ?", *reassignTo).Count(&count).Error; err != nil {
		r.log.ErrorContext(ctx, "check reassign target", "op", op, "err", err)
		return err
	}
	if count == 0 {
		return fmt.Errorf("%w: reassign target department", models.ErrNotFound)
	}
	if err := r.db.WithContext(ctx).Model(&models.Employee{}).
		Where("department_id = ?", id).
		Update("department_id", *reassignTo).Error; err != nil {
		r.log.ErrorContext(ctx, "reassign employees", "op", op, "err", err)
		return err
	}
	if err := r.db.WithContext(ctx).Model(&models.Department{}).
		Where("parent_id = ?", id).
		Update("parent_id", *reassignTo).Error; err != nil {
		r.log.ErrorContext(ctx, "reassign children", "op", op, "err", err)
		return err
	}
	if err := r.db.WithContext(ctx).Delete(&models.Department{}, id).Error; err != nil {
		r.log.ErrorContext(ctx, "delete department", "op", op, "id", id, "err", err)
		return err
	}
	r.log.InfoContext(ctx, "department deleted (reassign)", "op", op, "id", id, "reassign_to", *reassignTo)
	return nil
}

func (r *departmentRepo) deleteCascade(ctx context.Context, id uint) error {
	const op = "department-repo/deleteCascade"

	var children []models.Department
	if err := r.db.WithContext(ctx).Where("parent_id = ?", id).Find(&children).Error; err != nil {
		r.log.ErrorContext(ctx, "load children for cascade", "op", op, "id", id, "err", err)
		return err
	}
	for _, child := range children {
		if err := r.deleteCascade(ctx, child.ID); err != nil {
			return err
		}
	}
	if err := r.db.WithContext(ctx).Where("department_id = ?", id).Delete(&models.Employee{}).Error; err != nil {
		r.log.ErrorContext(ctx, "delete employees", "op", op, "department_id", id, "err", err)
		return err
	}
	if err := r.db.WithContext(ctx).Delete(&models.Department{}, id).Error; err != nil {
		r.log.ErrorContext(ctx, "delete department", "op", op, "id", id, "err", err)
		return err
	}
	r.log.InfoContext(ctx, "department cascade deleted", "op", op, "id", id)
	return nil
}

func (r *departmentRepo) checkCycle(ctx context.Context, id, newParentID uint) error {
	current := newParentID
	visited := map[uint]bool{}
	for {
		if current == id {
			return fmt.Errorf("%w: moving would create a cycle", models.ErrCycleDetected)
		}
		if visited[current] {
			break
		}
		visited[current] = true
		var dept models.Department
		if err := r.db.WithContext(ctx).Select("id, parent_id").First(&dept, current).Error; err != nil {
			break
		}
		if dept.ParentID == nil {
			break
		}
		current = *dept.ParentID
	}
	return nil
}

func (r *departmentRepo) checkNameUnique(ctx context.Context, name string, parentID *uint, excludeID *uint) error {
	const op = "department-repo/checkNameUnique"

	q := r.db.WithContext(ctx).Model(&models.Department{}).Where("name = ?", name)
	if parentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}
	if excludeID != nil {
		q = q.Where("id != ?", *excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		r.log.ErrorContext(ctx, "check name unique", "op", op, "err", err)
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: name '%s' already exists in this parent", models.ErrConflict, name)
	}
	return nil
}
