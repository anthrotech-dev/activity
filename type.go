package activity

import "time"

type Activity struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	UserID    string    `json:"user_id" gorm:"index"`
	Timestamp time.Time `json:"timestamp" gorm:"type:timestamp without time zone"`
	Type      string    `json:"type" gorm:"type:text"`
	Body      string    `json:"body" gorm:"type:text"`
	IsSpecial bool      `json:"is_special" gorm:"type:boolean;default:false"`
}
