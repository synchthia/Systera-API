package database

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/synchthia/systera-api/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// KnownUsernames - Player name change logs
type KnownUsernames struct {
	ID            uint   `gorm:"primary_key;AutoIncrement;"`
	PlayerUUID    string `gorm:"index;"`
	Username      string `gorm:"unique;"`
	UsernameLower string
	LastUsed      time.Time
}

func (s *Mysql) UpdateKnownUsername(playerUUID, username string) error {
	nowtime := time.Now()

	k := &KnownUsernames{
		PlayerUUID:    playerUUID,
		Username:      username,
		UsernameLower: strings.ToLower(username),
		LastUsed:      nowtime,
	}

	r := s.client.Model(&k).Where("player_uuid = ?", playerUUID).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "player_uuid"}},
		UpdateAll: true,
	}).Create(&k)

	if r.Error != nil {
		return r.Error
	}
	return nil
}

func (s *Mysql) GetIdentityByName(username string) (*PlayerIdentity, error) {
	var col *KnownUsernames
	r := s.client.Model(&col).
        Where("username_lower = ?", strings.ToLower(username)).
        Order("last_used DESC").First(&col)

	if r.Error != nil {
		if r.Error != gorm.ErrRecordNotFound {
            logrus.WithError(r.Error).Errorf("[GetIdentity] get identity failed: %v", username)
			return nil, r.Error
		} else {
			return nil, status.ErrPlayerNotFound.Error
		}
	}

	return &PlayerIdentity{
		UUID: col.PlayerUUID,
		Name: col.Username,
	}, r.Error
}
