package mdb

import (
	"github.com/golang-module/carbon/v2"
	"gorm.io/plugin/optimisticlock"
)

type Base struct {
	ID uint64 `json:"id,omitempty" gorm:"primaryKey;autoIncrement:true;autoIncrementIncrement:1;"`
	// mysql need tag "type:datetime;"
	CreatedAt *carbon.DateTime `json:"created_at,omitempty" gorm:"not null;index;default:CURRENT_TIMESTAMP;"`
	UpdatedAt *carbon.DateTime `json:"updated_at,omitempty" gorm:"not null;default:CURRENT_TIMESTAMP;"`
	// version
	Version optimisticlock.Version `json:"version,omitempty" gorm:"not null;default:1;comment:版本号;"`
}
