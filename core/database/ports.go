package database

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Port struct {
	gorm.Model

	ChallengeID uint
	PortNo      uint32 `gorm:"not null;unique"`
}

// Create an entry for the port in the Port table
// It returns an error if anything wrong happen during the
// transaction. If the entry already exists then it does not
// do anything and returns.
func PortEntryGetOrCreate(port Port, whereMap map[string]uint32) (Port, error) {
	tx := Db.Begin()

	if tx.Error != nil {
		return Port{}, fmt.Errorf("Error while starting transaction : %s", tx.Error)
	}

	var portEntry Port
	err := tx.Where(whereMap).First(&portEntry).Error
	if err != nil {
		return Port{}, fmt.Errorf("Error while get : %s", tx.Error)
	}

	if tx.RecordNotFound() {
		if err := tx.Create(&port).Error; err != nil {
			tx.Rollback()
			return Port{}, err
		}

		return port, tx.Commit().Error
	}

	return portEntry, tx.Commit().Error
}

func GetAllocatedPorts(challenge Challenge) ([]Port, error) {
	var ports []Port
	err := Db.Model(&challenge).Related(&ports).Error

	if err != nil {
		return nil, fmt.Errorf("Error while searching port for challenge : %s", err)
	}

	return ports, nil
}
