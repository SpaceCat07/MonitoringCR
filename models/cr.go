package models

import (
	"time"

	"gorm.io/gorm"
)

type ChangeRequest struct {
	gorm.Model
	Title          string    `gorm:"type:varchar(255);not null" json:"title"`
	Description    string    `gorm:"type:text;not null" json:"description"`
	Goal		   string    `gorm:"type:text;not null" json:"goal"`
	Impact		   string    `gorm:"type:text;not null" json:"impact"`
	Keterangan     string    `gorm:"type:text" json:"keterangan"`
	Modul          string    `gorm:"type:varchar(100);not null" json:"modul"`
	Category       string    `gorm:"type:varchar(50);not null" json:"category"`
	Status         string    `gorm:"type:varchar(50);not null;default:'DRAFT'" json:"status"`
	ReleaseDate    time.Time `gorm:"not null" json:"release_date"`
	StartDate	   time.Time `gorm:"not null" json:"start_date"`
	EndDate        time.Time `gorm:"not null" json:"end_date"`
	FileAttachment []string  `gorm:"type:jsonb;serializer:json;not null" json:"file_attachment"`
	CreatedBy      uint      `gorm:"index" json:"created_by"`
	Creator        *Users    `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	

	// relasi dengan user sebagai pic
	PICID		   *uint	 `gorm:"not null;index" json:"pic_id"`
	PIC 		   *Users	 `gorm:"foreignKey:PICID;references:ID" json:"pic"`

	// relasi satu-ke-banyak (One-to-Many)
	Activities     []Activity `gorm:"foreignKey:CRID" json:"activities"`
	SubTasks       []SubTask  `gorm:"foreignKey:CRID" json:"sub_tasks"`
}
