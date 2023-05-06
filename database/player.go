package database

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/minotar/minecraft"
	"github.com/sirupsen/logrus"
	"github.com/synchthia/systera-api/systerapb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PlayerData - PlayerProfile on Database
type Players struct {
	ID                 int32  `gorm:"primary_key;AutoIncrement;"`
	UUID               string `gorm:"unique;"`
	Name               string
	NameLower          string
	Groups             []GroupNames         `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Stats              PlayerStats          `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	KnownUsernames     []KnownUsernames     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	KnownUsernameLower []KnownUsernameLower `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	KnownAddresses     []PlayerAddresses    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Settings           PlayerSettings       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type GroupNames struct {
	ID        int32 `gorm:"primary_key;AutoIncrement;"`
	PlayersID int32 // foreignKey
	Name      string
}

type KnownUsernames struct {
	ID        int32 `gorm:"primary_key;AutoIncrement;"`
	PlayersID int32 // foreignKey
	Username  string
	Time      int64
}

type KnownUsernameLower struct {
	ID        int32 `gorm:"primary_key;AutoIncrement;"`
	PlayersID int32 // foreignKey
	Username  string
	Time      int64
}

// ToProtobuf - Convert to Protobuf Entry
func (p *Players) ToProtobuf() *systerapb.PlayerEntry {
	var groups []string
	for _, g := range p.Groups {
		groupJson, _ := json.Marshal(g)
		groups = append(groups, string(groupJson))
	}
	return &systerapb.PlayerEntry{
		Uuid:     p.UUID,
		Name:     p.Name,
		Groups:   groups,
		Settings: p.Settings.ToProtobuf(),
		Stats:    p.Stats.ToProtobuf(),
	}
}

// PlayerSettings - Settings in PlayerProfile
type PlayerSettings struct {
	PlayersID   int32 `gorm:"primary_key;AutoIncrement;"` // foreignKey
	JoinMessage bool
	Vanish      bool
	Japanize    bool
}

// GetOrDefault - Get Value or Default
func (s PlayerSettings) GetOrDefault(v interface{}, d bool) bool {
	if w, ok := v.(bool); ok {
		return w
	}
	return d
}

// ToProtobuf - Convert to Protobuf Entry
func (s *PlayerSettings) ToProtobuf() *systerapb.PlayerSettings {
	return &systerapb.PlayerSettings{
		JoinMessage: s.GetOrDefault(s.JoinMessage, true),
		Vanish:      s.GetOrDefault(s.Vanish, false),
		Japanize:    s.GetOrDefault(s.Japanize, false),
	}
}

// FromProtobuf - Convert from Protobuf Entry
func (s *PlayerSettings) FromProtobuf(p *systerapb.PlayerSettings) *PlayerSettings {
	s.JoinMessage = p.JoinMessage
	s.Vanish = p.Vanish
	s.Japanize = p.Japanize

	return s
}

// TODO: Merge to 'getOrDefault' function when migrated to mongo-go-driver
// fillAbsent - Fill Default value
func (s *PlayerSettings) fillAbsent() {
	s.JoinMessage = s.GetOrDefault(s.JoinMessage, true)
	s.Vanish = s.GetOrDefault(s.Vanish, false)
	s.Japanize = s.GetOrDefault(s.Japanize, false)
}

// PlayerStats - Stats in PlayerProfile
type PlayerStats struct {
	ID            int32 `gorm:"primary_key;AutoIncrement;"`
	PlayersID     int32 // foreignKey
	CurrentServer string
	FirstLogin    int64
	LastLogin     int64
}

// ToProtobuf - Convert to Protobuf Entry
func (s *PlayerStats) ToProtobuf() *systerapb.PlayerStats {
	return &systerapb.PlayerStats{
		CurrentServer: s.CurrentServer,
		FirstLogin:    s.FirstLogin,
		LastLogin:     s.LastLogin,
	}
}

// PlayerAddresses - Player Address Entries
type PlayerAddresses struct {
	ID        int32 `gorm:"primary_key;AutoIncrement;"`
	PlayersID int32 // foreignKey
	Address   string
	Hostname  string
	Date      int64
	FirstSeen int64
	LastSeen  int64
}

// AltLookupData - AltLookup Result
type AltLookupData struct {
	UUID      string            `bson:"uuid"`
	Name      string            `bson:"name"`
	Addresses []PlayerAddresses `bson:"addresses"`
}

// PlayerIdentity - Player Data Set (Used from ex. punishment, report...)
type PlayerIdentity struct {
	ID            int32 `gorm:"primary_key;AutoIncrement;"`
	ReportID      int32
	PunishmentsID int32
	UUID          string
	Name          string
}

// ToProtobuf - Convert to Protobuf
func (pi *PlayerIdentity) ToProtobuf() *systerapb.PlayerIdentity {
	return &systerapb.PlayerIdentity{
		Uuid: pi.UUID,
		Name: pi.Name,
	}
}

// UUIDToName - Get Player Name from UUID
// Not Used
func (s *Mysql) UUIDToName(uuid string) string {
	return ""
	// if _, err := GetMongoSession(); err != nil {
	// 	return ""
	// }

	// session := session.Copy()
	// defer session.Close()
	// coll := session.DB("systera").C("players")

	// playerData := PlayerData{}
	// err := coll.Find(bson.M{"uuid": uuid}).One(&playerData)
	// if err != nil {
	// 	logrus.WithError(err).Errorf("[Player] Failed UUIDToName(%s)", uuid)
	// 	return ""
	// }

	// return playerData.Name
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
	r := s.client.Model(&Players{}).Preload("Groups").Preload("Stats").Preload("KnownUsernames").Preload("KnownUsernameLower").Preload("KnownAddresses").Preload("Settings").First(&player, "uuid = ?", uuid)
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Player] FP: Failed Failed get profile (%s)", uuid)
		return Players{}, r.Error
	}
	return player, nil
}

// FindPlayerByName - Find PlayerProfile from Name
func (s *Mysql) FindPlayerByName(name string) (Players, error) {
	var player Players
	r := s.client.Model(&Players{}).Preload("Groups").Preload("Stats").Preload("KnownUsernames").Preload("KnownUsernameLower").Preload("KnownAddresses").Preload("Settings").First(&player, "name_lower = ?", strings.ToLower(name))
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Player] FPBN: Failed Failed get profile %s", name)
		return Players{}, r.Error
	}
	return player, nil
}

// Migrate
func (s *Mysql) Migrate() {
	// Some Migration are here...
}

// InitPlayerProfile - Initialize Player Profile
func (s *Mysql) InitPlayerProfile(uuid, name, ipAddress, hostname string) (*Players, error) {
	nowtime := time.Now().UnixNano() / int64(time.Millisecond)

	var player Players
	r := s.client.First(&player, "uuid = ?", uuid)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		logrus.WithError(r.Error).Errorf("[Player] IPP: Failed Failed get profile %s(%s)", name, uuid)
		return &Players{}, r.Error
	}

	if r.RowsAffected == 0 {
		// Initialize
		player.KnownUsernames = []KnownUsernames{}
		player.KnownUsernameLower = []KnownUsernameLower{}
		player.Stats.FirstLogin = nowtime
		player.Settings = PlayerSettings{
			JoinMessage: true,
			Vanish:      false,
			Japanize:    false,
		}
	}

	// // User Profile
	player.UUID = uuid
	player.Name = name
	player.NameLower = strings.ToLower(name)
	player.Stats.LastLogin = nowtime

	// Player Groups
	if len(player.Groups) == 0 {
		var groupName GroupNames
		groupName.Name = "default"
		player.Groups = append(player.Groups, groupName)
	}

	// User Log (Address / Name)
	override := false
	var newPlayerAddresses []PlayerAddresses
	for _, v := range player.KnownAddresses {
		if v.Address == ipAddress {
			newPlayerAddresses = append(newPlayerAddresses, PlayerAddresses{
				Address:   ipAddress,
				Hostname:  hostname,
				FirstSeen: v.FirstSeen,
				LastSeen:  nowtime,
			})
			override = true
		} else {
			newPlayerAddresses = append(newPlayerAddresses, v)
		}
	}

	if !override {
		newPlayerAddresses = append(newPlayerAddresses, PlayerAddresses{
			Address:   ipAddress,
			Hostname:  hostname,
			FirstSeen: nowtime,
			LastSeen:  nowtime,
		})
	}

	player.KnownAddresses = newPlayerAddresses

	player.KnownUsernames = append(player.KnownUsernames, KnownUsernames{
		PlayersID: player.ID,
		Username:  name,
		Time:      nowtime,
	})
	player.KnownUsernameLower = append(player.KnownUsernameLower, KnownUsernameLower{
		PlayersID: player.ID,
		Username:  strings.ToLower(name),
		Time:      nowtime,
	})

	result := s.client.Clauses(clause.OnConflict{
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
	var r *gorm.DB
	r = s.client.Model(&Players{}).First(&player, "uuid = ?", uuid)
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Player] FP: Failed Failed get profile (%s)", uuid)
		return r.Error
	}

	r = s.client.Delete(&GroupNames{}, "players_id = ?", player.ID)
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Player] FP: Failed Failed get profile (%s)", uuid)
		return r.Error
	}

	r = s.client.Model(&Players{}).Preload("Groups").First(&player, "uuid = ?", uuid)
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Player] FP: Failed Failed get profile (%s)", uuid)
		return r.Error
	}

	var groupName GroupNames
	groupName.Name = "default"
	player.Groups = append(player.Groups, groupName)

	for _, g := range groups {
		if g != "default" {
			var groupsName GroupNames
			groupsName.Name = g
			player.Groups = append(player.Groups, groupsName)
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
	r := s.client.Model(&Players{}).Preload("Stats").First(&player, "uuid = ?", uuid)
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Player] SPServ: Failed Failed get profile (%s)", uuid)
		return r.Error
	}

	player.Stats.CurrentServer = server

	result := s.client.Save(&player.Stats)

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

	player.Settings.JoinMessage = settings.JoinMessage
	player.Settings.Vanish = settings.Vanish
	player.Settings.Japanize = settings.Japanize

	result := s.client.Save(&player.Settings)

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
func (s *Mysql) AltLookup(uuid string) ([]AltLookupData, error) {
	// Find Player
	var player Players
	r := s.client.Model(&Players{}).Preload("Groups").Preload("Stats").Preload("KnownUsernames").Preload("KnownUsernameLower").Preload("KnownAddresses").Preload("Settings").First(&player, "uuid = ?", uuid)
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Player] ALU: Failed Failed get profile (%s)", uuid)
		return nil, r.Error
	}

	var altLookupData []AltLookupData
	altLookupPair := make(map[string]AltLookupData)

	// Player's KnownAddresses
	for _, ipEntry := range player.KnownAddresses {
		var playerAddressesData []PlayerAddresses

		r := s.client.Model(&PlayerAddresses{}).Where("address = ?", ipEntry.Address).Find(&playerAddressesData)
		if r.Error != nil {
			continue
		}

		var altPlayerData []Players
		var playerids []int32
		for _, v := range playerAddressesData {
			playerids = append(playerids, v.PlayersID)
		}
		err := s.client.Model(&Players{}).Preload("KnownAddresses").Where("id IN ?", playerids).Find(&altPlayerData)
		if err.Error != nil {
			continue
		}

		// Loop in Found PlayerData
		for _, pd := range altPlayerData {

			if pd.UUID == uuid {
				continue
			}

			altLookupPair[pd.UUID] = AltLookupData{
				UUID:      pd.UUID,
				Name:      pd.Name,
				Addresses: append(altLookupPair[pd.UUID].Addresses, MatchPlayerAddress(pd.KnownAddresses, ipEntry.Address)),
			}
		}
	}

	for _, v := range altLookupPair {
		altLookupData = append(altLookupData, v)
	}

	return altLookupData, nil
}
