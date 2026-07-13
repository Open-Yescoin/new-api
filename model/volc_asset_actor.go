package model

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	VolcAssetActorPending      = "Pending"
	VolcAssetActorActive       = "Active"
	VolcAssetActorExpired      = "Expired"
	VolcAssetActorRevoked      = "Revoked"
	VolcAssetActorLegacyReview = "LegacyReview"
)

var (
	ErrVolcAssetActorNotFound              = errors.New("Seedance actor not found")
	ErrVolcAssetInvalidDuration            = errors.New("invalid Seedance actor authorization duration")
	ErrVolcAssetActorAuthorizationRequired = errors.New("Seedance actor authorization is required")
	ErrVolcAssetActorExpired               = errors.New("Seedance actor authorization has expired")
	ErrVolcAssetActorRevoked               = errors.New("Seedance actor authorization has been revoked")
)

var allowedVolcAssetActorDurations = map[int]struct{}{
	30:  {},
	60:  {},
	180: {},
	365: {},
}

type VolcAssetActor struct {
	Id                        int    `json:"id" gorm:"primaryKey"`
	UserId                    int    `json:"-" gorm:"index;not null"`
	DisplayName               string `json:"display_name" gorm:"type:varchar(128);not null"`
	GroupId                   string `json:"-" gorm:"type:text"`
	Status                    string `json:"status" gorm:"type:varchar(16);index;not null"`
	AuthorizationDurationDays *int   `json:"authorization_duration_days,omitempty"`
	AuthorizedAt              *int64 `json:"authorized_at,omitempty" gorm:"type:bigint"`
	AuthorizationExpiresAt    *int64 `json:"authorization_expires_at,omitempty" gorm:"type:bigint;index"`
	ConsentVersion            string `json:"consent_version,omitempty" gorm:"type:varchar(32)"`
	IsDefault                 bool   `json:"is_default" gorm:"index;not null"`
	CreatedAt                 int64  `json:"created_at" gorm:"type:bigint;not null"`
	UpdatedAt                 int64  `json:"updated_at" gorm:"type:bigint;not null"`
}

type VolcAssetActorAsset struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"-" gorm:"index;not null"`
	ActorId   int    `json:"actor_id" gorm:"index;not null"`
	AssetId   string `json:"asset_id" gorm:"type:varchar(191);uniqueIndex;not null"`
	CreatedAt int64  `json:"created_at" gorm:"type:bigint;not null"`
	UpdatedAt int64  `json:"updated_at" gorm:"type:bigint;not null"`
}

func validateVolcAssetActorDuration(days *int) error {
	if days == nil {
		return nil
	}
	if _, ok := allowedVolcAssetActorDurations[*days]; !ok {
		return ErrVolcAssetInvalidDuration
	}
	return nil
}

func CreateVolcAssetActor(userId int, displayName string, durationDays *int, now int64) (*VolcAssetActor, error) {
	displayName = strings.TrimSpace(displayName)
	if userId <= 0 || displayName == "" || len([]rune(displayName)) > 128 {
		return nil, fmt.Errorf("invalid Seedance actor")
	}
	if err := validateVolcAssetActorDuration(durationDays); err != nil {
		return nil, err
	}
	actor := &VolcAssetActor{
		UserId:                    userId,
		DisplayName:               displayName,
		Status:                    VolcAssetActorPending,
		AuthorizationDurationDays: durationDays,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
	if err := DB.Create(actor).Error; err != nil {
		return nil, err
	}
	return actor, nil
}

func ListVolcAssetActors(userId int) ([]VolcAssetActor, error) {
	if userId <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	actors := make([]VolcAssetActor, 0)
	err := DB.Where("user_id = ?", userId).Order("is_default DESC, id ASC").Find(&actors).Error
	return actors, err
}

func GetVolcAssetActorForUser(userId, actorId int) (*VolcAssetActor, error) {
	if userId <= 0 || actorId <= 0 {
		return nil, ErrVolcAssetActorNotFound
	}
	var actor VolcAssetActor
	err := DB.Where("user_id = ? AND id = ?", userId, actorId).First(&actor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVolcAssetActorNotFound
	}
	if err != nil {
		return nil, err
	}
	return &actor, nil
}

func GetDefaultVolcAssetActor(userId int) (*VolcAssetActor, error) {
	if userId <= 0 {
		return nil, ErrVolcAssetActorNotFound
	}
	var actor VolcAssetActor
	err := DB.Where("user_id = ? AND is_default = ?", userId, true).First(&actor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVolcAssetActorNotFound
	}
	if err != nil {
		return nil, err
	}
	return &actor, nil
}

func RenameVolcAssetActor(userId, actorId int, displayName string, now int64) error {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len([]rune(displayName)) > 128 {
		return fmt.Errorf("invalid Seedance actor display name")
	}
	result := DB.Model(&VolcAssetActor{}).
		Where("user_id = ? AND id = ?", userId, actorId).
		Updates(map[string]any{"display_name": displayName, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrVolcAssetActorNotFound
	}
	return nil
}

func ActivateVolcAssetActor(userId, actorId int, groupId, consentVersion string, now int64) error {
	groupId = strings.TrimSpace(groupId)
	consentVersion = strings.TrimSpace(consentVersion)
	if groupId == "" || consentVersion == "" {
		return fmt.Errorf("invalid Seedance actor authorization")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var actor VolcAssetActor
		err := tx.Where("user_id = ? AND id = ?", userId, actorId).First(&actor).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrVolcAssetActorNotFound
		}
		if err != nil {
			return err
		}
		var expiresAt *int64
		if actor.AuthorizationDurationDays != nil {
			value := now + int64(*actor.AuthorizationDurationDays)*86400
			expiresAt = &value
		}
		updates := map[string]any{
			"group_id":                 groupId,
			"status":                   VolcAssetActorActive,
			"authorized_at":            now,
			"authorization_expires_at": expiresAt,
			"consent_version":          consentVersion,
			"updated_at":               now,
		}
		return tx.Model(&actor).Updates(updates).Error
	})
}

func RequireVolcAssetActorActiveAt(userId, actorId int, now int64) (*VolcAssetActor, error) {
	actor, err := GetVolcAssetActorForUser(userId, actorId)
	if err != nil {
		return nil, err
	}
	if actor.Status == VolcAssetActorRevoked {
		return nil, ErrVolcAssetActorRevoked
	}
	if actor.AuthorizationExpiresAt != nil && *actor.AuthorizationExpiresAt <= now {
		_ = DB.Model(actor).Where("status <> ?", VolcAssetActorExpired).Updates(map[string]any{
			"status":     VolcAssetActorExpired,
			"updated_at": now,
		}).Error
		return nil, ErrVolcAssetActorExpired
	}
	if actor.Status != VolcAssetActorActive || strings.TrimSpace(actor.GroupId) == "" {
		return nil, ErrVolcAssetActorAuthorizationRequired
	}
	return actor, nil
}

func RevokeVolcAssetActor(userId, actorId int, now int64) error {
	result := DB.Model(&VolcAssetActor{}).
		Where("user_id = ? AND id = ?", userId, actorId).
		Updates(map[string]any{"status": VolcAssetActorRevoked, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrVolcAssetActorNotFound
	}
	return nil
}

func SaveVolcAssetActorAsset(userId, actorId int, assetId string, now int64) error {
	assetId = strings.TrimSpace(assetId)
	if userId <= 0 || actorId <= 0 || assetId == "" {
		return fmt.Errorf("invalid Seedance actor asset mapping")
	}
	if _, err := GetVolcAssetActorForUser(userId, actorId); err != nil {
		return err
	}
	mapping := VolcAssetActorAsset{
		UserId: userId, ActorId: actorId, AssetId: assetId, CreatedAt: now, UpdatedAt: now,
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "asset_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"user_id": userId, "actor_id": actorId, "updated_at": now,
		}),
	}).Create(&mapping).Error
}

func FindVolcAssetActorByAsset(userId int, assetId string) (*VolcAssetActor, error) {
	assetId = strings.TrimSpace(assetId)
	if userId <= 0 || assetId == "" {
		return nil, ErrVolcAssetActorNotFound
	}
	var actor VolcAssetActor
	err := DB.Table("volc_asset_actors").
		Select("volc_asset_actors.*").
		Joins("JOIN volc_asset_actor_assets ON volc_asset_actor_assets.actor_id = volc_asset_actors.id").
		Where("volc_asset_actor_assets.user_id = ? AND volc_asset_actor_assets.asset_id = ? AND volc_asset_actors.user_id = ?", userId, assetId, userId).
		First(&actor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVolcAssetActorNotFound
	}
	if err != nil {
		return nil, err
	}
	return &actor, nil
}

func MigrateVolcAssetUserGroupsToActors() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var bindings []VolcAssetUserGroup
		if err := tx.Find(&bindings).Error; err != nil {
			return err
		}
		for _, binding := range bindings {
			var count int64
			if err := tx.Model(&VolcAssetActor{}).
				Where("user_id = ? AND is_default = ?", binding.UserId, true).
				Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			createdAt := binding.CreatedAt
			if createdAt == 0 {
				createdAt = binding.UpdatedAt
			}
			actor := VolcAssetActor{
				UserId:      binding.UserId,
				DisplayName: "默认演员",
				GroupId:     binding.GroupId,
				Status:      VolcAssetActorLegacyReview,
				IsDefault:   true,
				CreatedAt:   createdAt,
				UpdatedAt:   binding.UpdatedAt,
			}
			if err := tx.Create(&actor).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
