package database

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minotar/minecraft"
	"github.com/sirupsen/logrus"
	"github.com/synchthia/systera-api/systerapb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PlayerData - PlayerProfile on Database
type Players struct {
	ID            uuid.UUID `gorm:"type:uuid;default:UUID();"`
	UUID          string    `gorm:"index;unique;"`
	Name          string    `gorm:"index;not null;"`
	NameLower     string
	CurrentServer string
	FirstLogin    time.Time `gorm:"type:datetime"`
	LastLogin     time.Time `gorm:"type:datetime"`
	Groups        string
	IgnoreList []
	Settings      PlayerSettings `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// ToProtobuf - Convert to Protobuf Entry
func (p *Players) ToProtobuf() *systerapb.PlayerEntry {
	return &systerapb.PlayerEntry{
		Uuid:          p.UUID,
		Name:          p.Name,
		CurrentServer: p.CurrentServer,
		FirstLogin:    p.FirstLogin.UnixMilli(),
		LastLogin:     p.LastLogin.UnixMilli(),
		Groups:        strings.Split(p.Groups, ","),
		Settings:      p.Settings.ToProtobuf(),
	}
}

// PlayerSettings - Settings in PlayerProfile
type PlayerSettings struct {
	PlayersID   uuid.UUID `gorm:"foreign_key;unique;type:uuid;default:UUID();"` // foreignKey
	JoinMessage bool
	Vanish      bool
	Japanize    bool
	GlobalChat  bool
}

// ToProtobuf - Convert to Protobuf Entry
func (s *PlayerSettings) ToProtobuf() *systerapb.PlayerSettings {
	return &systerapb.PlayerSettings{
		JoinMessage: s.GetOrDefault(s.JoinMessage, true),
		Vanish:      s.GetOrDefault(s.Vanish, false),
		Japanize:    s.GetOrDefault(s.Japanize, true),
		GlobalChat:  s.GetOrDefault(s.GlobalChat, true),
	}
}

// FromProtobuf - Convert from Protobuf Entry
func (s *PlayerSettings) FromProtobuf(p *systerapb.PlayerSettings) *PlayerSettings {
	s.JoinMessage = p.JoinMessage
	s.Vanish = p.Vanish
	s.Japanize = p.Japanize
	s.GlobalChat = p.GlobalChat

	return s
}

// GetOrDefault - Get Value or Default
func (s PlayerSettings) GetOrDefault(v interface{}, d bool) bool {
	if w, ok := v.(bool); ok {
		return w
	}
	return d
}

// AltLookupData - AltLookup Result
type AltLookupData struct {
	UUID      string
	Name      string
	Addresses []PlayerAddresses
}

// PlayerIdentity - Player Data Set (Used from ex. punishment, report...)
type PlayerIdentity struct {
	UUID string
	Name string
}

// IgnoreEntry - Ignore chat entry
type IgnoreEntry struct {
	UUID string `gorm:"foreign_key"`
	Name string
}

// ToProtobuf - Convert to Protobuf
func (pi *PlayerIdentity) ToProtobuf() *systerapb.PlayerIdentity {
	return &systerapb.PlayerIdentity{
		Uuid: pi.UUID,
		Name: pi.Name,
	}
}

// NameToUUID - Get UUID from Player Name
func (s *Mysql) NameToUUID(name string) (string, error) {
	var player Players
	r := s.client.First(&player, "name_lower = ?", strings.ToLower(name))
	if r.Error != nil && r.Error == gorm.ErrRecordNotFound {
		uuid, err := NameToUUIDwithMojang(name)
		return uuid, err
	} else if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		logrus.WithError(r.Error).Errorf("[Player] NTU: Failed Failed get profile %s", name)
		return "", r.Error
	}

	return player.UUID, nil
}

// NameToUUIDwithMojang - Get UUID from Mojang API
func NameToUUIDwithMojang(name string) (string, error) {
	uuid, err := minecraft.NewMinecraft().GetUUID(name)
	if err != nil {
		return "", err
	}

	return uuid, nil
}

// FindPlayer - Find PlayerProfile
func (s *Mysql) FindPlayer(uuid string) (Players, error) {
	var player Players
	r := s.client.Model(&Players{}).Preload("Settings").First(&player, "uuid = ?", uuid)
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Player] FP: Failed Failed get profile (%s)", uuid)
		return Players{}, r.Error
	}
	return player, nil
}

// FindPlayerByName - Find PlayerProfile from Name
func (s *Mysql) FindPlayerByName(name string) (Players, error) {
	var player Players
	r := s.client.Model(&Players{}).Preload("Settings").First(&player, "name_lower = ?", strings.ToLower(name))
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Player] FPBN: Failed Failed get profile %s", name)
		return Players{}, r.Error
	}
	return player, nil
}

// InitPlayerProfile - Initialize Player Profile
func (s *Mysql) InitPlayerProfile(uuid, name, ipAddress, hostname string) (*Players, error) {
	nowtime := time.Now()

	var player Players
	r := s.client.Preload("Settings").First(&player, "uuid = ?", uuid)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		logrus.WithError(r.Error).Errorf("[Player] IPP: Failed Failed get profile %s(%s)", name, uuid)
		return &Players{}, r.Error
	}

	if r.RowsAffected == 0 {
		// Initialize
		player.FirstLogin = nowtime
		player.Settings = PlayerSettings{
			JoinMessage: true,
			Japanize:    false,
			Vanish:      false,
			GlobalChat:  true,
		}
	}

	// User Profile
	player.UUID = uuid
	player.Name = name
	player.NameLower = strings.ToLower(name)
	player.LastLogin = nowtime

	// Player Groups
	if len(player.Groups) == 0 {
		player.Groups = "default"
	}

	// Update address log
	if err := s.UpdateKnownAddress(uuid, ipAddress, hostname); err != nil {
		return nil, err
	}

	// Update username log
	if err := s.UpdateKnownUsername(uuid, name); err != nil {
		return nil, err
	}

	result := s.client.Preload("Settings").Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "uuid"}},
		UpdateAll: true,
	}).Create(&player)

	if result.Error != nil {
		logrus.WithError(result.Error).Errorf("[Player] IPP: Failed to create profile %s(%s)", name, uuid)
		return &Players{}, result.Error
	}

	logrus.WithFields(logrus.Fields{
		"name":    name,
		"uuid":    uuid,
		"address": ipAddress,
	}).Infof("[Player] InitPlayerProfile")

	return &player, nil
}

// SetPlayerGroups - Define Player Groups
func (s *Mysql) SetPlayerGroups(uuid string, groups []string) error {
	var player Players
	r := s.client.Model(&Players{}).First(&player, "uuid = ?", uuid)
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Player] FP: Failed Failed get profile (%s)", uuid)
		return r.Error
	}

	player.Groups = "default"

	for _, g := range groups {
		if g != "default" {
			player.Groups += "," + g
		}
	}

	result := s.client.Save(&player)

	if result.Error != nil {
		logrus.WithError(result.Error).Errorf("[Player] Failed Execute SetPlayerGroups")
		return result.Error
	}

	return nil
}

// SetPlayerServer - Define Player Current Server
func (s *Mysql) SetPlayerServer(uuid, server string) error {
	var player Players
	r := s.client.Model(&Players{}).First(&player, "uuid = ?", uuid)
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Player] SPServ: Failed Failed get profile (%s)", uuid)
		return r.Error
	}

	player.CurrentServer = server

	result := s.client.Save(&player)

	if result.Error != nil {
		logrus.WithError(result.Error).Errorf("[Player] Failed Execute SetPlayerServer")
		return result.Error
	}

	logrus.WithFields(logrus.Fields{
		"server": server,
	}).Debugf("[Player] Set Server: %s > %s(%s)", server, player.Name, uuid)

	return nil
}

// SetPlayerSettings - Set Player Settings
func (s *Mysql) SetPlayerSettings(uuid string, settings *PlayerSettings) error {
	var player Players
	r := s.client.Model(&Players{}).Preload("Settings").First(&player, "uuid = ?", uuid)
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Player] SPS: Failed Failed get profile (%s)", uuid)
		return r.Error
	}

	player.Settings.PlayersID = player.ID

	// Settings
	player.Settings.JoinMessage = settings.JoinMessage
	player.Settings.Vanish = settings.Vanish
	player.Settings.Japanize = settings.Japanize
	player.Settings.GlobalChat = settings.GlobalChat

	result := s.client.Clauses((clause.OnConflict{
		Columns:   []clause.Column{{Name: "players_id"}},
		UpdateAll: true,
	})).Create(&player.Settings)

	if result.Error != nil {
		logrus.WithError(result.Error).Errorf("[Player] Failed Execute SetPlayerSettings %s(%s)", player.Name, uuid)
		return result.Error
	}

	return nil
}

// MatchPlayerAddress - Return Matched Address in Array
func MatchPlayerAddress(playerAddresses []PlayerAddresses, address string) PlayerAddresses {
	for _, v := range playerAddresses {
		if v.Address == address {
			return v
		}
	}
	return PlayerAddresses{}
}

// AltLookup - AltLookup player accounts
func (s *Mysql) AltLookup(playerUUID string) ([]AltLookupData, error) {
	return nil, nil
	// // Find Player
	// var player Players
	// r := s.client.Model(&Players{}).Preload("Settings").First(&player, "uuid = ?", playerUUID)
	// if r.Error != nil {
	// 	logrus.WithError(r.Error).Errorf("[Player] ALU: Failed Failed get profile (%s)", playerUUID)
	// 	return nil, r.Error
	// }

	// var altLookupData []AltLookupData
	// altLookupPair := make(map[string]AltLookupData)

	// // Player's KnownAddresses
	// for _, ipEntry := range player.KnownAddresses {
	// 	var playerAddressesData []PlayerAddresses

	// 	r := s.client.Model(&PlayerAddresses{}).Where("address = ?", ipEntry.Address).Find(&playerAddressesData)
	// 	if r.Error != nil {
	// 		continue
	// 	}

	// 	var altPlayerData []Players
	// 	var playerids []uuid.UUID
	// 	for _, v := range playerAddressesData {
	// 		playerids = append(playerids, v.PlayersID)
	// 	}
	// 	err := s.client.Model(&Players{}).Where("id IN ?", playerids).Find(&altPlayerData)
	// 	if err.Error != nil {
	// 		continue
	// 	}

	// 	// Loop in Found PlayerData
	// 	for _, pd := range altPlayerData {

	// 		if pd.UUID == playerUUID {
	// 			continue
	// 		}

	// 		altLookupPair[pd.UUID] = AltLookupData{
	// 			UUID:      pd.UUID,
	// 			Name:      pd.Name,
	// 			Addresses: append(altLookupPair[pd.UUID].Addresses, MatchPlayerAddress(pd.KnownAddresses, ipEntry.Address)),
	// 		}
	// 	}
	// }

	// for _, v := range altLookupPair {
	// 	altLookupData = append(altLookupData, v)
	// }

	// return altLookupData, nil
}
