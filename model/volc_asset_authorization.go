package model

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const (
	VolcAssetAuthorizationPending   = "Pending"
	VolcAssetAuthorizationCompleted = "Completed"
	VolcAssetAuthorizationExpired   = "Expired"
	VolcAssetAuthorizationFailed    = "Failed"
)

var (
	ErrVolcAssetAuthorizationNotFound = errors.New("Volcengine asset authorization session not found")
	ErrVolcAssetAuthorizationExpired  = errors.New("Volcengine asset authorization session expired")
)

type VolcAssetAuthorizationSession struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id" gorm:"index;not null"`
	TokenHash string `json:"-" gorm:"type:varchar(64);uniqueIndex;not null"`
	Status    string `json:"status" gorm:"type:varchar(16);index;not null"`
	ExpiresAt int64  `json:"expires_at" gorm:"bigint;index;not null"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;not null"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint;not null"`
}

func ExpireVolcAssetAuthorization(userId int, tokenHash string, now int64) error {
	if err := validateVolcAssetAuthorizationInput(userId, tokenHash); err != nil {
		return err
	}

	result := DB.Model(&VolcAssetAuthorizationSession{}).
		Where("user_id = ? AND token_hash = ? AND status = ?", userId, tokenHash, VolcAssetAuthorizationPending).
		Updates(map[string]any{
			"status":     VolcAssetAuthorizationExpired,
			"updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}

	var session VolcAssetAuthorizationSession
	err := DB.Where("user_id = ? AND token_hash = ?", userId, tokenHash).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrVolcAssetAuthorizationNotFound
	}
	if err != nil {
		return err
	}
	if session.Status == VolcAssetAuthorizationExpired {
		return nil
	}
	return ErrVolcAssetAuthorizationNotFound
}

func HasPendingVolcAssetAuthorization(userId int, now int64) (bool, error) {
	if userId <= 0 {
		return false, fmt.Errorf("invalid user id")
	}
	if err := DB.Model(&VolcAssetAuthorizationSession{}).
		Where("user_id = ? AND status = ? AND expires_at <= ?", userId, VolcAssetAuthorizationPending, now).
		Updates(map[string]any{
			"status":     VolcAssetAuthorizationExpired,
			"updated_at": now,
		}).Error; err != nil {
		return false, err
	}

	var count int64
	if err := DB.Model(&VolcAssetAuthorizationSession{}).
		Where("user_id = ? AND status = ? AND expires_at > ?", userId, VolcAssetAuthorizationPending, now).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func validateVolcAssetAuthorizationInput(userId int, tokenHash string) error {
	if userId <= 0 {
		return fmt.Errorf("invalid user id")
	}
	if len(tokenHash) != 64 {
		return fmt.Errorf("invalid authorization token digest")
	}
	digest, err := hex.DecodeString(tokenHash)
	if err != nil || len(digest) != 32 {
		return fmt.Errorf("invalid authorization token digest")
	}
	return nil
}

func StartVolcAssetAuthorization(userId int, tokenHash string, expiresAt, now int64) (*VolcAssetAuthorizationSession, error) {
	if err := validateVolcAssetAuthorizationInput(userId, tokenHash); err != nil {
		return nil, err
	}
	if expiresAt <= now {
		return nil, fmt.Errorf("authorization expiry must be in the future")
	}

	session := &VolcAssetAuthorizationSession{
		UserId:    userId,
		TokenHash: tokenHash,
		Status:    VolcAssetAuthorizationPending,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&VolcAssetAuthorizationSession{}).
			Where("user_id = ? AND status = ?", userId, VolcAssetAuthorizationPending).
			Updates(map[string]any{
				"status":     VolcAssetAuthorizationExpired,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}
		return tx.Create(session).Error
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

func FindPendingVolcAssetAuthorization(userId int, tokenHash string, now int64) (*VolcAssetAuthorizationSession, error) {
	if err := validateVolcAssetAuthorizationInput(userId, tokenHash); err != nil {
		return nil, err
	}

	var session VolcAssetAuthorizationSession
	err := DB.Where("user_id = ? AND token_hash = ?", userId, tokenHash).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVolcAssetAuthorizationNotFound
	}
	if err != nil {
		return nil, err
	}
	if session.Status != VolcAssetAuthorizationPending {
		return nil, ErrVolcAssetAuthorizationNotFound
	}
	if session.ExpiresAt <= now {
		if err := DB.Model(&session).
			Where("status = ?", VolcAssetAuthorizationPending).
			Updates(map[string]any{
				"status":     VolcAssetAuthorizationExpired,
				"updated_at": now,
			}).Error; err != nil {
			return nil, err
		}
		return nil, ErrVolcAssetAuthorizationExpired
	}
	return &session, nil
}

func CompleteVolcAssetAuthorization(userId int, tokenHash, groupId string, now int64) error {
	if err := validateVolcAssetAuthorizationInput(userId, tokenHash); err != nil {
		return err
	}
	groupId = strings.TrimSpace(groupId)
	if groupId == "" {
		return fmt.Errorf("invalid Volcengine asset user group id")
	}

	var authorizationErr error
	err := DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&VolcAssetAuthorizationSession{}).
			Where(
				"user_id = ? AND token_hash = ? AND status = ? AND expires_at > ?",
				userId, tokenHash, VolcAssetAuthorizationPending, now,
			).
			Updates(map[string]any{
				"status":     VolcAssetAuthorizationCompleted,
				"updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 1 {
			return saveVolcAssetUserGroup(tx, userId, groupId)
		}

		var session VolcAssetAuthorizationSession
		err := tx.Where("user_id = ? AND token_hash = ?", userId, tokenHash).First(&session).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			authorizationErr = ErrVolcAssetAuthorizationNotFound
			return nil
		}
		if err != nil {
			return err
		}
		if session.Status == VolcAssetAuthorizationCompleted {
			return nil
		}
		if session.Status == VolcAssetAuthorizationPending && session.ExpiresAt <= now {
			if err := tx.Model(&session).Updates(map[string]any{
				"status":     VolcAssetAuthorizationExpired,
				"updated_at": now,
			}).Error; err != nil {
				return err
			}
			authorizationErr = ErrVolcAssetAuthorizationExpired
			return nil
		}
		authorizationErr = ErrVolcAssetAuthorizationNotFound
		return nil
	})
	if err != nil {
		return err
	}
	return authorizationErr
}
