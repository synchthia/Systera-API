package database

import (
	"strings"
	"time"

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
