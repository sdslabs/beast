package database

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Port struct {
	gorm.Model

	ChallengeID uint   `gorm:"not null"`
	PortNo      uint32 `gorm:"not null;unique"`
}

// Create an entry for the port in the Port table
// It returns an error if anything wrong happen during the
// transaction. If the entry already exists then it does not
// do anything and returns.
func PortEntryGetOrCreate(port *Port) (Port, error) {
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	if tx.Error != nil {
		return Port{}, fmt.Errorf("Error while starting transaction : %s", tx.Error)
	}

	err := tx.FirstOrCreate(port, *port).Error
	if err != nil {
		tx.Rollback()
		return Port{}, err
	}

	return *port, tx.Commit().Error
}

func GetAllocatedPorts(challenge Challenge) ([]Port, error) {
	var ports []Port

	DBMux.Lock()
	defer DBMux.Unlock()

	if err := Db.Model(&challenge).Association("Ports").Find(&ports); err != nil {
		return nil, fmt.Errorf("error while searching port for challenge : %s", err)
	}

	return ports, nil
}

func UpdatePorts(challenge *Challenge) (error) {
	var ports []Port

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	if err := tx.Model(&challenge).Association("Ports").Find(&ports); err != nil {
		tx.Rollback()
		return fmt.Errorf("error while searching port for challenge : %s", err)
	}
	tx.Commit()

	if err := Db.Unscoped().Where("challenge_id = ?", challenge.ID).Delete(ports); err != nil {
		return err.Error
	}

	return nil
}

func DeleteRelatedPorts(portList []Port) error {

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while starting transaction : %s", tx.Error)
	}

	if err := tx.Where("1 = 1").Unscoped().Delete(portList).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
