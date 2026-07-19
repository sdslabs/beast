package database

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type Port struct {
	gorm.Model

	ChallengeID uint   `gorm:"not null"`
	Server      string `gorm:"not null;default:'';uniqueIndex:idx_ports_server_port"`
	PortNo      uint32 `gorm:"not null;uniqueIndex:idx_ports_server_port"`
}

// Create an entry for the port in the Port table
// It returns an error if anything wrong happen during the
// transaction. If the entry already exists then it does not
// do anything and returns.
func PortEntryGetOrCreate(port *Port) (Port, error) {
	if port == nil || port.ChallengeID == 0 {
		return Port{}, errors.New("persisted challenge port is required")
	}
	DBMux.Lock()
	defer DBMux.Unlock()

	err := Db.Transaction(func(tx *gorm.DB) error {
		if port.Server == "" {
			var challenge Challenge
			if err := tx.Select("server_deployed").First(&challenge, port.ChallengeID).Error; err != nil {
				return fmt.Errorf("resolve challenge server: %w", err)
			}
			port.Server = challenge.ServerDeployed
		}
		return tx.FirstOrCreate(port, Port{Server: port.Server, PortNo: port.PortNo}).Error
	})
	return *port, err
}

func GetAllocatedPorts(challenge Challenge) ([]Port, error) {
	var ports []Port

	DBMux.RLock()
	defer DBMux.RUnlock()

	if err := Db.Model(&challenge).Association("Ports").Find(&ports); err != nil {
		return nil, fmt.Errorf("error while searching port for challenge : %s", err)
	}

	return ports, nil
}

func UpdatePorts(challenge *Challenge) error {
	if challenge == nil || challenge.ID == 0 {
		return errors.New("persisted challenge is required")
	}

	DBMux.Lock()
	defer DBMux.Unlock()

	return Db.Unscoped().Where("challenge_id = ?", challenge.ID).Delete(&Port{}).Error
}

func DeleteRelatedPorts(portList []Port) error {
	if len(portList) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(portList))
	for _, port := range portList {
		if port.ID == 0 {
			return errors.New("persisted port is required")
		}
		ids = append(ids, port.ID)
	}

	DBMux.Lock()
	defer DBMux.Unlock()

	return Db.Unscoped().Where("id IN ?", ids).Delete(&Port{}).Error
}

func MigratePortUniqueness() error {
	if Db == nil {
		return errors.New("database is not initialized")
	}
	return Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
UPDATE ports
SET server = challenges.server_deployed
FROM challenges
WHERE ports.challenge_id = challenges.id AND ports.server = ''`).Error; err != nil {
			return fmt.Errorf("backfill port servers: %w", err)
		}
		statements := []string{
			`ALTER TABLE ports DROP CONSTRAINT IF EXISTS uni_ports_port_no`,
			`ALTER TABLE ports DROP CONSTRAINT IF EXISTS ports_port_no_key`,
			`DROP INDEX IF EXISTS idx_ports_port_no`,
			`DROP INDEX IF EXISTS uix_ports_port_no`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_ports_server_port ON ports (server, port_no)`,
		}
		for _, statement := range statements {
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("migrate port uniqueness: %w", err)
			}
		}
		return nil
	})
}
