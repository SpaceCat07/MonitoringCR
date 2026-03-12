package models

import (
	"gorm.io/gorm"
)

type SPR struct {
	gorm.Model
	Title                 string  `gorm:"type:varchar(800);not null" json:"title"`
	BudgetType            string  `gorm:"type:varchar(50)" json:"budget_type"`
	BudgetYear            int     `json:"budget_year"`
	ExpenseClassification string  `gorm:"type:varchar(100)" json:"expense_classification"`
	SPRType               string  `gorm:"type:varchar(50)" json:"spr_type"`
	ProcurementType       string  `gorm:"type:varchar(50)" json:"procurement_type"`
	BudgetCode            string  `gorm:"type:varchar(50)" json:"budget_code"`
	WorkProgram           string  `gorm:"type:varchar(255)" json:"work_program"`
	RemainingBudget       float64 `gorm:"type:numeric" json:"remaining_budget"`
	Status                string  `gorm:"type:varchar(20);default:'draft'" json:"status"`
}
