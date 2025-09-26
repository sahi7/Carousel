package models

import (
	"time"
	"encoding/json"
)

// CustomUser represents the user model
type CustomUser struct {
	ID                int64
	Username          string
	Role              string
	Salary            float64
	PreferredLanguage string
	Timezone          string
	Branches          []int64 `json:"branches"`        // From cre_customuser_branches M2M table
    Restaurants       []int64 `json:"restaurants"`     // From cre_customuser_restaurants M2M table
    Companies         []int64 `json:"companies"`       // From cre_customuser_companies M2M table
}

// Rule represents a payroll rule
type Rule struct {
	ID            int64
	Name          string
	Type          string
	Amount        float64
	Scope         string
	CompanyID     *int64
	RestaurantID  *int64
	BranchID      *int64
	Priority      int
	IsActive      bool
	EffectiveFrom time.Time
}

// RuleTarget represents a rule target
type RuleTarget struct {
	RuleID      int64
	TargetType  string
	TargetValue string
	BranchID    *int64
}

// MappedTarget represents a rule mapped to specific user IDs
type MappedTarget struct {
	RuleID           int64
	AssociatedUserIDs []int64
}

// UnmappedTarget represents a user with unmapped rule IDs
type UnmappedTarget struct {
	UserID  int64
	RuleIDs []int64
	UserIDs []int64
}

// Period represents a payroll period
type Period struct {
	ID    int64
	Month int
	Year  int
}

// Override represents a payroll override
type Override struct {
	RuleID       int64
	PeriodID     *int64
	UserID       *int64
	OverrideType string
	Amount       float64
	BranchID     *int64
	Notes        string
	EffectiveFrom time.Time
	ExpiresAt    *time.Time
	Type         string
	Name         string
}

// MatchedOverride represents an override matched to a specific user
type MatchedOverride struct {
	Override Override
	UserID   int64
}

// EffectiveRule represents a precomputed rule
type EffectiveRule struct {
	PeriodID   int64
	UserID     int64
	BranchID   int64
	RuleID     int64
	Amount     *float64
	Percentage *float64
	Source     string
	Type       string
}

// Record represents a payroll record
type Record struct {
	UserID         int64
	BranchID       int64
	PeriodID       int64
	BaseSalary     float64
	TotalBonus     float64
	TotalDeduction float64
	NetPay         float64
	Status         string
	GeneratedAt    time.Time
	BonusDetails   json.RawMessage
	DeductionDetails json.RawMessage
}

// Component represents a rule's contribution to a payroll record
type Component struct {
	RecordID int64
	RuleID   int64
	Amount   float64
}

// AuditLog represents a payroll audit log using branch_activity table
type AuditLog struct {
	ActivityType string
	Timestamp    time.Time
	Details      map[string]interface{} // JSONB
	BranchID     *int64
	UserID       *int64
}