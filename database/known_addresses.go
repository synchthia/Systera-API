package database

import (
	"time"
)

// PlayerAddresses - Player Address Entries
type PlayerAddresses struct {
	ID         uint   `gorm:"primary_key;AutoIncrement;"`
	PlayerUUID string `gorm:"index;foreign_key;"` // foreignKey
	Address    string
	Hostname   string
	FirstSeen  time.Time `gorm:"type:datetime"`
	LastSeen   time.Time `gorm:"type:datetime"`
}

// 同じプレイヤーIDで同じアドレスならLastSeen上書き
// プレイヤーIDとアドレスの2つの組み合わせがユニークでなくてはならない
func (s *Mysql) UpdateKnownAddress(playerUUID, address, hostname string) error {
	nowtime := time.Now()

	var pa []PlayerAddresses

	// 同一UUIDで、同じアドレスの項目を取得
	r := s.client.Where("player_uuid = ?", playerUUID).Where("address = ?", address).Find(&pa)
	if r.Error != nil {
		return r.Error
	}

	// 存在しないUUID or アドレスなので新規追加
	if len(pa) == 0 {
		pa = append(pa, PlayerAddresses{
			PlayerUUID: playerUUID,
			Address:    address,
			Hostname:   hostname,
			FirstSeen:  nowtime,
			LastSeen:   nowtime,
		})

		if r := r.Create(&pa); r.Error != nil {
			return r.Error
		}
	} else {
		if r := r.Model(&pa).Updates(map[string]interface{}{"hostname": hostname, "last_seen": nowtime}); r.Error != nil {
			return r.Error
		}
	}

	return nil
}
