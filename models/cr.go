package models

import (
	"time"

	"gorm.io/gorm"
)

type ChangeRequest struct {
	gorm.Model
	Title          string    `gorm:"type:varchar(255);not null" json:"title"`
	Description    string    `gorm:"type:text;not null" json:"description"`
	Category       string    `gorm:"type:varchar(50);not null" json:"category"`
	Status         string    `gorm:"type:varchar(50);not null;default:'ISSUED'" json:"status"`
	ReleaseDate    time.Time `gorm:"not null" json:"release_date"`
	EndDate        time.Time `gorm:"not null" json:"end_date"`
	FileAttachment []string  `gorm:"type:jsonb;serializer:json;not null" json:"file_attachment"`
	CreatedBy      uint      `gorm:"index" json:"created_by"`
	Creator        *Users    `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}
