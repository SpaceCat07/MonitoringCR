package models

import (
	"time"

	"gorm.io/gorm"
)

type SubTask struct {
	gorm.Model

	// relasi dengan CR
	CRID *uint          `gorm:"not null;index" json:"cr_id"`
	CR   *ChangeRequest `gorm:"foreignKey:CRID;references:ID" json:"cr"`

	// relasi dengan collaborator
	CollaboratorID *uint  `gorm:"index" json:"collaborator_id"`
	Collaborator   *Users `gorm:"foreignKey:CollaboratorID;references:ID" json:"collaborator"`

	TaskName string `gorm:"type:varchar(255);not null" json:"task_name"`
	Done     bool   `gorm:"type:boolean;default:false" json:"done"`

	DueDate    time.Time `gorm:"not null" json:"due_date"`
	Progress   uint      `gorm:"int;not null;default:0;check:progress >= 0 AND progress <= 100" json:"progress"`
	Keterangan string    `gorm:"type:text;default:''" json:"keterangan"`
}
