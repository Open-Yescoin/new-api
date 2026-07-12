package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
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
	if userId <= 0 || groupId == "" {
		return fmt.Errorf("invalid Volcengine asset user group binding")
	}

	var existing VolcAssetUserGroup
	err := DB.Where("user_id = ?", userId).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	now := common.GetTimestamp()
	created := VolcAssetUserGroup{
		UserId:    userId,
		GroupId:   groupId,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := DB.Create(&created).Error; err != nil {
		// Another request may have created the unique user binding first.
		if _, readErr := GetVolcAssetUserGroup(userId); readErr == nil {
			return nil
		}
		return err
	}
	return nil
}
