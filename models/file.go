package models

import "gorm.io/gorm"

type MetaData struct {
	gorm.Model
	Type   string `gorm:"not null"`
	Name   string `gorm:"unique;not null;index:setting_key"`
	Size   string `gorm:"size:65535"`
	Md5    string `gorm:"type:varchar(32);not null"`
	Author string `gorm:"type:varchar(32);not null"`
	Owner  string `gorm:"type:varchar(32);not null"`
	Path   string `gorm:"type:varchar(255);not null"`
}
