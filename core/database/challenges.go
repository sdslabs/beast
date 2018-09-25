package database

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Challenge struct {
	gorm.Model

	ChallengeId string
	Name        string
	Author      string
	Format      string
	ContainerId string `gorm:"size:64"`
	ImageId     string `gorm:"size:64"`
	Ports       string
	Status      string
}
