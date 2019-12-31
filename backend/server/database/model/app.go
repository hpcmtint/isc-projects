package dbmodel

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"github.com/pkg/errors"

	//log "github.com/sirupsen/logrus"

	storkutil "isc.org/stork/util"
)

type Bind9Daemon struct {
	Pid     int32
	Name    string
	Active  bool
	Version string
}

type KeaDaemon struct {
	Pid             int32
	Name            string
	Active          bool
	Version         string
	ExtendedVersion string
}

const KeaAppType = "kea"

type AppKea struct {
	ExtendedVersion string
	Daemons         []KeaDaemon
}

const Bind9AppType = "bind9"

type AppBind9 struct {
	Daemon Bind9Daemon
}

// Part of app table in database that describes metadata of app. In DB it is stored as JSONB.
type AppMeta struct {
	Version string
}

// Represents a app held in app table in the database.
type App struct {
	ID          int64
	Created     time.Time
	Deleted     time.Time
	MachineID   int64
	Machine     Machine
	Type        string
	CtrlAddress string
	CtrlPort    int64
	Active      bool
	Meta        AppMeta
	Details     interface{} // here we have either AppKea or AppBind9
}

// This is a hook to go-pg that is called just after reading rows from database.
// It reconverts app details from json string maps to particular Go structure.
func (app *App) AfterScan(ctx context.Context) error {
	if app.Details == nil {
		return nil
	}

	bytes, err := json.Marshal(app.Details)
	if err != nil {
		return errors.Wrapf(err, "problem with marshaling %s app details: %v ", app.Type, app.Details)
	}

	switch app.Type {
	case KeaAppType:
		var keaDetails AppKea
		err = json.Unmarshal(bytes, &keaDetails)
		if err != nil {
			return errors.Wrapf(err, "problem with unmarshaling kea app details")
		}
		app.Details = keaDetails

	case Bind9AppType:
		var bind9Details AppBind9
		err = json.Unmarshal(bytes, &bind9Details)
		if err != nil {
			return errors.Wrapf(err, "problem with unmarshaling bind9 app details")
		}
		app.Details = bind9Details
	}
	return nil
}

func AddApp(db *pg.DB, app *App) error {
	err := db.Insert(app)
	if err != nil {
		return errors.Wrapf(err, "problem with inserting app %v", app)
	}
	return nil
}

func GetAppByID(db *pg.DB, id int64) (*App, error) {
	app := App{}
	q := db.Model(&app).Where("app.id = ?", id)
	q = q.Relation("Machine")
	err := q.Select()
	if err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem with getting app %v", id)
	}
	return &app, nil
}

func GetAppsByMachine(db *pg.DB, machineID int64) ([]App, error) {
	var apps []App

	q := db.Model(&apps)
	q = q.Where("machine_id = ?", machineID)
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem with getting apps")
	}
	return apps, nil
}

// Fetches a collection of apps from the database. The offset and limit specify the
// beginning of the page and the maximum size of the page. Limit has to be greater
// then 0, otherwise error is returned.
func GetAppsByPage(db *pg.DB, offset int64, limit int64, text string, appType string) ([]App, int64, error) {
	if limit == 0 {
		return nil, 0, errors.New("limit should be greater than 0")
	}
	var apps []App

	// prepare query
	q := db.Model(&apps)
	q = q.Where("app.deleted is NULL")
	q = q.Relation("Machine")
	if appType != "" {
		q = q.Where("type = ?", appType)
	}
	if text != "" {
		text = "%" + text + "%"
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			qq = qq.WhereOr("meta->>'Version' ILIKE ?", text)
			return qq, nil
		})
	}

	// and then, first get total count
	total, err := q.Clone().Count()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting apps total")
	}

	// then retrieve given page of rows
	q = q.Order("id ASC").Offset(int(offset)).Limit(int(limit))
	err = q.Select()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting apps")
	}
	return apps, int64(total), nil
}

func DeleteApp(db *pg.DB, app *App) error {
	app.Deleted = storkutil.UTCNow()
	err := db.Update(app)
	if err != nil {
		return errors.Wrapf(err, "problem with deleting app %v", app.ID)
	}
	return nil
}

func GetAllApps(db *pg.DB) ([]App, error) {
	var apps []App

	// prepare query
	q := db.Model(&apps)
	q = q.Where("app.deleted is NULL")

	// retrieve apps from db
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem with getting apps")
	}
	return apps, nil
}

// Returns a list of names of active DHCP deamons. This is useful for
// creating commands to be send to active DHCP servers.
func (app *App) GetActiveDHCPDeamonNames() (deamons []string) {
	if kea, ok := app.Details.(AppKea); ok {
		for _, d := range kea.Daemons {
			if d.Active && (d.Name == "dhcp4" || d.Name == "dhcp6") {
				deamons = append(deamons, d.Name)
			}
		}
	}
	return deamons
}
