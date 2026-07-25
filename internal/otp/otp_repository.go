package otp

import (
	"time"

	"github.com/belllllx/social-media-go/internal/models"
	"gorm.io/gorm"
)

type OTPRepository interface {
	Create(db *gorm.DB, otp *models.OTP) error
	FindByEmail(db *gorm.DB, email string) (*models.OTP, error)
	FindNotExpired(db *gorm.DB, email string) (*models.OTP, error)
	Delete(db *gorm.DB, email string) error
	DeleteByExpired(db *gorm.DB) error
}

type otpRepositoryDB struct {
}

func NewOTPRepositoryDB() OTPRepository {
	return &otpRepositoryDB{}
}

func (r *otpRepositoryDB) Create(db *gorm.DB, otp *models.OTP) error {
	return db.Create(otp).Error
}

func (r *otpRepositoryDB) FindByEmail(db *gorm.DB, email string) (*models.OTP, error) {
	otp := &models.OTP{}
	err := db.Where("email = ?", email).Take(otp).Error
	if err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *otpRepositoryDB) FindNotExpired(db *gorm.DB, email string) (*models.OTP, error) {
	otp := &models.OTP{}
	err := db.Where("email = ? AND expired_at > ?", email, time.Now()).Take(otp).Error
	if err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *otpRepositoryDB) Delete(db *gorm.DB, email string) error {
	otp := &models.OTP{}
	return db.Where("email = ?", email).Delete(otp).Error
}

func (r *otpRepositoryDB) DeleteByExpired(db *gorm.DB) error {
	otp := &models.OTP{}
	return db.Where("expired_at < ?", time.Now()).Delete(otp).Error
}
