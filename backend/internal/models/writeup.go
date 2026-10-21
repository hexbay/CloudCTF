package models

import "time"

type Writeup struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CompetitionID uint      `gorm:"not null;index:idx_writeup_competition_user,unique" json:"competition_id"`
	UserID        uint      `gorm:"not null;index:idx_writeup_competition_user,unique" json:"user_id"`
	TeamID        *uint     `gorm:"index" json:"team_id"`
	Title         string    `gorm:"size:200;not null" json:"title"`
	FilePath      string    `gorm:"size:500;not null" json:"file_path"`
	FileName      string    `gorm:"size:200;not null" json:"file_name"`
	FileSize      int64     `gorm:"not null;default:0" json:"file_size"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Writeup) TableName() string { return "writeups" }
