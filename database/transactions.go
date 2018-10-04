package database

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Transaction struct {
	gorm.Model
	Action string
	User   string
}
