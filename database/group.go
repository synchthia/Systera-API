package database

import (
	"errors"

	"github.com/sirupsen/logrus"
	"github.com/synchthia/systera-api/systerapb"
)

// Group - Permission Group Data
type Groups struct {
	ID          int32  `gorm:"primary_key;AutoIncrement;"`
	Name        string `gorm:"index;not null;"`
	Prefix      string
	Permissions []PermissionsData `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// PermissionsData - Permission Data
type PermissionsData struct {
	ID         int32  `gorm:"primary_key;AutoIncrement;"`
	GroupsID   int32  // foreignKey
	ServerName string `gorm:"primary_key;"`
	Parmission string
}

// ToProtobuf - Convert to Protobuf
func (g *Groups) ToProtobuf(serverName string) *systerapb.GroupEntry {
	e := &systerapb.GroupEntry{
		GroupName:   g.Name,
		GroupPrefix: g.Prefix,
	}

	for _, p := range g.Permissions {
		if serverName != "" && p.ServerName == serverName {
			e.ServerPerms = append(e.ServerPerms, p.Parmission)
		}
		if p.ServerName == "global" {
			e.GlobalPerms = append(e.GlobalPerms, p.Parmission)
		}
	}

	return e
}

// GetGroupData - Get Group Entry
func (s *Mysql) GetGroupData(name string) (Groups, error) {
	group := Groups{}
	r := s.client.Find(&group, "name = ?", name)
	if r.Error != nil {
		return Groups{}, r.Error
	}

	return group, nil
}

// GetAllGroup - Find All Group Entry
func (s *Mysql) GetAllGroup() ([]Groups, error) {
	var groups []Groups
	r := s.client.Model(&Groups{}).Preload("Permissions").Find(&groups)
	if r.Error != nil {
		logrus.WithError(r.Error).Errorf("[Group] Failed Find GroupData: %s", r.Error)
		return nil, r.Error
	}

	return groups, nil
}

// CreateGroup - Create New Group
func (s *Mysql) CreateGroup(group Groups) error {
	r := s.client.First(&Groups{}, "name = ?", group.Name)

	if r.RowsAffected != 0 {
		return errors.New("group already exists")
	} else if r.Error != nil && r.RowsAffected != 0 {
		return r.Error
	}

	result := s.client.Create(&group)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

// RemoveGroup - Remove Group
func (s *Mysql) RemoveGroup(groupName string) error {
	r := s.client.Select("PermissionsData").Delete(&Groups{}, "name = ?", groupName)
	if r.Error != nil {
		return r.Error
	}

	return nil
}

// AddPermission - Add Permission
func (s *Mysql) AddPermission(groupName, target string, permissions []string) error {
	var group Groups
	r := s.client.First(&group, "name = ?", groupName)
	if r.Error != nil {
		return r.Error
	}

	var dbPerms []PermissionsData
	for _, v := range permissions {
		parm := PermissionsData{
			GroupsID:   group.ID,
			ServerName: target,
			Parmission: v,
		}
		dbPerms = append(dbPerms, parm)
	}

	result := s.client.Create(&dbPerms)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

// RemovePermission - Remove Permission
func (s *Mysql) RemovePermission(groupName, target string, permissions []string) error {
	var group Groups
	r := s.client.First(&group, "name = ?", groupName)
	if r.Error != nil {
		return r.Error
	}

	for _, v := range permissions {
		result := s.client.Delete(&PermissionsData{}, "groups_id = ? and server_name = ? and parmission = ?", group.ID, target, v)
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}
