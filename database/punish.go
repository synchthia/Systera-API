package database

import (
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/synchthia/systera-api/systerapb"
	"gorm.io/gorm"
)

// PunishLevel - Punishment level
type PunishLevel int32

const (
	// UNKNOWN - Unknown (Not handled?)
	UNKNOWN PunishLevel = iota

	//WARN - Warning
	WARN

	//KICK - Kick from Server
	KICK

	//TEMPBAN - Temporary BAN
	TEMPBAN

	//PERMBAN - Permanently BAN
	PERMBAN
)

// PunishmentData - PunishData on Database
type Punishments struct {
	ID                 uint `gorm:"primary_key;AutoIncrement;"`
	Available          bool
	Level              PunishLevel `gorm:"type:tinyint;"`
	Reason             string
	Date               time.Time `gorm:"type:datetime"`
	Expire             time.Time `gorm:"type:datetime"`
	PunisherPlayerUUID string
	PunisherPlayerName string
	TargetPlayerUUID   string `gorm:"index;"`
	TargetPlayerName   string
}

// PunishRule - Validation Rules (true -> Permit)
type PunishRule struct {
	NoProfile bool
	Duplicate bool
	Cooldown  bool
	Offline   bool
}

// ToProtobuf - Convert to Protobuf
func (p *Punishments) ToProtobuf() *systerapb.PunishEntry {
	return &systerapb.PunishEntry{
		Available: p.Available,
		Level:     p.Level.ToProtobuf(),
		Reason:    p.Reason,
		Date:      p.Date.UnixMilli(),
		Expire:    p.Expire.UnixMilli(),
		PunishedFrom: &systerapb.PlayerIdentity{
			Uuid: p.PunisherPlayerUUID,
			Name: p.PunisherPlayerName,
		},
		PunishedTo: &systerapb.PlayerIdentity{
			Uuid: p.TargetPlayerUUID,
			Name: p.TargetPlayerName,
		},
	}
}

func (i PunishLevel) String() string {
	switch i {
	case WARN:
		return "WARN"
	case KICK:
		return "KICK"
	case TEMPBAN:
		return "TEMPBAN"
	case PERMBAN:
		return "PERMBAN"
	default:
		return "UNKNOWN"
	}
}

// ToProtobuf - Convert to Protobuf
func (i PunishLevel) ToProtobuf() systerapb.PunishLevel {
	switch i {
	case WARN:
		return systerapb.PunishLevel_WARN
	case KICK:
		return systerapb.PunishLevel_KICK
	case TEMPBAN:
		return systerapb.PunishLevel_TEMPBAN
	case PERMBAN:
		return systerapb.PunishLevel_PERMBAN
	default:
		return systerapb.PunishLevel_UNKNOWN
	}
}

// GetPlayerPunishment - Get Player Punishment History
func (s *Mysql) GetPlayerPunishment(playerUUID string, filterLevel PunishLevel, includeExpired bool) ([]Punishments, error) {
	var punishments []Punishments

	nowtime := time.Now()

	// All results must be sorted this rules...
	// - level: low_level -> high_level
	// - date: old_date -> now_date
	if includeExpired {
		r := s.client.Model(&Punishments{}).
			Order("date ASC").
			Find(&punishments, "target_player_uuid = ? AND level >= ?", playerUUID, filterLevel)
		if r.Error != nil {
			logrus.WithError(r.Error).Errorf("[Punish] Failed GetPlayerPunishment(%s)", playerUUID)
			return nil, r.Error
		}
	} else {
		r := s.client.Model(&Punishments{}).
			Order("date ASC").
			Find(&punishments, "target_player_uuid = ? AND level >= ? AND available = true AND expire = '1970-01-01' OR expire >= ?", playerUUID, filterLevel, nowtime)
		if r.Error != nil {
			logrus.WithError(r.Error).Errorf("[Punish] Failed GetPlayerPunishment(%s)", playerUUID)
			return nil, r.Error
		}
	}

	return punishments, nil
}

// SetPlayerPunishment - Punish Player
func (s *Mysql) SetPlayerPunishment(force bool, from, to PlayerIdentity, level PunishLevel, reason string, date, expire int64) (success bool, result PunishRule, err error) {
	// Error Status
	success = false
	result = PunishRule{
		NoProfile: false,
		Duplicate: false,
		Cooldown:  false,
		Offline:   false,
	}

	availableBan, _ := s.GetPlayerPunishment(to.UUID, TEMPBAN, false)
	for _, ban := range availableBan {
		if ban.Level == PERMBAN {
			result.Duplicate = true
			return
		}

		if level <= TEMPBAN && ban.Level == TEMPBAN {
			result.Cooldown = true
		}
	}
	var player Players
	r := s.client.Model(&Players{}).Preload("Settings").First(&player, "uuid = ?", to.UUID)

	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		return success, result, r.Error
	}

	if r.Error != nil && r.Error == gorm.ErrRecordNotFound {
		result.NoProfile = true
	} else {
		if player.CurrentServer == "" {
			result.Offline = true
		}
	}

	// Check
	// NoProfile
	if !force && result.NoProfile {
		return
	}

	// Duplicate
	if result.Duplicate {
		return
	}

	// Cooldown
	if result.Cooldown {
		if level == TEMPBAN {
			return
		}
	}

	// Offline
	if !force && result.Offline {
		if level == WARN || level == KICK {
			return
		}
	}

	punishData := Punishments{
		Available:          true,
		Level:              level,
		Reason:             reason,
		Date:               time.UnixMilli(date),
		Expire:             time.UnixMilli(expire),
		PunisherPlayerUUID: from.UUID,
		PunisherPlayerName: from.Name,
		TargetPlayerUUID:   to.UUID,
		TargetPlayerName:   to.Name,
	}

	r = s.client.Create(&punishData)

	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Punish] Failed Punish Player")
		return
	}

	var expireDate string
	if expire != 0 {
		expireDate = time.Unix(expire/1000, 0).String()
	}

	logrus.WithFields(logrus.Fields{
		"level":  level,
		"reason": reason,
		"expire": expireDate,
	}).Infof("[Punishment] %s -> %s", from.Name, to.Name)

	return true, result, err
}

// UnBan - Disable available tempban/permban
func (s *Mysql) UnBan(targetUUID string) error {
	p, err := s.GetPlayerPunishment(targetUUID, TEMPBAN, false)
	if err != nil {
		return err
	}

	// Get latest
	if len(p) == 0 {
		return errors.New("player not punished")
	}

	latest := p[len(p)-1]
	latest.Available = false

	// r := s.client.Model(&latest).Update("available", false)
	r := s.client.Save(&latest)
	return r.Error
}
