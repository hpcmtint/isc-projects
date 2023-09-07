package dbmodel

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	pkgerrors "github.com/pkg/errors"
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// Source of the host information, i.e. configuration file or API (host_cmds).
type HostDataSource string

// Valid host data sources.
const (
	HostDataSourceConfig HostDataSource = "config"
	HostDataSourceAPI    HostDataSource = "api"
)

// Converts HostDataSource to string.
func (s HostDataSource) String() string {
	return string(s)
}

// Creates HostDataSource instance from string. It returns an error
// when specified string is neither "api" nor "config".
func ParseHostDataSource(s string) (hds HostDataSource, err error) {
	hds = HostDataSource(s)
	if hds != HostDataSourceConfig && hds != HostDataSourceAPI {
		err = pkgerrors.Errorf("unsupported host data source '%s'", s)
	}
	return
}

// This structure reflects a row in the host_identifier table. It includes
// a value and type of the identifier used to match the client with a host. The
// following types are available: hw-address, duid, circuit-id, client-id
// and flex-id (same as those available in Kea).
type HostIdentifier struct {
	ID     int64
	Type   string
	Value  []byte
	HostID int64
}

// This structure reflects a row in the ip_reservation table. It represents
// a single IP address or prefix reservation associated with a selected host.
type IPReservation struct {
	ID      int64
	Address string
	HostID  int64
}

// Checks if reservation is a delegated prefix.
func (r *IPReservation) IsPrefix() bool {
	ip := storkutil.ParseIP(r.Address)
	if ip == nil {
		return false
	}
	return ip.Prefix
}

// This structure reflects a row in the host table. The host may be associated
// with zero, one or multiple IP reservations. It may also be associated with
// one or more identifiers which are used for matching DHCP clients with the
// host.
type Host struct {
	ID        int64 `pg:",pk"`
	CreatedAt time.Time
	SubnetID  int64
	Subnet    *Subnet `pg:"rel:has-one"`

	Hostname string

	HostIdentifiers []HostIdentifier `pg:"rel:has-many"`
	IPReservations  []IPReservation  `pg:"rel:has-many"`

	LocalHosts []LocalHost `pg:"rel:has-many"`
}

// This structure links a host entry stored in the database with a daemon from
// which it has been retrieved. It provides M:N relationship between hosts
// and daemons.
type LocalHost struct {
	HostID     int64   `pg:",pk"`
	DaemonID   int64   `pg:",pk"`
	Daemon     *Daemon `pg:"rel:has-one"`
	Host       *Host   `pg:"rel:has-one"`
	DataSource HostDataSource

	ClientClasses     []string `pg:",array"`
	NextServer        string
	ServerHostname    string
	BootFileName      string
	DHCPOptionSet     []DHCPOption
	DHCPOptionSetHash string
}

// Associates a host with DHCP with host identifiers.
func addHostIdentifiers(tx *pg.Tx, host *Host) error {
	for i, id := range host.HostIdentifiers {
		identifier := id
		identifier.HostID = host.ID
		_, err := tx.Model(&identifier).
			OnConflict("(type, host_id) DO UPDATE").
			Set("value = EXCLUDED.value").
			Insert()
		if err != nil {
			err = pkgerrors.Wrapf(err, "problem adding host identifier with type %s for host with ID %d",
				identifier.Type, host.ID)
			return err
		}
		host.HostIdentifiers[i] = identifier
	}
	return nil
}

// Associates a host with IP reservations.
func addIPReservations(tx *pg.Tx, host *Host) error {
	for i, r := range host.IPReservations {
		reservation := r
		reservation.HostID = host.ID
		_, err := tx.Model(&reservation).
			OnConflict("DO NOTHING").
			Insert()
		if err != nil {
			err = pkgerrors.Wrapf(err, "problem adding IP reservation %s for host with ID %d",
				reservation.Address, host.ID)
			return err
		}
		host.IPReservations[i] = reservation
	}
	return nil
}

// Adds new host, its reservations and identifiers into the database in
// a transaction.
func addHost(tx *pg.Tx, host *Host) error {
	// Add the host and fetch its id.
	_, err := tx.Model(host).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem adding new host")
		return err
	}
	// Associate the host with the given id with its identifiers.
	err = addHostIdentifiers(tx, host)
	if err != nil {
		return err
	}
	// Associate the host with the given id with its reservations.
	err = addIPReservations(tx, host)
	if err != nil {
		return err
	}
	return nil
}

// Adds new host, its reservations and identifiers into the database.
// It begins a new transaction when dbi has a *pg.DB type or uses an
// existing transaction when dbi has a *pg.Tx type.
func AddHost(dbi dbops.DBI, host *Host) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addHost(tx, host)
		})
	}
	return addHost(dbi.(*pg.Tx), host)
}

// Updates a host, its reservations and identifiers in the database
// in a transaction.
func updateHost(tx *pg.Tx, host *Host) error {
	// Collect updated identifiers.
	hostIDTypes := []string{}
	for _, hostID := range host.HostIdentifiers {
		hostIDTypes = append(hostIDTypes, hostID.Type)
	}
	q := tx.Model((*HostIdentifier)(nil)).
		Where("host_identifier.host_id = ?", host.ID)
	// If the new reservation has any host identifiers exclude them from
	// the deleted ones. Otherwise, delete all reservations belonging to
	// the old host version.
	if len(hostIDTypes) > 0 {
		q = q.Where("host_identifier.type NOT IN (?)", pg.In(hostIDTypes))
	}
	_, err := q.Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting host identifiers for host %d", host.ID)
		return err
	}
	// Add or update host identifiers.
	err = addHostIdentifiers(tx, host)
	if err != nil {
		return pkgerrors.WithMessagef(err, "problem updating host with ID %d", host.ID)
	}

	// Collect updated identifiers.
	ipAddresses := []string{}
	for _, resrv := range host.IPReservations {
		ipAddresses = append(ipAddresses, resrv.Address)
	}
	q = tx.Model((*IPReservation)(nil)).
		Where("ip_reservation.host_id = ?", host.ID)
	// If the new reservation has some reserved IP addresses exclude them
	// from the deleted ones. Otherwise, delete all IP addresses belonging
	// to the old host version.
	if len(ipAddresses) > 0 {
		q = q.Where("ip_reservation.address NOT IN (?)", pg.In(ipAddresses))
	}
	_, err = q.Delete()

	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting IP reservations for host %d", host.ID)
		return err
	}
	// Add or update host reservations.
	err = addIPReservations(tx, host)
	if err != nil {
		return pkgerrors.WithMessagef(err, "problem updating host with ID %d", host.ID)
	}

	// Update the host information.
	result, err := tx.Model(host).WherePK().ExcludeColumn("created_at").Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating host with ID %d", host.ID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "host with ID %d does not exist", host.ID)
	}
	return err
}

// Updates a host, its reservations and identifiers in the database.
func UpdateHost(dbi pg.DBI, host *Host) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return updateHost(tx, host)
		})
	}
	return updateHost(dbi.(*pg.Tx), host)
}

// Attempts to update a host and its local hosts with in an existing transaction.
func updateHostWithLocalHosts(tx *pg.Tx, host *Host) error {
	err := updateHost(tx, host)
	if err != nil {
		return err
	}
	// Delete current associations of the host with the daemons.
	_, err = tx.Model((*LocalHost)(nil)).
		Where("host_id = ?", host.ID).
		Delete()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem deleting daemons from host %d", host.ID)
	}
	// Add new associations.
	err = AddHostLocalHosts(tx, host)
	return err
}

// Attempts to update a host and its local hosts within a transaction. If the dbi
// does not point to a transaction, a new transaction is started.
func UpdateHostWithLocalHosts(dbi dbops.DBI, host *Host) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return updateHostWithLocalHosts(tx, host)
		})
	}
	return updateHostWithLocalHosts(dbi.(*pg.Tx), host)
}

// Fetch the host by ID.
func GetHost(dbi dbops.DBI, hostID int64) (*Host, error) {
	host := &Host{}
	err := dbi.Model(host).
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("Subnet.LocalSubnets").
		Relation("LocalHosts.Daemon.App.Machine").
		Relation("LocalHosts.Daemon.App.AccessPoints").
		Where("host.id = ?", hostID).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting host with ID %d", hostID)
		return nil, err
	}
	return host, err
}

// Fetch all hosts having address reservations belonging to a specific family
// or all reservations regardless of the family.
func GetAllHosts(dbi dbops.DBI, family int) ([]Host, error) {
	hosts := []Host{}
	q := dbi.Model(&hosts).DistinctOn("id")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Join("INNER JOIN ip_reservation AS r ON r.host_id = host.id")
		q = q.Where("family(r.address) = ?", family)
	}

	// Include host identifiers and IP reservations.
	q = q.
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("LocalHosts").
		OrderExpr("id ASC")

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting all hosts for family %d", family)
		return nil, err
	}
	return hosts, err
}

// Fetches a collection of hosts by subnet ID. This function may be sometimes
// used within a transaction. In particular, when we're synchronizing hosts
// fetched from the Kea hosts backend in multiple chunks.`.
func GetHostsBySubnetID(dbi dbops.DBI, subnetID int64) ([]Host, error) {
	hosts := []Host{}
	q := dbi.Model(&hosts).
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("LocalHosts").
		OrderExpr("id ASC")

	// Subnet ID is never zero, it may be NULL. The reason for it is that we
	// have a foreign key that requires subnet to exist for non NULL value.
	// This constraint allows for NULL subnet_id though. Therefore, searching
	// for a host with subnet_id of zero is really searching for a host with
	// the NULL value.
	if subnetID == 0 {
		q = q.Where("host.subnet_id IS NULL")
	} else {
		q = q.Where("host.subnet_id = ?", subnetID)
	}

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting hosts by subnet ID %d", subnetID)
		return nil, err
	}
	return hosts, err
}

// Fetches a collection of hosts by daemon ID and optionally filters by a
// data source.
func GetHostsByDaemonID(dbi dbops.DBI, daemonID int64, dataSource HostDataSource) ([]Host, int64, error) {
	hosts := []Host{}
	q := dbi.Model(&hosts).
		Join("INNER JOIN local_host AS lh ON host.id = lh.host_id").
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("LocalHosts").
		Relation("Subnet.LocalSubnets").
		OrderExpr("id ASC").
		Where("lh.daemon_id = ?", daemonID)

	// Optionally filter by a data source.
	if len(dataSource) > 0 {
		q = q.Where("lh.data_source = ?", dataSource)
	}

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, 0, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting hosts for daemon %d", daemonID)
	}
	return hosts, int64(total), err
}

// Container for values filtering hosts fetched by page.
//
// The AppID, if different than 0, is used to fetch hosts whose local hosts belong to
// the indicated app.
//
// The optional SubnetID is used to fetch hosts belonging to the particular IPv4 or
// IPv6 subnet. If this value is set to nil all subnets are returned. The value of 0
// indicates that only global hosts are to be returned.
//
// The LocalSubnetID filters hosts by the subnet ID specified in the Kea configuration.
//
// Filtering text allows for searching hosts by reserved IP addresses, host identifiers
// (using hexadecimal digits or a textual format) and hostnames. It is allowed to specify
// colons while searching for hosts by host identifiers. If Global flag is true then only
// hosts from the global scope are returned (i.e. not assigned to any subnet), if false
// then only hosts from subnets are returned.
type HostsByPageFilters struct {
	AppID         *int64
	SubnetID      *int64
	LocalSubnetID *int64
	FilterText    *string
	Global        *bool
}

// Fetches a collection of hosts from the database.
//
// The offset and limit specify the beginning of the page and the maximum size of the
// page.
//
// sortField allows indicating sort column in database and sortDir allows selection the
// order of sorting. If sortField is empty then id is used for sorting. If SortDirAny is
// used then ASC order is used.
func GetHostsByPage(dbi dbops.DBI, offset, limit int64, filters HostsByPageFilters, sortField string, sortDir SortDirEnum) ([]Host, int64, error) {
	hosts := []Host{}
	q := dbi.Model(&hosts)

	// prepare distinct on expression to include sort field, otherwise distinct on will fail
	distinctOnFields := "host.id"
	if sortField != "" && sortField != "id" && sortField != "host.id" {
		distinctOnFields = sortField + ", " + distinctOnFields
	}
	q = q.DistinctOn(distinctOnFields)

	// When filtering by appID we also need the local_host table as it holds the
	// application identifier.
	if filters.AppID != nil && *filters.AppID != 0 {
		q = q.Join("JOIN local_host").JoinOn("host.id = local_host.host_id")
		q = q.Join("JOIN daemon").JoinOn("local_host.daemon_id = daemon.id")
		q = q.Where("daemon.app_id = ?", *filters.AppID)
	}

	// filter by subnet id
	if filters.SubnetID != nil && *filters.SubnetID != 0 {
		// Get hosts for matching subnet id.
		q = q.Where("host.subnet_id = ?", *filters.SubnetID)
	}

	// filter by local subnet id
	if filters.LocalSubnetID != nil {
		q = q.Join("JOIN local_subnet").JoinOn("local_subnet.subnet_id = host.subnet_id")
		q = q.Where("local_subnet.local_subnet_id = ?", *filters.LocalSubnetID)
	}

	// filter global or non-global hosts
	if (filters.Global != nil && *filters.Global) || (filters.SubnetID != nil && *filters.SubnetID == 0) {
		q = q.WhereOr("host.subnet_id IS NULL")
	}
	if filters.Global != nil && !*filters.Global {
		q = q.WhereOr("host.subnet_id IS NOT NULL")
	}

	// filter by text
	if filters.FilterText != nil && len(*filters.FilterText) > 0 {
		// It is possible that the user is typing a search text with colons
		// for host identifiers. We need to remove them because they are
		// not present in the database.
		colonlessFilterText := strings.ReplaceAll(*filters.FilterText, ":", "")
		q = q.Join("JOIN ip_reservation AS r").JoinOn("r.host_id = host.id")
		q = q.Join("JOIN host_identifier AS i").JoinOn("i.host_id = host.id")
		q = q.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q = q.WhereOr("text(r.address) ILIKE ?", "%"+*filters.FilterText+"%").
				WhereOr("i.type::text ILIKE ?", "%"+*filters.FilterText+"%").
				WhereOr("encode(i.value, 'hex') ILIKE ?", "%"+colonlessFilterText+"%").
				WhereOr("encode(i.value, 'escape') ILIKE ?", "%"+*filters.FilterText+"%").
				WhereOr("host.hostname ILIKE ?", "%"+*filters.FilterText+"%")
			return q, nil
		})
	}

	q = q.
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("LocalHosts").
		Relation("LocalHosts.Daemon.App").
		Relation("LocalHosts.Daemon.App.Machine").
		Relation("LocalHosts.Daemon.App.AccessPoints")

	// Only join the subnet if querying all hosts or hosts belonging to a
	// given subnet.
	if filters.SubnetID == nil || *filters.SubnetID > 0 {
		q = q.Relation("Subnet")
	}

	// prepare sorting expression, offset and limit
	ordExpr := prepareOrderExpr("host", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, 0, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting hosts by page")
	}
	return hosts, int64(total), err
}

// Delete host, host identifiers and reservations by id.
func DeleteHost(dbi dbops.DBI, hostID int64) error {
	host := &Host{
		ID: hostID,
	}
	result, err := dbi.Model(host).WherePK().Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting the host with ID %d", hostID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "host with ID %d does not exist", hostID)
	}
	return err
}

// Associates a daemon with the host having a specified ID.
func AddDaemonToHost(dbi dbops.DBI, host *Host, daemonID int64, source HostDataSource) error {
	hostCopy := Host{
		ID: host.ID,
		LocalHosts: []LocalHost{
			{
				DaemonID:   daemonID,
				DataSource: source,
			},
		},
	}
	return AddHostLocalHosts(dbi, &hostCopy)
}

// Iterates over the LocalHost instances of a Host and inserts them or
// updates in the database.
func AddHostLocalHosts(dbi dbops.DBI, host *Host) error {
	for i := range host.LocalHosts {
		host.LocalHosts[i].HostID = host.ID
		q := dbi.Model(&host.LocalHosts[i]).
			OnConflict("(daemon_id, host_id) DO UPDATE").
			Set("data_source = EXCLUDED.data_source").
			Set("client_classes = EXCLUDED.client_classes").
			Set("dhcp_option_set = EXCLUDED.dhcp_option_set").
			Set("dhcp_option_set_hash = EXCLUDED.dhcp_option_set_hash")

		_, err := q.Insert()
		if err != nil {
			return pkgerrors.Wrapf(err, "problem associating the daemon %d with the host %d",
				host.LocalHosts[i].DaemonID, host.ID)
		}
	}
	return nil
}

// Attempts to add a host and its local hosts within an existing transaction.
func addHostWithLocalHosts(tx *pg.Tx, host *Host) error {
	err := addHost(tx, host)
	if err != nil {
		return err
	}
	err = AddHostLocalHosts(tx, host)
	return err
}

// Attempts to add a host and its local hosts within a transaction. If the dbi
// does not point to a transaction, a new transaction is started.
func AddHostWithLocalHosts(dbi dbops.DBI, host *Host) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addHostWithLocalHosts(tx, host)
		})
	}
	return addHostWithLocalHosts(dbi.(*pg.Tx), host)
}

// Dissociates a daemon from the hosts. The dataSource designates a data
// source from which the deleted hosts were fetched. If it is an empty value
// the hosts from all sources are deleted. The first returned value indicates
// if any row was removed from the local_host table.
func DeleteDaemonFromHosts(dbi dbops.DBI, daemonID int64, dataSource HostDataSource) (int64, error) {
	q := dbi.Model((*LocalHost)(nil)).
		Where("daemon_id = ?", daemonID)

	if len(dataSource) > 0 {
		q = q.Where("data_source = ?", dataSource)
	}

	result, err := q.Delete()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem deleting the daemon %d from hosts", daemonID)
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}

// Deletes hosts which are not associated with any apps. Returns deleted host
// count and an error.
func DeleteOrphanedHosts(dbi dbops.DBI) (int64, error) {
	subquery := dbi.Model(&[]LocalHost{}).
		Column("id").
		Limit(1).
		Where("host.id = local_host.host_id")
	result, err := dbi.Model(&[]Host{}).
		Where("(?) IS NULL", subquery).
		Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting orphaned hosts")
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}

// Iterates over the list of hosts and commits them into the database. The hosts
// can be associated with a subnet or can be made global. The committed hosts
// must already include associations with the daemons and other information
// specific to daemons, e.g., DHCP options.
func commitHostsIntoDB(dbi dbops.DBI, hosts []Host, subnetID int64, daemon *Daemon) (err error) {
	for i := range hosts {
		hosts[i].SubnetID = subnetID
		for j := range hosts[i].LocalHosts {
			if hosts[i].LocalHosts[j].DaemonID == 0 {
				hosts[i].LocalHosts[j].DaemonID = daemon.ID
			}
		}
		if hosts[i].ID == 0 {
			err = AddHost(dbi, &hosts[i])
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to add detected host to the database")
				return err
			}
		} else {
			err = UpdateHost(dbi, &hosts[i])
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to update detected host in the database")
				return err
			}
		}
		if err = AddHostLocalHosts(dbi, &hosts[i]); err != nil {
			return err
		}
	}
	return nil
}

// Iterates over the list of hosts and commits them as global hosts.
func CommitGlobalHostsIntoDB(dbi dbops.DBI, hosts []Host, daemon *Daemon) (err error) {
	if db, ok := dbi.(*pg.DB); ok {
		err = db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return commitHostsIntoDB(dbi, hosts, 0, daemon)
		})
		return
	}
	return commitHostsIntoDB(dbi, hosts, 0, daemon)
}

// Iterates over the hosts belonging to the given subnet and stores them
// or updates in the database.
func CommitSubnetHostsIntoDB(dbi dbops.DBI, subnet *Subnet, daemon *Daemon) (err error) {
	return commitHostsIntoDB(dbi, subnet.Hosts, subnet.ID, daemon)
}

// This function checks if the given host includes a reservation for the
// given address.
func (host Host) HasIPAddress(ipAddress string) bool {
	for _, r := range host.IPReservations {
		hostCidr, err := storkutil.MakeCIDR(r.Address)
		if err != nil {
			continue
		}
		argCidr, err := storkutil.MakeCIDR(ipAddress)
		if err != nil {
			return false
		}
		if hostCidr == argCidr {
			return true
		}
	}
	return false
}

// This function checks if the given host has specified identifier and if
// the identifier value matches. The first returned value indicates if the
// identifiers exists. The second one indicates if the value matches.
func (host Host) HasIdentifier(idType string, identifier []byte) (bool, bool) {
	for _, i := range host.HostIdentifiers {
		if idType == i.Type {
			if bytes.Equal(i.Value, identifier) {
				return true, true
			}
			return true, false
		}
	}
	return false, false
}

// This function checks if the given host has an identifier of a given type.
func (host Host) HasIdentifierType(idType string) bool {
	for _, i := range host.HostIdentifiers {
		if idType == i.Type {
			return true
		}
	}
	return false
}

// Checks if two hosts have the same IP reservations.
func (host Host) HasEqualIPReservations(other *Host) bool {
	if len(host.IPReservations) != len(other.IPReservations) {
		return false
	}

	for _, o := range other.IPReservations {
		if !host.HasIPAddress(o.Address) {
			return false
		}
	}

	return true
}

// Checks if two Host instances describe the same host. The host is
// the same when it has equal host identifiers, IP reservations and
// hostname.
func (host Host) IsSame(other *Host) bool {
	if len(host.HostIdentifiers) != len(other.HostIdentifiers) {
		return false
	}

	for _, o := range other.HostIdentifiers {
		if _, ok := host.HasIdentifier(o.Type, o.Value); !ok {
			return false
		}
	}

	if !host.HasEqualIPReservations(other) {
		return false
	}

	return host.Hostname == other.Hostname
}

// Returns local host instance for the daemon ID or nil.
func (host Host) GetLocalHost(daemonID int64) *LocalHost {
	for i := range host.LocalHosts {
		if host.LocalHosts[i].DaemonID == daemonID {
			return &host.LocalHosts[i]
		}
	}
	return nil
}

// Converts host identifier value to a string of hexadecimal digits.
func (id HostIdentifier) ToHex(separator string) string {
	// Convert binary value to hexadecimal value.
	encoded := hex.EncodeToString(id.Value)
	// If no separator specified, return what we have.
	if len(separator) == 0 {
		return encoded
	}
	var separated string
	// Iterate over pairs of hexadecimal digits and insert separator
	// between them.
	for i := 0; i < len(encoded); i += 2 {
		if len(separated) > 0 {
			separated += separator
		}
		separated += encoded[i : i+2]
	}
	return separated
}

// Count out-of-pool IPv4 and IPv6 addresses for all subnets.
// Output is a mapping between subnet ID and count.
// The function assumes that the reservation can be only in
// the subnet in which it is defined. If it is outside this
// subnet it is considered out-of-pool, even if it happens to overlap
// with another subnet.
func CountOutOfPoolAddressReservations(dbi dbops.DBI) (map[int64]uint64, error) {
	// Output row.
	// Out-of-pool count per subnet.
	var res []struct {
		SubnetID int64
		// Stork uses the int64 data type for the host reservation ID.
		// It means that we expect at most 2^64 out-of-pool reservations.
		Oop uint64
	}

	// Check if IP reservation address is in any subnet pool
	inAnyPoolSubquery := dbi.Model((*AddressPool)(nil)).
		// We don't need any data from this query, we check only row existence
		ColumnExpr("1").
		Join("JOIN local_subnet").JoinOn("address_pool.local_subnet_id = local_subnet.id").
		// We assume that the reservation can be only in
		// the subnet in which it is defined
		Where("local_subnet.subnet_id = host.subnet_id").
		// Is it in a pool? - from lower to upper bands inclusively
		Where("ip_reservation.address BETWEEN address_pool.lower_bound AND address_pool.upper_bound").
		// We want only to know if the address is in at least one pool
		Limit(1)

	// Find out-of-pool host reservations.
	err := dbi.Model((*IPReservation)(nil)).
		Column("host.subnet_id").
		ColumnExpr("COUNT(*) AS oop").
		Join("LEFT JOIN host").JoinOn("ip_reservation.host_id = host.id").
		// Exclude global reservations
		Where("host.subnet_id IS NOT NULL").
		// The IP reservation table contains the address and prefix reservations both.
		// In this query, we check out-of-pool address reservations.
		// We need to exclude prefix reservations. We take into account
		// only IPv4 reservations (as IPv4 has no prefix concept) and
		// single IPv6 hosts - entries with 128 mask length (128 mask length
		// implies that it's an IPv6 address).
		WhereGroup(func(q *pg.Query) (*pg.Query, error) {
			return q.
				Where("family(ip_reservation.address) = 4").
				WhereOr("masklen(ip_reservation.address) = 128"), nil
		}).
		// Is it out of all pools? - Is it not in any pool?
		Where("NOT EXISTS (?)", inAnyPoolSubquery).
		// Group out-of-pool reservations per subnet
		// and count them (in SELECT)
		Group("host.subnet_id").
		Select(&res)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "cannot count out-of-pool addresses")
	}

	countsPerSubnet := make(map[int64]uint64)

	for _, row := range res {
		countsPerSubnet[row.SubnetID] = row.Oop
	}

	return countsPerSubnet, nil
}

// Count out-of-pool prefixes for all subnets.
// Output is a mapping between subnet ID and count.
// The function assumes that the reservation can be only in
// the subnet in which it is defined. If it is outside this
// subnet then it is outside all subnets.
func CountOutOfPoolPrefixReservations(dbi dbops.DBI) (map[int64]uint64, error) {
	// Output row.
	// Out-of-pool count per subnet.
	var res []struct {
		SubnetID int64
		// Stork uses the int64 data type for the host reservation ID.
		// It means that we expect at most 2^64 out-of-pool reservations.
		Oop uint64
	}

	// Check if prefix reservation is in any prefix pool
	inAnyPrefixPoolSubquery := dbi.Model((*PrefixPool)(nil)).
		// We don't need any data from this query, we check only row existence
		ColumnExpr("1").
		Join("JOIN local_subnet").JoinOn("prefix_pool.local_subnet_id = local_subnet.id").
		// We assume that the reservation can be only in
		// the subnet in which it is defined
		Where("local_subnet.subnet_id = host.subnet_id").
		// Reserved prefix is in prefix pool if it is contained by the prefix of the pool
		// and if the reserved prefix length is narrower than the delegation length.
		// For example for pool 3001::/48 and delegation length equals to 64:
		// - Prefix 3001:42::/80 is not in the pool, because it has different prefix.
		// - Prefixes 3001::/48 and 3001::/62 are not in the pool. They are in an expected network
		// (has the same 48 starting bits), but the mask lengths are less then 64.
		// - Prefixes 3001::/64 and 3001::/80 are in the pool. They are in an expected network
		// and the mask lengths are greater or equals 64.
		// The `<<=` is an operator that check if the left CIDR is contained within right CIDR.
		Where("ip_reservation.address <<= prefix_pool.prefix AND masklen(ip_reservation.address) >= prefix_pool.delegated_len").
		// We want only to know if the address is in at least one pool
		Limit(1)

	// Find out-of-pool host reservations.
	err := dbi.Model((*IPReservation)(nil)).
		Column("host.subnet_id").
		ColumnExpr("COUNT(*) AS oop").
		Join("LEFT JOIN host").JoinOn("ip_reservation.host_id = host.id").
		// Exclude global reservations
		Where("host.subnet_id IS NOT NULL").
		// The IP reservation table contains the address and prefix reservations both.
		// In this query, we check out-of-pool prefix reservations.
		// We need to exclude address reservations. We take into account
		// only IPv6 reservations (as only IPv6 has prefix concept) and
		// non single IPv6 hosts - entries with mask length less then 128 (128 mask length
		// implies that they are IPv6 addresses).
		Where("family(ip_reservation.address) = 6").
		Where("masklen(ip_reservation.address) != 128").
		// Is it out of all pools? - Is it not in any pool?
		Where("NOT EXISTS (?)", inAnyPrefixPoolSubquery).
		// Group out-of-pool reservations per subnet
		// and count them (in SELECT)
		Group("host.subnet_id").
		Select(&res)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "cannot count out-of-pool prefixes")
	}

	countsPerSubnet := make(map[int64]uint64)

	for _, row := range res {
		countsPerSubnet[row.SubnetID] = row.Oop
	}

	return countsPerSubnet, nil
}

// Count global reservations of IPv4 and IPv6 addresses and delegated prefixes.
// We assume that global reservations are always out-of-pool.
// It's possible to define in-pool global reservation, but it's not recommended.
// The query without this assumption is very inefficient.
func CountGlobalReservations(dbi dbops.DBI) (ipv4Addresses, ipv6Addresses, prefixes uint64, err error) {
	// Output row.
	var res struct {
		Ipv4Addresses uint64
		Ipv6Addresses uint64
		Prefixes      uint64
	}

	err = dbi.Model((*IPReservation)(nil)).
		// Window functions aren't supported well by Go-PG
		ColumnExpr("COUNT(ip_reservation.id) FILTER (WHERE family(ip_reservation.address) = 4) AS ipv4_addresses").
		ColumnExpr("COUNT(ip_reservation.id) FILTER (WHERE family(ip_reservation.address) = 6 AND masklen(ip_reservation.address) = 128) AS ipv6_addresses").
		ColumnExpr("COUNT(ip_reservation.id) FILTER (WHERE family(ip_reservation.address) = 6 AND masklen(ip_reservation.address) != 128) AS prefixes").
		Join("LEFT JOIN host").JoinOn("ip_reservation.host_id = host.id").
		// Include only global reservations
		Where("host.subnet_id IS NULL").
		Select(&res)
	err = pkgerrors.Wrap(err, "cannot count global out-of-pool reservations")

	ipv4Addresses = res.Ipv4Addresses
	ipv6Addresses = res.Ipv6Addresses
	prefixes = res.Prefixes
	return
}

// Implementation of the keaconfig.Host interface - used in conversions
// between Host and keaconfig.Reservation.

// Returns host identifiers.
func (host Host) GetHostIdentifiers() (identifiers []struct {
	Type  string
	Value []byte
},
) {
	for _, ids := range host.HostIdentifiers {
		identifiers = append(identifiers, struct {
			Type  string
			Value []byte
		}{
			Type:  ids.Type,
			Value: ids.Value,
		})
	}
	return
}

// Returns reserved IP addresses and prefixes.
func (host Host) GetIPReservations() (ips []string) {
	for _, r := range host.IPReservations {
		ips = append(ips, r.Address)
	}
	return
}

// Returns reserved hostname.
func (host Host) GetHostname() string {
	return host.Hostname
}

// Returns reserved client classes.
func (host Host) GetClientClasses(daemonID int64) (clientClasses []string) {
	if lh := host.GetLocalHost(daemonID); lh != nil {
		clientClasses = lh.ClientClasses
	}
	return
}

// Returns reserved next server address.
func (host Host) GetNextServer(daemonID int64) (nextServer string) {
	if lh := host.GetLocalHost(daemonID); lh != nil {
		nextServer = lh.NextServer
	}
	return
}

// Returns reserved server hostname.
func (host Host) GetServerHostname(daemonID int64) (serverHostname string) {
	if lh := host.GetLocalHost(daemonID); lh != nil {
		serverHostname = lh.ServerHostname
	}
	return
}

// Returns reserved boot file name.
func (host Host) GetBootFileName(daemonID int64) (bootFileName string) {
	if lh := host.GetLocalHost(daemonID); lh != nil {
		bootFileName = lh.BootFileName
	}
	return
}

// Returns DHCP options associated with the host and for a specified
// daemon ID.
func (host Host) GetDHCPOptions(daemonID int64) (options []dhcpmodel.DHCPOptionAccessor) {
	for _, lh := range host.LocalHosts {
		if lh.DaemonID == daemonID {
			for _, o := range lh.DHCPOptionSet {
				options = append(options, o)
			}
		}
	}
	return
}

// Returns local subnet ID for a specified daemon. It returns an error
// if the specified daemon is not associated with the host. It returns 0
// if the host is not associated with a subnet (global host reservation case).
func (host Host) GetSubnetID(daemonID int64) (subnetID int64, err error) {
	if host.Subnet != nil {
		for _, ls := range host.Subnet.LocalSubnets {
			if ls.DaemonID == daemonID {
				subnetID = ls.LocalSubnetID
				return
			}
		}
		err = pkgerrors.Errorf("local subnet id not found in host %d for daemon %d", host.ID, daemonID)
	}
	return
}

// Fetches daemon information for each daemon ID within the local hosts.
// The host information can be partial when it is created from the request
// received over the REST API. In particular, the LocalHosts can merely
// contain DaemonID values and the Daemon pointers can be nil. In order
// to initialize Daemon pointers, this function fetches the daemons from
// the database and assigns them to the respective LocalHost instances.
// If any of the daemons does not exist or an error occurs, the host
// is not updated.
func (host Host) PopulateDaemons(dbi dbops.DBI) error {
	var daemons []*Daemon
	for _, lh := range host.LocalHosts {
		// DaemonID is required for this function to run.
		if lh.DaemonID == 0 {
			return pkgerrors.Errorf("problem with populating daemons: host %d lacks daemon ID", host.ID)
		}
		daemon, err := GetDaemonByID(dbi, lh.DaemonID)
		if err != nil {
			return pkgerrors.WithMessage(err, "problem with populating daemons")
		}
		// Daemon does not exist.
		if daemon == nil {
			return pkgerrors.Errorf("problem with populating daemons for host %d: daemon %d does not exist", host.ID, lh.DaemonID)
		}
		daemons = append(daemons, daemon)
	}
	// Everything fine. Assign fetched daemons to the host.
	for i := range host.LocalHosts {
		host.LocalHosts[i].Daemon = daemons[i]
	}
	return nil
}

// Fetches subnet information for a non-zero subnet ID in the host. The
// host information can be partial when it is created from the request
// received over the REST API. This function can be called to initialize
// the Subnet structure in the host with the full information about the
// subnet the host belongs to. This function is no-op when subnet ID is
// 0 or when the Subnet pointer is already non-nil. Otherwise, it fetches
// the relevant subnet information from the database. If the subnet
// doesn't exist, an error is returned.
func (host *Host) PopulateSubnet(dbi dbops.DBI) error {
	if host.SubnetID != 0 && host.Subnet == nil {
		subnet, err := GetSubnet(dbi, host.SubnetID)
		if err != nil {
			return pkgerrors.WithMessagef(err, "problem with populating subnet %d for host %d", host.SubnetID, host.ID)
		}
		if subnet == nil {
			return pkgerrors.Errorf("problem with populating subnet %d for host %d because such subnet does not exist", host.SubnetID, host.ID)
		}
		host.Subnet = subnet
	}
	return nil
}

// Sets LocalHost instance for the Host. If the corresponding LocalHost
// (having the same daemon ID) already exists, it is replaced with the
// specified instance. Otherwise, the instance is appended to the slice
// of LocalHosts.
func (host *Host) SetLocalHost(localHost *LocalHost) {
	for i, lh := range host.LocalHosts {
		if lh.DaemonID == localHost.DaemonID {
			host.LocalHosts[i] = *localHost
			return
		}
	}
	host.LocalHosts = append(host.LocalHosts, *localHost)
}

// Combines two hosts into a single host by copying LocalHost data from
// the other host. It returns a boolean value indicating whether or not
// joining the hosts was successful. It returns false when joined hosts
// are not the same ones (have different identifiers, hostnames etc.).
func (host *Host) Join(other *Host) bool {
	if !host.IsSame(other) {
		return false
	}
	for i := range other.LocalHosts {
		host.SetLocalHost(&other.LocalHosts[i])
	}
	return true
}
