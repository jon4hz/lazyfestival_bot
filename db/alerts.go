package db

import "time"

type Alert struct {
	ID         int64
	Band       string
	Time       time.Time
	Min        int32
	TelegramId int64
}

func (db *Database) GetAlerts() ([]Alert, error) {
	var alerts []Alert
	err := db.sql.Select(&alerts, "SELECT * FROM alerts")
	return alerts, err
}

func (db *Database) GetAlertsByTgIDAndBand(tgID int64, band string) ([]Alert, error) {
	var alerts []Alert
	err := db.sql.Select(&alerts, "SELECT * FROM alerts WHERE telegramid = ? AND band = ?", tgID, band)
	return alerts, err
}

func (db *Database) GetReadyAlerts() ([]Alert, error) {
	var alerts []Alert
	err := db.sql.Select(&alerts, "SELECT * FROM alerts WHERE datetime('now') >= datetime(time, '-' || min || ' minutes');")
	return alerts, err
}

func (db *Database) CreateAlert(alert Alert) error {
	_, err := db.sql.Exec("INSERT INTO alerts (band, time, min, telegramid) VALUES (?, datetime(?), ?, ?)", alert.Band, alert.Time.UTC().Format("2006-01-02 15:04:05.000"), alert.Min, alert.TelegramId)
	return err
}

func (db *Database) DeleteAlert(alert Alert) error {
	_, err := db.sql.Exec("DELETE FROM alerts WHERE band = ? AND time = datetime(?) AND min = ? AND telegramid = ?", alert.Band, alert.Time.UTC().Format("2006-01-02 15:04:05.000"), alert.Min, alert.TelegramId)
	return err
}
