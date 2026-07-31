package otp

import (
	"context"
	"time"

	"github.com/belllllx/social-media-go/internal/models"
	"gorm.io/gorm"
)

type OTPRepository interface {
	Create(
		ctx context.Context,
		db *gorm.DB,
		otp *models.OTP,
	) error
	FindByEmail(
		ctx context.Context,
		db *gorm.DB,
		email string,
	) (*models.OTP, error)
	FindNotExpired(
		ctx context.Context,
		db *gorm.DB,
		email string,
	) (*models.OTP, error)
	Delete(
		ctx context.Context,
		db *gorm.DB,
		email string,
	) error
	DeleteByExpired(ctx context.Context, db *gorm.DB) error
}

type otpRepositoryDB struct {
}

func NewOTPRepositoryDB() OTPRepository {
	return &otpRepositoryDB{}
}

func (r *otpRepositoryDB) Create(
	ctx context.Context,
	db *gorm.DB,
	otp *models.OTP,
) error {
	return db.WithContext(ctx).Create(otp).Error
}

func (r *otpRepositoryDB) FindByEmail(
	ctx context.Context,
	db *gorm.DB,
	email string,
) (*models.OTP, error) {
	otp := &models.OTP{}
	err := db.
		WithContext(ctx).
		Where("email = ?", email).
		Take(otp).Error
	if err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *otpRepositoryDB) FindNotExpired(
	ctx context.Context,
	db *gorm.DB,
	email string,
) (*models.OTP, error) {
	otp := &models.OTP{}
	err := db.
		WithContext(ctx).
		Where("email = ? AND expired_at > ?", email, time.Now()).
		Take(otp).Error
	if err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *otpRepositoryDB) Delete(
	ctx context.Context,
	db *gorm.DB,
	email string,
) error {
	otp := &models.OTP{}
	return db.
		WithContext(ctx).
		Where("email = ?", email).
		Delete(otp).Error
}

func (r *otpRepositoryDB) DeleteByExpired(ctx context.Context, db *gorm.DB) error {
	otp := &models.OTP{}
	return db.
		WithContext(ctx).
		Where("expired_at < ?", time.Now()).
		Delete(otp).Error
}
