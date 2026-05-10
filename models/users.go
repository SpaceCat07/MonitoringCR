package models

import (
	"gorm.io/gorm"
)

type Users struct {
	gorm.Model
	Fullname     string `gorm:"type:varchar(100);not null" json:"fullname"`
	Email        string `gorm:"type:varchar(100);not null;uniqueIndex" json:"email"`
	PasswordHash string `gorm:"size:255" json:"password"`
	Role         string `gorm:"type:varchar(25);check:role IN ('Admin', 'Manager', 'PIC', 'Collaborator');" json:"role"`
	ParentPIC    *uint  `gorm:"nullable;index" json:"parent_pic"`
	Parent       *Users `gorm:"foreignKey:ParentPIC;references:ID" json:"parent"`
}
