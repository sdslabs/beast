package database

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Notification struct {
	gorm.Model

	Title       string `gorm:not null;unique`
	Description string `gorm:not null`
}

func AddNotification(notification *Notification) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while adding notification", tx.Error)
	}

	if err := tx.FirstOrCreate(notification, *notification).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func DeleteNotification(notification *Notification) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while deleting notification : %s", tx.Error)
	}

	if err := tx.Unscoped().Delete(notification).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func QueryNotificationEntries(key string, value string) ([]Notification, error) {
	queryKey := fmt.Sprintf("%s = ?", key)

	var notifications []Notification

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where(queryKey, value).Find(&notifications)
	if tx.RecordNotFound() {
		return nil, nil
	}

	if tx.Error != nil {
		return notifications, tx.Error
	}

	return notifications, nil
}

func QueryFirstNotificationEntry(key string, value string) (Notification, error) {
	notifications, err := QueryNotificationEntries(key, value)
	if err != nil {
		return Notification{}, err
	}

	if len(notifications) == 0 {
		return Notification{}, nil
	}

	return notifications[0], nil
}

func QueryAllNotification() ([]Notification, error) {
	var notifications []Notification

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Find(&notifications)
	if tx.RecordNotFound() {
		return nil, nil
	}

	return notifications, tx.Error
}

func UpdateNotification(notify *Notification, m map[string]interface{}) error {

	DBMux.Lock()
	defer DBMux.Unlock()

	return Db.Model(notify).Update(m).Error
}
