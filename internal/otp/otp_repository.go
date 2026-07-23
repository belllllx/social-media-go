package otp

import (
	"time"

	"gorm.io/gorm"
)

type OTP struct {
	ID        int64
	Email     string `gorm:"type:varchar(30);uniqueIndex"`
	OTPHash   string
	ExpiredAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OTPRepository interface {
	Create(db *gorm.DB, otp *OTP) error
	FindByEmail(db *gorm.DB, email string) (*OTP, error)
	FindNotExpired(db *gorm.DB, email string) (*OTP, error)
	Delete(db *gorm.DB, email string) error
	DeleteByExpired(db *gorm.DB) error
}

type otpRepositoryDB struct {
}

func NewOTPRepositoryDB(db *gorm.DB) OTPRepository {
	db.AutoMigrate(&OTP{})
	return &otpRepositoryDB{}
}

func (r *otpRepositoryDB) Create(db *gorm.DB, otp *OTP) error {
	return db.Create(otp).Error
}

func (r *otpRepositoryDB) FindByEmail(db *gorm.DB, email string) (*OTP, error) {
	otp := &OTP{}
	err := db.Where("email = ?", email).Take(otp).Error
	if err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *otpRepositoryDB) FindNotExpired(db *gorm.DB, email string) (*OTP, error) {
	otp := &OTP{}
	err := db.Where("email = ? AND expired_at > ?", email, time.Now()).Take(otp).Error
	if err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *otpRepositoryDB) Delete(db *gorm.DB, email string) error {
	otp := &OTP{}
	return db.Where("email = ?", email).Delete(otp).Error
}

func (r *otpRepositoryDB) DeleteByExpired(db *gorm.DB) error {
	otp := &OTP{}
	return db.Where("expired_at < ?", time.Now()).Delete(otp).Error
}
