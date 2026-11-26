package database

import (
	"fmt"

	"gorm.io/gorm"
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

func UpdatePorts(challenge *Challenge) error {
	DBMux.Lock()
	defer DBMux.Unlock()
	return Db.Unscoped().Where("challenge_id = ?", challenge.ID).Delete(&Port{}).Error
}

func DeleteRelatedPorts(portList []Port) error {
	if len(portList) == 0 {
		return nil
	}
	DBMux.Lock()
	defer DBMux.Unlock()

	ids := make([]uint, 0, len(portList))
	for _, port := range portList {
		ids = append(ids, port.ID)
	}
	return Db.Unscoped().Where("id IN ?", ids).Delete(&Port{}).Error
}
