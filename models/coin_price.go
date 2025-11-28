package models

import (
	"time"

	"gorm.io/gorm"
)

type CoinPrice struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Coin      string         `gorm:"type:varchar(10);not null;index" json:"coin"`
	Price     float64        `gorm:"type:decimal(20,8);not null" json:"price"`
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (CoinPrice) TableName() string {
	return "coin_prices"
}

