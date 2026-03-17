package models

import (
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrCycleDetected = errors.New("cycle detected in department tree")
)

// ── DB models ─────────────────────────────────────────────────────────────────

type Department struct {
	ID        uint      `gorm:"primaryKey"                                          json:"id"`
	Name      string    `gorm:"not null"                                            json:"name"`
	ParentID  *uint     `gorm:"index"                                               json:"parent_id"`
	CreatedAt time.Time `                                                           json:"created_at"`

	Children  []Department `gorm:"foreignKey:ParentID"                                 json:"-"`
	Employees []Employee   `gorm:"foreignKey:DepartmentID;constraint:OnDelete:CASCADE" json:"-"`
}

type Employee struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	DepartmentID uint       `gorm:"not null;index" json:"department_id"`
	FullName     string     `gorm:"not null"   json:"full_name"`
	Position     string     `gorm:"not null"   json:"position"`
	HiredAt      *time.Time `                  json:"hired_at"`
	CreatedAt    time.Time  `                  json:"created_at"`
}

// ── Requests ──────────────────────────────────────────────────────────────────

type CreateDepartmentRequest struct {
	Name     string `json:"name"`
	ParentID *uint  `json:"parent_id"`
}

type UpdateDepartmentRequest struct {
	Name        *string `json:"name"`
	ParentID    *uint   `json:"parent_id"`
	ClearParent bool    `json:"clear_parent"` // true → parent_id = NULL
}

type CreateEmployeeRequest struct {
	FullName string     `json:"full_name"`
	Position string     `json:"position"`
	HiredAt  *time.Time `json:"hired_at"`
}

// ── Responses ─────────────────────────────────────────────────────────────────

type DepartmentDetailResponse struct {
	ID        uint                       `json:"id"`
	Name      string                     `json:"name"`
	ParentID  *uint                      `json:"parent_id"`
	CreatedAt time.Time                  `json:"created_at"`
	Employees []Employee                 `json:"employees,omitempty"`
	Children  []DepartmentDetailResponse `json:"children,omitempty"`
}

type ResponseError struct {
	Error string `json:"error"`
}
