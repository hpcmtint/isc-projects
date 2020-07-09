package dbmodel

import (
	"strconv"

	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"
)

// This module provides global settings that can be used anywhere in the code.
// All settings with their defaul values are defined in defaultSettings table,
// in InitializeSettings function. There is a few functions for getting and
// setting these settings. Generally setters are used in API function
// so users can set these settings. Getters are used around the code
// where given setting is needed.

// TODO: add caching to avoid trips to database; candidate caching libs:
// https://allegro.tech/2016/03/writing-fast-cache-service-in-go.html

const SettingValTypeInt = 1
const SettingValTypeBool = 2
const SettingValTypeStr = 3
const SettingValTypePasswd = 4

// Represents a setting held in setting table in the database.
type Setting struct {
	Name    string `pg:",pk"`
	ValType int64
	Value   string `pg:",use_zero"`
}

// Initialize settings in db. If new setting needs to be added then add it to defaultSettings list
// and it will be automatically added to db here in this function.
func InitializeSettings(db *pg.DB) error {
	// list of all stork settings with default values
	defaultSettings := []Setting{
		{
			Name:    "bind9_stats_puller_interval", // in seconds
			ValType: SettingValTypeInt,
			Value:   "60",
		},
		{
			Name:    "kea_stats_puller_interval", // in seconds
			ValType: SettingValTypeInt,
			Value:   "60",
		},
		{
			Name:    "kea_hosts_puller_interval", // in seconds
			ValType: SettingValTypeInt,
			Value:   "60",
		},
		{
			Name:    "kea_status_puller_interval", // in seconds
			ValType: SettingValTypeInt,
			Value:   "30",
		},
		{
			Name:    "apps_state_puller_interval", // in seconds
			ValType: SettingValTypeInt,
			Value:   "30",
		},
		{
			Name:    "grafana_url",
			ValType: SettingValTypeStr,
			Value:   "",
		},
		{
			Name:    "prometheus_url",
			ValType: SettingValTypeStr,
			Value:   "",
		},
	}

	// Check if there are new settings vs existing ones. Add new ones to DB.
	_, err := db.Model(&defaultSettings).OnConflict("DO NOTHING").Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem with inserting default settings")
	}
	return err
}

// Get setting record from db based on its name.
func GetSetting(db *pg.DB, name string) (*Setting, error) {
	setting := Setting{}
	q := db.Model(&setting).Where("setting.name = ?", name)
	err := q.Select()
	if err == pg.ErrNoRows {
		return nil, errors.Wrapf(err, "setting %s is missing", name)
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem with getting setting %s", name)
	}
	return &setting, nil
}

// Get setting by name and check if its type matches to expected one.
func getAndCheckSetting(db *pg.DB, name string, expValType int64) (*Setting, error) {
	s, err := GetSetting(db, name)
	if err != nil {
		return nil, err
	}
	if s.ValType != expValType {
		return nil, errors.Errorf("not matching setting type of %s (%d vs %d expected)", name, s.ValType, expValType)
	}
	return s, nil
}

// Get int value of given setting by name.
func GetSettingInt(db *pg.DB, name string) (int64, error) {
	s, err := getAndCheckSetting(db, name, SettingValTypeInt)
	if err != nil {
		return 0, err
	}
	val, err := strconv.ParseInt(s.Value, 10, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

// Get bool value of given setting by name.
func GetSettingBool(db *pg.DB, name string) (bool, error) {
	s, err := getAndCheckSetting(db, name, SettingValTypeBool)
	if err != nil {
		return false, err
	}
	val, err := strconv.ParseBool(s.Value)
	if err != nil {
		return false, err
	}
	return val, nil
}

// Get string value of given setting by name.
func GetSettingStr(db *pg.DB, name string) (string, error) {
	s, err := getAndCheckSetting(db, name, SettingValTypeStr)
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

// Get password value of given setting by name.
func GetSettingPasswd(db *pg.DB, name string) (string, error) {
	s, err := getAndCheckSetting(db, name, SettingValTypePasswd)
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

// Get all settings.
func GetAllSettings(db *pg.DB) (map[string]interface{}, error) {
	settings := []*Setting{}
	q := db.Model(&settings)
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem with getting all settings")
	}

	settingsMap := make(map[string]interface{})

	for _, s := range settings {
		switch s.ValType {
		case SettingValTypeInt:
			val, err := strconv.ParseInt(s.Value, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "problem with getting setting value of %s", s.Name)
			}
			settingsMap[s.Name] = val
		case SettingValTypeBool:
			val, err := strconv.ParseBool(s.Value)
			if err != nil {
				return nil, errors.Wrapf(err, "problem with getting setting value of %s", s.Name)
			}
			settingsMap[s.Name] = val
		case SettingValTypeStr:
			settingsMap[s.Name] = s.Value
		case SettingValTypePasswd:
			// do not return passwords to users
		}
	}

	return settingsMap, nil
}

// Set int value of given setting by name.
func SetSettingInt(db *pg.DB, name string, value int64) error {
	s, err := getAndCheckSetting(db, name, SettingValTypeInt)
	if err != nil {
		return err
	}
	s.Value = strconv.FormatInt(value, 10)
	err = db.Update(s)
	if err != nil {
		return errors.Wrapf(err, "problem with updating setting %s", name)
	}
	return nil
}

// Set bool value of given setting by name.
func SetSettingBool(db *pg.DB, name string, value bool) error {
	s, err := getAndCheckSetting(db, name, SettingValTypeBool)
	if err != nil {
		return err
	}
	s.Value = strconv.FormatBool(value)
	err = db.Update(s)
	if err != nil {
		return errors.Wrapf(err, "problem with updating setting %s", name)
	}
	return nil
}

// Set string value of given setting by name.
func SetSettingStr(db *pg.DB, name string, value string) error {
	s, err := getAndCheckSetting(db, name, SettingValTypeStr)
	if err != nil {
		return err
	}
	s.Value = value
	err = db.Update(s)
	if err != nil {
		return errors.Wrapf(err, "problem with updating setting %s", name)
	}
	return nil
}

// Set password value of given setting by name.
func SetSettingPasswd(db *pg.DB, name string, value string) error {
	s, err := getAndCheckSetting(db, name, SettingValTypePasswd)
	if err != nil {
		return err
	}
	s.Value = value
	err = db.Update(s)
	if err != nil {
		return errors.Wrapf(err, "problem with updating setting %s", name)
	}
	return nil
}
