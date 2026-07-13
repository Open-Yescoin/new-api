package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VolcAssetUserGroup struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id" gorm:"uniqueIndex"`
	GroupId   string `json:"group_id" gorm:"type:text"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func GetVolcAssetUserGroup(userId int) (*VolcAssetUserGroup, error) {
	if userId <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	var binding VolcAssetUserGroup
	if err := DB.Where("user_id = ?", userId).First(&binding).Error; err != nil {
		return nil, err
	}
	return &binding, nil
}

func SaveVolcAssetUserGroup(userId int, groupId string) error {
	return saveVolcAssetUserGroup(DB, userId, groupId)
}

func saveVolcAssetUserGroup(tx *gorm.DB, userId int, groupId string) error {
	if userId <= 0 || groupId == "" {
		return fmt.Errorf("invalid Volcengine asset user group binding")
	}

	now := common.GetTimestamp()
	created := VolcAssetUserGroup{
		UserId:    userId,
		GroupId:   groupId,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"group_id":   groupId,
			"updated_at": now,
		}),
	}).Create(&created).Error
}
