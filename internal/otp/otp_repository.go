package otp

import (
	"time"

	"gorm.io/gorm"
)

type OTP struct {
	ID        int64
	Email     string `gorm:"type:varchar(30);unique"`
	OTPHash   string
	ExpiredAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OTPRepository interface {
	Create(otp *OTP) error
	FindByEmail(email string) (*OTP, error)
	FindNotExpired(email string) (*OTP, error)
	Delete(email string) error
}

type otpRepositoryDB struct {
	db *gorm.DB
}

func NewOTPRepositoryDB(db *gorm.DB) OTPRepository {
	db.AutoMigrate(&OTP{})
	return &otpRepositoryDB{db: db}
}

func (r *otpRepositoryDB) Create(otp *OTP) error {
	return r.db.Create(otp).Error
}

func (r *otpRepositoryDB) FindByEmail(email string) (*OTP, error) {
	otp := &OTP{}
	err := r.db.Where("email = ?", email).Take(otp).Error
	if err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *otpRepositoryDB) FindNotExpired(email string) (*OTP, error) {
	otp := &OTP{}
	err := r.db.Where("email = ? AND expired_at > ?", email, time.Now()).Take(otp).Error
	if err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *otpRepositoryDB) Delete(email string) error {
	otp := &OTP{}
	return r.db.Where("email = ?", email).Delete(otp).Error
}
