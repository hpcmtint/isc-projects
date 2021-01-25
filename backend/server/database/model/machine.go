package dbmodel

import (
	"errors"
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	pkgerrors "github.com/pkg/errors"
)

// Part of machine table in database that describes state of machine. In DB it is stored as JSONB.
type MachineState struct {
	AgentVersion         string
	Cpus                 int64
	CpusLoad             string
	Memory               int64
	Hostname             string
	Uptime               int64
	UsedMemory           int64
	Os                   string
	Platform             string
	PlatformFamily       string
	PlatformVersion      string
	KernelVersion        string
	KernelArch           string
	VirtualizationSystem string
	VirtualizationRole   string
	HostID               string
}

// Represents a machine held in machine table in the database.
type Machine struct {
	ID              int64
	CreatedAt       time.Time
	Address         string
	AgentPort       int64
	LastVisitedAt   time.Time
	Error           string
	State           MachineState
	Apps            []*App
	AgentToken      string
	CertFingerprint [32]byte
	Authorized      bool `pg:",use_zero"`
}

// Add new machine to database.
func AddMachine(db *pg.DB, machine *Machine) error {
	err := db.Insert(machine)
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with inserting machine %+v", machine)
	}
	return err
}

// Update a machine in database.
func UpdateMachine(db *pg.DB, machine *Machine) error {
	err := db.Update(machine)
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with updating machine %+v", machine)
	}
	return err
}

// Get a machine by address and agent port.
func GetMachineByAddressAndAgentPort(db *pg.DB, address string, agentPort int64) (*Machine, error) {
	machine := Machine{}
	q := db.Model(&machine)
	q = q.Where("address = ?", address)
	q = q.Where("agent_port = ?", agentPort)
	q = q.Relation("Apps.AccessPoints")
	err := q.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem with getting machine %s:%d", address, agentPort)
	}
	return &machine, nil
}

// Get a machine by its ID.
func GetMachineByID(db *pg.DB, id int64) (*Machine, error) {
	machine := Machine{}
	q := db.Model(&machine).Where("machine.id = ?", id)
	q = q.Relation("Apps.Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Apps.Daemons.Bind9Daemon")
	q = q.Relation("Apps.AccessPoints")
	err := q.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem with getting machine %v", id)
	}

	return &machine, nil
}

// Refresh machine from database.
func RefreshMachineFromDB(db *pg.DB, machine *Machine) error {
	machine.Apps = []*App{}
	q := db.Model(machine).Where("id = ?", machine.ID)
	q = q.Relation("Apps.AccessPoints")
	err := q.Select()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem with getting machine %v", machine.ID)
	}

	return nil
}

// Fetches a collection of machines from the database.
//
// The offset and limit specify the beginning of the page and the
// maximum size of the page. Limit has to be greater then 0, otherwise
// error is returned.
//
// filterText allows filtering machines by provided text. It is check
// against several different fields in Machine record. If not provided
// then no filtering by text happens.
//
// authorized allows filtering machines by authorized field in Machine
// record. It can be true or false then authorized or unauthorized
// machines are returned. If it is nil then no filtering by authorized
// happens (ie. all machines are returned).
//
// sortField allows indicating sort column in database and sortDir
// allows selection the order of sorting. If sortField is empty then
// id is used for sorting.  in SortDirAny is used then ASC order is
// used.
func GetMachinesByPage(db *pg.DB, offset int64, limit int64, filterText *string, authorized *bool, sortField string, sortDir SortDirEnum) ([]Machine, int64, error) {
	if limit == 0 {
		return nil, 0, pkgerrors.New("limit should be greater than 0")
	}
	var machines []Machine

	// prepare query
	q := db.Model(&machines)
	q = q.Relation("Apps.AccessPoints")
	q = q.Relation("Apps.Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Apps.Daemons.Bind9Daemon")

	// prepare filtering by text
	if filterText != nil {
		text := "%" + *filterText + "%"
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			qq = qq.WhereOr("address ILIKE ?", text).
				WhereOr("state->>'AgentVersion' ILIKE ?", text).
				WhereOr("state->>'Hostname' ILIKE ?", text).
				WhereOr("state->>'Os' ILIKE ?", text).
				WhereOr("state->>'Platform' ILIKE ?", text).
				WhereOr("state->>'PlatformFamily' ILIKE ?", text).
				WhereOr("state->>'PlatformVersion' ILIKE ?", text).
				WhereOr("state->>'KernelVersion' ILIKE ?", text).
				WhereOr("state->>'KernelArch' ILIKE ?", text).
				WhereOr("state->>'VirtualizationSystem' ILIKE ?", text).
				WhereOr("state->>'VirtualizationRole' ILIKE ?", text).
				WhereOr("state->>'HostID' ILIKE ?", text)
			return qq, nil
		})
	}

	// prepare filtering by authorized
	if authorized != nil {
		if *authorized {
			q = q.Where("authorized = ?", true)
		} else {
			q = q.Where("authorized != ?", true)
		}
	}

	// prepare sorting expression, offset and limit
	ordExpr := prepareOrderExpr("machine", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return []Machine{}, 0, nil
		}
		return nil, 0, pkgerrors.Wrapf(err, "problem with getting machines")
	}

	return machines, int64(total), nil
}

// Get all machines from database. It can be filtered by authorized field.
func GetAllMachines(db *pg.DB, authorized *bool) ([]Machine, error) {
	var machines []Machine

	// prepare query
	q := db.Model(&machines)
	if authorized != nil {
		if *authorized {
			q = q.Where("authorized = ?", true)
		} else {
			q = q.Where("authorized != ?", true)
		}
	}
	q = q.Relation("Apps.AccessPoints")
	q = q.Relation("Apps.Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Apps.Daemons.Bind9Daemon")

	err := q.Select()
	if err != nil && errors.Is(err, pg.ErrNoRows) {
		return nil, pkgerrors.Wrapf(err, "problem with getting machines")
	}

	return machines, nil
}

// Delete a machine from database.
func DeleteMachine(db *pg.DB, machine *Machine) error {
	err := db.Delete(machine)
	if err != nil {
		return pkgerrors.Wrapf(err, "problem with deleting machine %v", machine.ID)
	}
	return nil
}
