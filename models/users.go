package models

import (
	"gorm.io/gorm"
)

type Users struct {
	gorm.Model
	Fullname		string `gorm:"type:varchar(100);not null" json:"fullname"`
	Email			string `gorm:"type:varchar(100);not null;uniqueIndex" json:"email"`
	PasswordHash	string `gorm:"size:255" json:"password"`
}