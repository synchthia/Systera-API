package database

import (
	"strings"
	"time"

	"github.com/minotar/minecraft"
	"github.com/synchthia/systera-api/systerapb"

	"github.com/globalsign/mgo/bson"
	"github.com/sirupsen/logrus"
)

// PlayerData - PlayerProfile on Database
type PlayerData struct {
	ID                 bson.ObjectId     `bson:"_id,omitempty"`
	UUID               string            `bson:"uuid" json:"id"`
	Name               string            `bson:"name"`
	NameLower          string            `bson:"name_lower"`
	Groups             []string          `bson:"groups"`
	Stats              PlayerStats       `bson:"stats"`
	KnownUsernames     map[string]int64  `bson:"known_usernames"`
	KnownUsernameLower map[string]int64  `bson:"known_usernames_lower"`
	KnownAddresses     []PlayerAddresses `bson:"known_addresses"`
	Settings           *PlayerSettings   `bson:"settings,omitempty"`
}

// ToProtobuf - Convert to Protobuf Entry
func (p *PlayerData) ToProtobuf() *systerapb.PlayerEntry {
	return &systerapb.PlayerEntry{
		Uuid:     p.UUID,
		Name:     p.Name,
		Groups:   p.Groups,
		Settings: p.Settings.ToProtobuf(),
		Stats:    p.Stats.ToProtobuf(),
	}
}

// PlayerSettings - Settings in PlayerProfile
type PlayerSettings struct {
	JoinMessage interface{} `bson:"join_message,omitempty"`
	Vanish      interface{} `bson:"vanish,omitempty"`
	Japanize    interface{} `bson:"japanize,omitempty"`
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
	s.JoinMessage = &p.JoinMessage
	s.Vanish = &p.Vanish
	s.Japanize = &p.Japanize

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
	CurrentServer string `bson:"current_server"`
	FirstLogin    int64  `bson:"first_login"`
	LastLogin     int64  `bson:"last_login"`
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
	Address   string `bson:"address"`
	Hostname  string `bson:"hostname"`
	Date      int64  `bson:"date,omitempty"`
	FirstSeen int64  `bson:"first_seen"`
	LastSeen  int64  `bson:"last_seen"`
}

// AltLookupData - AltLookup Result
type AltLookupData struct {
	UUID      string            `bson:"uuid"`
	Name      string            `bson:"name"`
	Addresses []PlayerAddresses `bson:"addresses"`
}

// PlayerIdentity - Player Data Set (Used from ex. punishment, report...)
type PlayerIdentity struct {
	UUID string `bson:"uuid"`
	Name string `bson:"name"`
}

// ToProtobuf - Convert to Protobuf
func (pi *PlayerIdentity) ToProtobuf() *systerapb.PlayerIdentity {
	return &systerapb.PlayerIdentity{
		Uuid: pi.UUID,
		Name: pi.Name,
	}
}

// UUIDToName - Get Player Name from UUID
func UUIDToName(uuid string) string {
	if _, err := GetMongoSession(); err != nil {
		return ""
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	playerData := PlayerData{}
	err := coll.Find(bson.M{"uuid": uuid}).One(&playerData)
	if err != nil {
		logrus.WithError(err).Errorf("[Player] Failed UUIDToName(%s)", uuid)
		return ""
	}

	return playerData.Name
}

// NameToUUID - Get UUID from Player Name
func NameToUUID(name string) (string, error) {
	if _, err := GetMongoSession(); err != nil {
		return "", err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	playerData := PlayerData{}
	err := coll.Find(bson.M{"name_lower": strings.ToLower(name)}).One(&playerData)
	if err != nil {
		uuid, err := NameToUUIDwithMojang(name)
		return uuid, err
	}

	return playerData.UUID, nil
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
func FindPlayer(uuid string) (PlayerData, error) {
	if _, err := GetMongoSession(); err != nil {
		return PlayerData{}, err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")
	playerData := PlayerData{}

	err := coll.Find(bson.M{"uuid": uuid}).One(&playerData)
	return playerData, err
}

// FindPlayerByName - Find PlayerProfile from Name
func FindPlayerByName(name string) (PlayerData, error) {
	if _, err := GetMongoSession(); err != nil {
		return PlayerData{}, err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")
	playerData := PlayerData{}

	nameLower := strings.ToLower(name)
	err := coll.Find(bson.M{"name_lower": nameLower}).Sort("-stats.last_login").One(&playerData)
	return playerData, err
}

// Migrate
func Migrate() {
	// Some Migration are here...
}

// InitPlayerProfile - Initialize Player Profile
func InitPlayerProfile(uuid, name, ipAddress, hostname string) (*PlayerData, error) {
	if _, err := GetMongoSession(); err != nil {
		return nil, err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	nowtime := time.Now().UnixNano() / int64(time.Millisecond)

	profileCnt, err := coll.Find(bson.M{"uuid": uuid}).Count()
	if err != nil {
		logrus.WithError(err).Errorf("[Player] IPP: Failed Failed get profile %s(%s)", name, uuid)
		return nil, err
	}

	playerData := &PlayerData{}

	if profileCnt == 0 {
		// Initialize
		playerData.KnownUsernames = make(map[string]int64)
		playerData.KnownUsernameLower = make(map[string]int64)
		playerData.Stats.FirstLogin = nowtime
	} else {
		coll.Find(bson.M{"uuid": uuid}).One(playerData)
	}

	// User Profile
	playerData.UUID = uuid
	playerData.Name = name
	playerData.NameLower = strings.ToLower(name)
	playerData.Stats.LastLogin = nowtime

	// Settings -> Fill Absent Entries
	playerData.Settings.fillAbsent()

	// Player Groups
	if len(playerData.Groups) == 0 {
		playerData.Groups = []string{"default"}
	}

	// User Log (Address / Name)
	override := false
	var newPlayerAddresses []PlayerAddresses
	for _, v := range playerData.KnownAddresses {
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

	playerData.KnownAddresses = newPlayerAddresses

	playerData.KnownUsernames[name] = nowtime
	playerData.KnownUsernameLower[strings.ToLower(name)] = nowtime

	//coll.Upsert(bson.M{"uuid": uuid}, playerData)
	coll.Upsert(
		bson.M{"uuid": uuid},
		bson.M{"$set": playerData},
	)

	logrus.WithFields(logrus.Fields{
		"name":    name,
		"uuid":    uuid,
		"address": ipAddress,
	}).Infof("[Player] InitPlayerProfile")

	return playerData, nil
}

// SetPlayerGroups - Define Player Groups
func SetPlayerGroups(uuid string, groups []string) error {
	if _, err := GetMongoSession(); err != nil {
		return err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	var newGroups []string
	newGroups = append(newGroups, "default")

	for _, g := range groups {
		if g != "default" {
			newGroups = append(newGroups, g)
		}
	}

	err := coll.Update(bson.M{"uuid": uuid}, bson.M{"$set": bson.M{"groups": newGroups}})
	if err != nil {
		logrus.WithError(err).Errorf("[Player] Failed Execute SetPlayerGroups")
		return err
	}

	return nil
}

// SetPlayerServer - Define Player Current Server
func SetPlayerServer(uuid, server string) error {
	if _, err := GetMongoSession(); err != nil {
		return err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	playerData := PlayerData{}
	coll.Find(bson.M{"uuid": uuid}).One(&playerData)

	err := coll.Update(bson.M{"uuid": uuid}, bson.M{"$set": bson.M{"stats.current_server": server}})
	if err != nil {
		logrus.WithError(err).Errorf("[Player] Failed Execute SetPlayerServer")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"server": server,
	}).Debugf("[Player] Set Server: %s > %s(%s)", server, playerData.Name, uuid)

	return nil
}

// SetPlayerSettings - Set Player Settings
func SetPlayerSettings(uuid string, settings *PlayerSettings) error {
	if _, err := GetMongoSession(); err != nil {
		return err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	_, err := coll.Upsert(bson.M{"uuid": uuid}, bson.M{"$set": bson.M{"settings": settings}})
	if err != nil {
		logrus.WithError(err).Errorf("[Player] Failed Set Player Settings")
	}

	return err
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
func AltLookup(uuid string) ([]AltLookupData, error) {
	if _, err := GetMongoSession(); err != nil {
		return nil, err
	}

	session := session.Copy()
	defer session.Close()
	coll := session.DB("systera").C("players")

	playerData := PlayerData{}

	var altLookupData []AltLookupData
	//playerAddressPair := make(map[string][]PlayerAddresses)
	altLookupPair := make(map[string]AltLookupData)

	// Find Player
	err := coll.Find(bson.M{"uuid": uuid}).One(&playerData)
	if err != nil {
		return nil, err
	}

	// Player's KnownAddresses
	for _, ipEntry := range playerData.KnownAddresses {
		var altPlayerData []PlayerData

		err := coll.Find(bson.M{"known_addresses.address": ipEntry.Address}).All(&altPlayerData)
		if err != nil {
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
