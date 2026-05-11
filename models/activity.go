package models

import "gorm.io/gorm"

type Activity struct {
	gorm.Model

	// relasi dengan cr
	CRID			*uint `gorm:"not null;index" json:"cr_id"`
	CR 				*ChangeRequest `gorm:"foreignKey:CRID;references:ID" json:"cr"`

	// relasi dengan user
	UserID			*uint `gorm:"not null;index" json:"user_id"`
	User			*Users `gorm:"foreignKey:UserID;references:ID" json:"user"`

	Action			string `gorm:"type:varchar(50);check:action IN ('Comment', 'Change');not null" json:"action"`
	Activities		string `gorm:"type:varchar(255);nullable" json:"activity"`
	Comment			string `gorm:"type:varchar(255);nullable" json:"comment"`
}