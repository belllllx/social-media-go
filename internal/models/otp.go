package models

import "time"

type OTP struct {
	ID        int64
	Email     string `gorm:"type:varchar(30);uniqueIndex"`
	OTPHash   string
	ExpiredAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
