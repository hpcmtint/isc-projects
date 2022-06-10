package dbmodel

import (
	"context"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Convenience function checking if the slice of hosts returned from
// the database contains the given host. It excludes the Subnet value
// in the returned hosts from the comparison.
func hostsContain(t *testing.T, returned []Host, host Host) {
	for i := range returned {
		returned[i].Subnet = nil
	}
	require.Contains(t, returned, host)
}

// This function creates multiple hosts used in tests which fetch and
// filter hosts.
func addTestHosts(t *testing.T, db *pg.DB) []Host {
	subnets := []Subnet{
		{
			ID:     1,
			Prefix: "192.0.2.0/24",
		},
		{
			ID:     2,
			Prefix: "2001:db8:1::/64",
		},
	}
	for i, s := range subnets {
		subnet := s
		err := AddSubnet(db, &subnet)
		require.NoError(t, err)
		require.NotZero(t, subnet.ID)
		subnets[i] = subnet
	}

	hosts := []Host{
		{
			SubnetID: 1,
			HostIdentifiers: []HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
				{
					Type:  "circuit-id",
					Value: []byte{0xf1, 0xf2, 0xf3, 0xf4},
				},
			},
			IPReservations: []IPReservation{
				{
					Address: "192.0.2.4/32",
				},
				{
					Address: "192.0.2.5/32",
				},
			},
			Hostname: "first.example.org",
		},
		{
			HostIdentifiers: []HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{2, 3, 4, 5, 6, 7},
				},
				{
					Type:  "circuit-id",
					Value: []byte{2, 3, 4, 5},
				},
			},
			IPReservations: []IPReservation{
				{
					Address: "192.0.2.6/32",
				},
				{
					Address: "192.0.2.7/32",
				},
			},
		},
		{
			SubnetID: 2,
			HostIdentifiers: []HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
			},
			IPReservations: []IPReservation{
				{
					Address: "2001:db8:1::1/128",
				},
			},
			Hostname: "second.example.org",
		},
		{
			HostIdentifiers: []HostIdentifier{
				{
					Type:  "duid",
					Value: []byte{1, 2, 3, 4},
				},
				{
					Type:  "flex-id",
					Value: []byte{0x51, 0x52, 0x53, 0x54},
				},
			},
			IPReservations: []IPReservation{
				{
					Address: "2001:db8:1::2/128",
				},
			},
		},
	}

	for i, h := range hosts {
		host := h
		err := AddHost(db, &host)
		require.NoError(t, err)
		require.NotZero(t, host.ID)
		hosts[i] = host
	}
	return hosts
}

// This test verifies that the new host along with identifiers and reservations
// can be added to the database.
func TestAddHost(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a host with two identifiers and two reservations.
	host := &Host{
		Hostname: "host.example.org",
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 1, 1, 1, 1},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4",
			},
			{
				Address: "2001:db8:1::4",
			},
		},
	}
	err := AddHost(db, host)
	require.NoError(t, err)
	require.NotZero(t, host.ID)

	// Get the host from the database.
	returned, err := GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	require.Equal(t, host.ID, returned.ID)
	require.Equal(t, "host.example.org", host.Hostname)
	require.Len(t, returned.HostIdentifiers, 2)
	require.Len(t, returned.IPReservations, 2)

	// Make sure that the returned host identifiers match.
	for i := range returned.HostIdentifiers {
		require.Contains(t, returned.HostIdentifiers, host.HostIdentifiers[i])
	}

	// Make sure that the returned reservations match.
	for i := range returned.IPReservations {
		require.Contains(t, returned.IPReservations[i].Address, host.IPReservations[i].Address)
	}
}

// Test that the host can be updated and that this update includes extending
// the list of reservations and identifiers.
func TestUpdateHostExtend(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add the host with two reservations and two identifiers.
	host := &Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 1, 1, 1, 1},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4/32",
			},
			{
				Address: "2001:db8:1::4/128",
			},
		},
	}
	err := AddHost(db, host)
	require.NoError(t, err)
	require.NotZero(t, host.ID)

	// Modify the value of the first identifier.
	host.HostIdentifiers[0].Value = []byte{6, 5, 4, 3, 2, 1}
	// Modify the identifier type of the second identifier.
	host.HostIdentifiers[1].Type = "client-id"
	// Add one more identifier.
	host.HostIdentifiers = append(host.HostIdentifiers, HostIdentifier{
		Type:  "flex-id",
		Value: []byte{2, 2, 2, 2, 2},
	})

	// Modify the first reservation.
	host.IPReservations[0].Address = "192.0.3.4/32"
	// Add one more reservation.
	host.IPReservations = append(host.IPReservations, IPReservation{
		Address: "3000::/64",
	})

	// Not only does updating the host modify the host value but also adds
	// or removes reservations and identifiers.
	err = UpdateHost(db, host)
	require.NoError(t, err)
	require.NotZero(t, host.ID)

	// Get the updated host.
	returned, err := GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	require.Len(t, returned.HostIdentifiers, 3)
	require.Len(t, returned.IPReservations, 3)

	// Make sure that the identifiers and reservations were modified.
	require.ElementsMatch(t, returned.HostIdentifiers, host.HostIdentifiers)
	require.ElementsMatch(t, returned.IPReservations, host.IPReservations)
}

// Test that the host can be updated and that some reservations and
// host identifiers are deleted as a result of this update.
func TestUpdateHostShrink(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	host := &Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 1, 1, 1, 1},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4/32",
			},
			{
				Address: "2001:db8:1::4/128",
			},
		},
	}
	err := AddHost(db, host)
	require.NoError(t, err)
	require.NotZero(t, host.ID)

	// Remove one host identifier and one reservation.
	host.HostIdentifiers = host.HostIdentifiers[0:1]
	host.IPReservations = host.IPReservations[1:]

	// Updating the host should result in removal of this identifier
	// and the reservation from the database.
	err = UpdateHost(db, host)
	require.NoError(t, err)
	require.NotZero(t, host.ID)

	// Get the updated host.
	returned, err := GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	// Verify that only one identifier and one reservation have left.
	require.Len(t, returned.HostIdentifiers, 1)
	require.Len(t, returned.IPReservations, 1)

	require.Equal(t, "hw-address", returned.HostIdentifiers[0].Type)
	require.Equal(t, []byte{1, 2, 3, 4, 5, 6}, returned.HostIdentifiers[0].Value)

	require.Equal(t, "2001:db8:1::4/128", returned.IPReservations[0].Address)

	// Remove the IP reservation and add the reserved hostname instead.
	host.IPReservations = []IPReservation{}
	host.Hostname = "host.example.org."
	err = UpdateHost(db, host)
	require.NoError(t, err)

	// Get the updated host.
	returned, err = GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Empty(t, returned.IPReservations)
	require.Len(t, returned.HostIdentifiers, 1)
	require.Equal(t, "host.example.org.", returned.Hostname)

	// Remove the host identifiers.
	host.HostIdentifiers = []HostIdentifier{}
	err = UpdateHost(db, host)
	require.NoError(t, err)

	// Get the updated host.
	returned, err = GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Empty(t, returned.IPReservations)
	require.Empty(t, returned.HostIdentifiers)
	require.Equal(t, "host.example.org.", returned.Hostname)

	require.Empty(t, returned.IPReservations)
	require.Empty(t, host.HostIdentifiers)
	require.Equal(t, "host.example.org.", returned.Hostname)
}

// Test that all hosts or all hosts having IP reservations of specified family
// can be fetched.
func TestGetAllHosts(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts := addTestHosts(t, db)

	// Fetch all hosts having IPv4 reservations.
	returned, err := GetAllHosts(db, 4)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	require.Contains(t, returned, hosts[0])
	require.Contains(t, returned, hosts[1])

	// Fetch all hosts having IPv6 reservations.
	returned, err = GetAllHosts(db, 6)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	require.Contains(t, returned, hosts[2])
	require.Contains(t, returned, hosts[3])

	// Fetch all hosts.
	returned, err = GetAllHosts(db, 0)
	require.NoError(t, err)
	require.Len(t, returned, 4)

	for _, host := range hosts {
		require.Contains(t, returned, host)
	}
}

// Test that hosts can be fetched by subnet ID.
func TestGetHostsBySubnetID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts := addTestHosts(t, db)

	// Fetch host having IPv4 reservations.
	returned, err := GetHostsBySubnetID(db, 1)
	require.NoError(t, err)
	require.Len(t, returned, 1)
	require.Contains(t, returned, hosts[0])

	// Fetch host having IPv6 reservations.
	returned, err = GetHostsBySubnetID(db, 2)
	require.NoError(t, err)
	require.Len(t, returned, 1)
	require.Contains(t, returned, hosts[2])
}

// Test that page of the hosts can be fetched without filtering.
func TestGetHostsByPageNoFiltering(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	_ = addTestHosts(t, db)

	returned, total, err := GetHostsByPage(db, 0, 10, 0, nil, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
}

// Test that page of the hosts can be fetched with filtering by subnet id.
func TestGetHostsByPageSubnet(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts := addTestHosts(t, db)

	// Get global hosts only.
	subnetID := int64(0)
	returned, total, err := GetHostsByPage(db, 0, 10, 0, &subnetID, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	require.Contains(t, returned, hosts[1])
	require.Contains(t, returned, hosts[3])

	// Get hosts associated with subnet id 1.
	subnetID = int64(1)
	returned, total, err = GetHostsByPage(db, 0, 10, 0, &subnetID, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returned, 1)
	require.EqualValues(t, hosts[0].ID, returned[0].ID)
	require.EqualValues(t, 1, returned[0].SubnetID)
	require.NotNil(t, returned[0].Subnet)
	require.Equal(t, "192.0.2.0/24", returned[0].Subnet.Prefix)
	require.ElementsMatch(t, returned[0].HostIdentifiers, hosts[0].HostIdentifiers)
	require.ElementsMatch(t, returned[0].IPReservations, hosts[0].IPReservations)

	// Get hosts associated with subnet id 2.
	subnetID = int64(2)
	returned, total, err = GetHostsByPage(db, 0, 10, 0, &subnetID, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returned, 1)
	require.EqualValues(t, hosts[2].ID, returned[0].ID)
	require.EqualValues(t, 2, returned[0].SubnetID)
	require.NotNil(t, returned[0].Subnet)
	require.Equal(t, "2001:db8:1::/64", returned[0].Subnet.Prefix)
	require.ElementsMatch(t, returned[0].HostIdentifiers, hosts[2].HostIdentifiers)
	require.ElementsMatch(t, returned[0].IPReservations, hosts[2].IPReservations)
}

// Test that page of the hosts can be fetched with filtering by app id.
func TestGetHostsByPageApp(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps and hosts into the database.
	apps := addTestSubnetApps(t, db)
	hosts := addTestHosts(t, db)

	// Associate the first host with the first app.
	err := AddDaemonToHost(db, &hosts[0], apps[0].Daemons[0].ID, "api")
	require.NoError(t, err)
	err = AddDaemonToHost(db, &hosts[1], apps[0].Daemons[0].ID, "api")
	require.NoError(t, err)
	err = AddDaemonToHost(db, &hosts[2], apps[1].Daemons[0].ID, "api")
	require.NoError(t, err)
	err = AddDaemonToHost(db, &hosts[3], apps[1].Daemons[0].ID, "api")
	require.NoError(t, err)

	// Get global hosts only.
	returned, total, err := GetHostsByPage(db, 0, 10, apps[0].ID, nil, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	require.True(t,
		(returned[0].ID == hosts[0].ID && returned[1].ID == hosts[1].ID) ||
			(returned[0].ID == hosts[1].ID && returned[1].ID == hosts[2].ID))
}

// Test that page of the hosts can be filtered by IP reservations and
// hostnames.
func TestGetHostsByPageFilteringText(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts := addTestHosts(t, db)

	filterText := "0.2.4"
	returned, total, err := GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returned, 1)
	require.NotNil(t, returned[0].Subnet)
	require.Equal(t, "192.0.2.0/24", returned[0].Subnet.Prefix)
	hostsContain(t, returned, hosts[0])

	filterText = "192.0.2"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	require.NotNil(t, returned[0].Subnet)
	require.Equal(t, "192.0.2.0/24", returned[0].Subnet.Prefix)
	require.Nil(t, returned[1].Subnet)
	hostsContain(t, returned, hosts[0])
	hostsContain(t, returned, hosts[1])

	filterText = "0"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)

	for i := range returned {
		returned[i].Subnet = nil
	}

	require.ElementsMatch(t, returned, hosts)

	// Case insensitive address matching.
	filterText = "2001:Db8:1"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	hostsContain(t, returned, hosts[2])
	hostsContain(t, returned, hosts[3])

	// Filter by identifier value.
	filterText = "01:02:03"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, returned, 3)
	hostsContain(t, returned, hosts[0])
	hostsContain(t, returned, hosts[2])
	hostsContain(t, returned, hosts[3])

	// Case insensitive identifier matching.
	filterText = "F1:f2:F3"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returned, 1)
	hostsContain(t, returned, hosts[0])

	// Case insensitive identifier type matching.
	filterText = "DuI"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returned, 1)
	hostsContain(t, returned, hosts[3])

	// Case insensitive hostname matching.
	filterText = "ExamplE"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	hostsContain(t, returned, hosts[0])
	hostsContain(t, returned, hosts[2])

	// Filter by partial flex-id using textual format (case insensitive).
	filterText = "qRs"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returned, 1)
	hostsContain(t, returned, hosts[3])

	// The same host should be returned for the filter text in hex format.
	filterText = "51:52:53"
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returned, 1)
	hostsContain(t, returned, hosts[3])
}

// Test that page of the hosts can be global/not global hosts.
func TestGetHostsByPageGlobal(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two global and two non-global.
	hosts := addTestHosts(t, db)

	// find only global hosts
	global := true
	returned, total, err := GetHostsByPage(db, 0, 10, 0, nil, nil, &global, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	require.Contains(t, returned, hosts[1])
	require.Contains(t, returned, hosts[3])

	// find only non-global hosts
	global = false
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, nil, &global, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	require.Contains(t, []int64{hosts[0].ID, hosts[2].ID}, returned[0].ID)
	require.Contains(t, []int64{hosts[0].ID, hosts[2].ID}, returned[1].ID)
}

// Test hosts can be sorted by different fields.
func TestGetHostsByPageWithSorting(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	addTestHosts(t, db)

	// check sorting by id asc
	returned, total, err := GetHostsByPage(db, 0, 10, 0, nil, nil, nil, "id", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
	require.EqualValues(t, 1, returned[0].ID)
	require.EqualValues(t, 4, returned[3].ID)

	// check sorting by id desc
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, nil, nil, "id", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
	require.EqualValues(t, 4, returned[0].ID)
	require.EqualValues(t, 1, returned[3].ID)

	// check sorting by subnet_id asc
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, nil, nil, "subnet_id", SortDirAsc)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
	require.EqualValues(t, 2, returned[0].ID)
	require.EqualValues(t, 3, returned[3].ID)

	// check sorting by subnet_id desc
	returned, total, err = GetHostsByPage(db, 0, 10, 0, nil, nil, nil, "subnet_id", SortDirDesc)
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, returned, 4)
	require.EqualValues(t, 3, returned[0].ID)
	require.EqualValues(t, 4, returned[3].ID)
}

// Test that page of the hosts can be fetched by daemon ID.
func TestGetHostsByDaemonID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps and hosts into the database.
	apps := addTestSubnetApps(t, db)
	hosts := addTestHosts(t, db)

	// Associate the first host with the first app.
	err := AddDaemonToHost(db, &hosts[0], apps[0].Daemons[0].ID, "api")
	require.NoError(t, err)
	err = AddDaemonToHost(db, &hosts[1], apps[0].Daemons[0].ID, "api")
	require.NoError(t, err)
	err = AddDaemonToHost(db, &hosts[2], apps[1].Daemons[0].ID, "api")
	require.NoError(t, err)
	err = AddDaemonToHost(db, &hosts[3], apps[1].Daemons[0].ID, "api")
	require.NoError(t, err)

	// Get hosts for the first daemon.
	returned, total, err := GetHostsByDaemonID(db, apps[0].Daemons[0].ID, "api")
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	require.True(t,
		(returned[0].ID == hosts[0].ID && returned[1].ID == hosts[1].ID) ||
			(returned[0].ID == hosts[1].ID && returned[1].ID == hosts[0].ID))

	// Get hosts for the second daemon.
	returned, total, err = GetHostsByDaemonID(db, apps[1].Daemons[0].ID, "")
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, returned, 2)
	require.True(t,
		(returned[0].ID == hosts[2].ID && returned[1].ID == hosts[3].ID) ||
			(returned[0].ID == hosts[3].ID && returned[1].ID == hosts[2].ID))

	// Use filtering by data source. It should return no hosts.
	returned, total, err = GetHostsByDaemonID(db, apps[0].Daemons[0].ID, "config")
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, returned)
}

// Test that the host and its identifiers and reservations can be
// deleted.
func TestDeleteHost(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	host := &Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4/32",
			},
		},
	}
	err := AddHost(db, host)
	require.NoError(t, err)
	require.NotZero(t, host.ID)

	err = DeleteHost(db, host.ID)
	require.NoError(t, err)

	returned, err := GetHost(db, host.ID)
	require.NoError(t, err)
	require.Nil(t, returned)
}

// Test that a daemon can be associated with a host.
func TestAddDaemonToHost(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps and hosts into the database.
	apps := addTestSubnetApps(t, db)
	hosts := addTestHosts(t, db)

	// Associate the first host with the first app.
	host := hosts[0]
	err := AddDaemonToHost(db, &host, apps[0].Daemons[0].ID, "api")
	require.NoError(t, err)

	// Fetch the host from the database.
	returned, err := GetHost(db, host.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)

	// Make sure that the host includes the local host information which
	// associates daemon with the host.
	require.Len(t, returned.LocalHosts, 1)
	require.Equal(t, "api", returned.LocalHosts[0].DataSource)
	require.EqualValues(t, apps[0].Daemons[0].ID, returned.LocalHosts[0].DaemonID)
	// When fetching one selected host the daemon and app information should
	// be also returned.
	require.NotNil(t, returned.LocalHosts[0].Daemon)
	require.NotNil(t, returned.LocalHosts[0].Daemon.App)

	// Get all hosts.
	returnedList, err := GetAllHosts(db, 0)
	require.NoError(t, err)
	require.Len(t, returnedList, 4)
	require.Len(t, returnedList[0].LocalHosts, 1)
	require.Equal(t, "api", returnedList[0].LocalHosts[0].DataSource)
	require.EqualValues(t, apps[0].Daemons[0].ID, returnedList[0].LocalHosts[0].DaemonID)
	// When fetching all hosts, the detailed daemon information should not be returned.
	require.Nil(t, returnedList[0].LocalHosts[0].Daemon)

	// Get the first host by reserved IP address.
	filterText := "192.0.2.4"
	returnedList, total, err := GetHostsByPage(db, 0, 10, 0, nil, &filterText, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, returnedList, 1)
	require.Len(t, returnedList[0].LocalHosts, 1)
	require.Equal(t, "api", returnedList[0].LocalHosts[0].DataSource)
	require.EqualValues(t, apps[0].Daemons[0].ID, returnedList[0].LocalHosts[0].DaemonID)
	// When fetching all hosts, the detailed app and daemon information
	// should be returned as well.
	require.NotNil(t, returnedList[0].LocalHosts[0].Daemon)
	require.NotNil(t, returnedList[0].LocalHosts[0].Daemon.App)
}

// Test that daemon's associations with multiple hosts can be removed.
func TestDeleteDaemonFromHosts(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps and hosts into the database.
	apps := addTestSubnetApps(t, db)
	hosts := addTestHosts(t, db)

	// Associate the first app with two hosts.
	err := AddDaemonToHost(db, &hosts[0], apps[0].Daemons[0].ID, "api")
	require.NoError(t, err)

	err = AddDaemonToHost(db, &hosts[1], apps[0].Daemons[0].ID, "api")
	require.NoError(t, err)

	// Associate the second app with another host.
	err = AddDaemonToHost(db, &hosts[2], apps[1].Daemons[0].ID, "api")
	require.NoError(t, err)

	// Removing associations with non-matching data source should
	// affect no hosts.
	count, err := DeleteDaemonFromHosts(db, apps[0].Daemons[0].ID, "config")
	require.NoError(t, err)
	require.Zero(t, count)

	// Remove associations of the first app.
	count, err = DeleteDaemonFromHosts(db, apps[0].Daemons[0].ID, "api")
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	// Ensure that the associations were removed for the first app.
	returned, count, err := GetHostsByPage(db, 0, 1000, apps[0].ID, nil, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.Zero(t, count)
	require.Empty(t, returned)

	// The association should still exist for the second app.
	returned, count, err = GetHostsByPage(db, 0, 1000, apps[1].ID, nil, nil, nil, "", SortDirAny)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, returned, 1)
}

// Test deleting hosts not assigned to any apps.
func TestDeleteOrphanedHosts(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps and hosts into the database.
	apps := addTestSubnetApps(t, db)
	hosts := addTestHosts(t, db)

	// Associate one of the hosts with one of the daemons. The
	// other two hosts are orphaned.
	err := AddDaemonToHost(db, &hosts[0], apps[0].Daemons[0].ID, "api")
	require.NoError(t, err)

	// Delete hosts not assigned to any apps.
	count, err := DeleteOrphanedHosts(db)
	require.NoError(t, err)
	require.EqualValues(t, len(hosts)-1, count)

	// There should be one host left.
	returned, err := GetAllHosts(db, 4)
	require.NoError(t, err)
	require.Len(t, returned, 1)
}

// Tests the function checking if the host includes a reservation for the
// given IP address.
func TestHasIPAddress(t *testing.T) {
	host := Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 2, 3, 4},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4/32",
			},
			{
				Address: "192.0.2.5/32",
			},
		},
	}

	require.True(t, host.HasIPAddress("192.0.2.4"))
	require.True(t, host.HasIPAddress("192.0.2.4/32"))
	require.True(t, host.HasIPAddress("192.0.2.5"))
	require.False(t, host.HasIPAddress("192.0.2.7/32"))
}

// Tests the function checking if the host includes a given identifier.
func TestHasIdentifier(t *testing.T) {
	host := Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 2, 3, 4},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4/32",
			},
			{
				Address: "192.0.2.5/32",
			},
		},
	}

	exists, equal := host.HasIdentifier("hw-address", []byte{1, 2, 3, 4, 5, 6})
	require.True(t, exists)
	require.True(t, equal)
	require.True(t, host.HasIdentifierType("hw-address"))

	exists, equal = host.HasIdentifier("circuit-id", []byte{1, 2, 3, 4})
	require.True(t, exists)
	require.True(t, equal)
	require.True(t, host.HasIdentifierType("circuit-id"))

	exists, equal = host.HasIdentifier("hw-address", []byte{1, 2, 3, 4})
	require.True(t, exists)
	require.False(t, equal)
	require.True(t, host.HasIdentifierType("hw-address"))

	exists, equal = host.HasIdentifier("duid", []byte{1, 2, 3, 4})
	require.False(t, exists)
	require.False(t, equal)
	require.False(t, host.HasIdentifierType("duid"))
}

// Test the functions which compares two hosts for equality and which
// compare IP reservations for equality.
func TestHostsEqual(t *testing.T) {
	host1 := Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 2, 3, 4},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4/32",
			},
			{
				Address: "192.0.2.5/32",
			},
		},
	}

	host2 := Host{
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "circuit-id",
				Value: []byte{1, 2, 3, 4},
			},
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.5/32",
			},
			{
				Address: "192.0.2.4/32",
			},
		},
	}

	require.True(t, host1.Equal(&host2))
	require.True(t, host2.Equal(&host1))
	require.True(t, host1.HasEqualIPReservations(&host2))
	require.True(t, host2.HasEqualIPReservations(&host1))

	host1.HostIdentifiers = append(host1.HostIdentifiers, HostIdentifier{
		Type:  "client-id",
		Value: []byte{1, 1, 1, 1},
	})
	host1.IPReservations = append(host1.IPReservations, IPReservation{
		Address: "192.0.2.6",
	})

	require.False(t, host1.Equal(&host2))
	require.False(t, host2.Equal(&host1))
	require.False(t, host1.HasEqualIPReservations(&host2))
	require.False(t, host2.HasEqualIPReservations(&host1))
}

func TestHostIdentifierToHex(t *testing.T) {
	id := HostIdentifier{
		Value: []byte{1, 2, 3, 4, 5, 0xa, 0xb},
	}
	require.Equal(t, "01:02:03:04:05:0a:0b", id.ToHex(":"))
	require.Equal(t, "01020304050a0b", id.ToHex(""))
}

// Tests that global host reservations and their associations with the apps
// are properly stored in the database.
func TestCommitGlobalHostsIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	apps := addTestSubnetApps(t, db)

	// Create two global hosts. The global hosts have no subnet ID.
	hosts := []Host{
		{
			HostIdentifiers: []HostIdentifier{
				{
					Type:  "hw-address",
					Value: []byte{1, 2, 3, 4, 5, 6},
				},
			},
			IPReservations: []IPReservation{
				{
					Address: "192.0.2.56",
				},
			},
		},
		{
			HostIdentifiers: []HostIdentifier{
				{
					Type:  "client-id",
					Value: []byte{1, 2, 3, 4},
				},
			},
			IPReservations: []IPReservation{
				{
					Address: "192.0.2.156",
				},
			},
		},
	}
	// Add the hosts and their associations with the daemon to the database.
	err := db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		return CommitGlobalHostsIntoDB(tx, hosts, apps[0].Daemons[0], "api")
	})
	require.NoError(t, err)

	// Fetch global hosts.
	returned, err := GetHostsBySubnetID(db, 0)
	require.NoError(t, err)
	require.Len(t, returned, 2)

	// Make sure that the returned hosts are associated with the given daemon
	// and that they remain global, i.e. subnet id is unspecified.
	for _, h := range returned {
		require.Len(t, h.LocalHosts, 1)
		require.EqualValues(t, apps[0].Daemons[0].ID, h.LocalHosts[0].DaemonID)
		require.Zero(t, h.SubnetID)
	}
}

// Test that the prefix reservations are properly recognized.
func TestIsPrefixForPrefix(t *testing.T) {
	// Arrange
	prefixes := []string{
		"30::/16",
		"AA:BB:CC:DD::/64",
		"31:00:00:01:02::/80",
	}

	// Act
	for _, prefix := range prefixes {
		reservation := &IPReservation{
			Address: prefix,
		}

		// Assert
		require.True(t, reservation.IsPrefix())
	}
}

// Test that the address reservations are not recognized as prefixes.
func TestIsPrefixForAddress(t *testing.T) {
	// Arrange
	addresses := []string{
		"10.0.0.0",
		"10.0.0.0/32",
		"88.33.153.144/32",
		"192.168.0.1",
		"192.168.0.1/32",
		"30::",
		"30::/128",
		"AA:BB:CC:DD::EE:FF",
		"AA:BB:CC:DD::EE:FF/128",
		"01:02:03:04:05:06:07:08:09:10:11:12:13:14:15:16",
		"01:02:03:04:05:06:07:08:09:10:11:12:13:14:15:16/128",
		"",
	}

	// Act
	for _, address := range addresses {
		reservation := &IPReservation{
			Address: address,
		}

		// Assert
		require.False(t, reservation.IsPrefix())
	}
}

// Test calculating out-of-pool reservations for IPv4 and IPv6 networks.
func TestCountOutOfPoolCounters(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// IPv4
	subnetIPv4 := &Subnet{
		Prefix: "192.0.2.0/24",
		AddressPools: []AddressPool{
			{
				LowerBound: "192.0.2.1",
				UpperBound: "192.0.2.10",
			},
			{
				LowerBound: "192.0.2.21",
				UpperBound: "192.0.2.30",
			},
		},
	}
	_ = AddSubnet(db, subnetIPv4)

	host := &Host{
		CreatedAt: time.Now(),
		SubnetID:  subnetIPv4.ID,
		Hostname:  "foo",
		IPReservations: []IPReservation{
			{
				// In pool
				Address: "192.0.2.5",
			},
			{
				// In pool
				Address: "192.0.2.10",
			},
			{
				// Out of pool
				Address: "192.0.2.15",
			},
		},
	}
	_ = AddHost(db, host)

	host = &Host{
		CreatedAt: time.Now(),
		SubnetID:  subnetIPv4.ID,
		Hostname:  "bar",
		IPReservations: []IPReservation{
			{
				// In pool
				Address: "192.0.2.21",
			},
			{
				// In pool
				Address: "192.0.2.25",
			},
			{
				// Out of pool
				Address: "192.0.2.40",
			},
		},
	}
	_ = AddHost(db, host)

	// IPv6
	subnetIPv6 := &Subnet{
		Prefix: "fe80::/64",
		AddressPools: []AddressPool{
			{
				LowerBound: "fe80::1",
				UpperBound: "fe80::10",
			},
			{
				LowerBound: "fe80::21",
				UpperBound: "fe80::30",
			},
		},
		PrefixPools: []PrefixPool{
			{
				Prefix:       "3001:1::/48",
				DelegatedLen: 64,
			},
			{
				Prefix:       "3001:2::/64",
				DelegatedLen: 80,
			},
		},
	}
	_ = AddSubnet(db, subnetIPv6)

	host = &Host{
		CreatedAt: time.Now(),
		SubnetID:  subnetIPv6.ID,
		Hostname:  "baz",
		IPReservations: []IPReservation{
			// Addresses
			{
				// In pool
				Address: "fe80::5",
			},
			{
				// In pool
				Address: "fe80::10",
			},
			{
				// Out of pool
				Address: "fe80::15",
			},
			// Prefixes
			{
				// Out of pool - different prefix
				Address: "3001:3::/96",
			},
			{
				// Out of pool - prefix contains the pool prefix
				Address: "3001::/16",
			},
			{
				// Out of pool - mask length less than the length of the pool prefix
				Address: "3001:1::/32",
			},
			{
				// Out of pool - mask length between the length of the pool prefix and
				// the delegation length
				Address: "3001:1::/58",
			},
			{
				// Out of prefix pool, but in the subnet
				Address: "fe80::/80",
			},
			{
				// In pool, mask length equals to the delegation length
				Address: "3001:1:0:10::/64",
			},
			{
				// In pool, mask length greater than the delegation length
				Address: "3001:1:0:10:20::/80",
			},
		},
	}
	_ = AddHost(db, host)

	// Global reservations
	host = &Host{
		CreatedAt: time.Now(),
		SubnetID:  0,
		Hostname:  "biz",
		IPReservations: []IPReservation{
			// Addresses
			{
				Address: "10.42.0.1",
			},
			{
				Address: "10.42.0.2",
			},
			{
				Address: "10.42.0.3",
			},
			{
				Address: "EC::1",
			},
			{
				Address: "EC::2",
			},
			// Prefixes
			{
				Address: "DD:1::/64",
			},
			{
				Address: "DD:2::/64",
			},
			{
				Address: "DD:3::/64",
			},
			{
				Address: "DD:4::/64",
			},
		},
	}

	_ = AddHost(db, host)

	// Act
	addressCounters, errAddresses := CountOutOfPoolAddressReservations(db)
	prefixCounters, errPrefixes := CountOutOfPoolPrefixReservations(db)
	globalAddresses, globalNAs, globalPDs, errGlobal := CountGlobalReservations(db)

	// Assert
	require.NoError(t, errAddresses)
	require.NoError(t, errPrefixes)
	require.NoError(t, errGlobal)

	require.EqualValues(t, 2, addressCounters[subnetIPv4.ID])
	require.EqualValues(t, 1, addressCounters[subnetIPv6.ID])
	require.Len(t, addressCounters, 2)

	require.EqualValues(t, 5, prefixCounters[subnetIPv6.ID])
	require.Len(t, prefixCounters, 1)

	require.EqualValues(t, 3, globalAddresses)
	require.EqualValues(t, 2, globalNAs)
	require.EqualValues(t, 4, globalPDs)
}

// Test that Host properly implements keaconfig.Host interface.
func TestKeaConfigHostInterface(t *testing.T) {
	host := &Host{
		Subnet: &Subnet{
			LocalSubnets: []*LocalSubnet{
				{
					DaemonID:      1,
					LocalSubnetID: 123,
				},
				{
					DaemonID:      2,
					LocalSubnetID: 234,
				},
			},
		},
		Hostname: "host.example.org",
		HostIdentifiers: []HostIdentifier{
			{
				Type:  "hw-address",
				Value: []byte{1, 2, 3, 4, 5, 6},
			},
			{
				Type:  "circuit-id",
				Value: []byte{1, 1, 1, 1, 1},
			},
		},
		IPReservations: []IPReservation{
			{
				Address: "192.0.2.4",
			},
			{
				Address: "2001:db8:1::4",
			},
		},
		LocalHosts: []LocalHost{
			{
				DaemonID: 1,
				DHCPOptionSet: []DHCPOption{
					{
						Code:        123,
						Encapsulate: "dhcp4",
						Universe:    storkutil.IPv4,
					},
				},
			},
		},
	}
	ids := host.GetHostIdentifiers()
	require.Len(t, ids, 2)
	require.Equal(t, "hw-address", ids[0].Type)
	require.Equal(t, []byte{1, 2, 3, 4, 5, 6}, ids[0].Value)
	require.Equal(t, "circuit-id", ids[1].Type)
	require.Equal(t, []byte{1, 1, 1, 1, 1}, ids[1].Value)
	ips := host.GetIPReservations()
	require.Len(t, ips, 2)
	require.Equal(t, "host.example.org", host.GetHostname())
	subnetID, err := host.GetSubnetID(1)
	require.NoError(t, err)
	require.EqualValues(t, 123, subnetID)
	subnetID, err = host.GetSubnetID(2)
	require.NoError(t, err)
	require.EqualValues(t, 234, subnetID)
	_, err = host.GetSubnetID(3)
	require.Error(t, err)
	options := host.GetDHCPOptions(1)
	require.Len(t, options, 1)
	require.EqualValues(t, 123, options[0].GetCode())
	require.Equal(t, "dhcp4", options[0].GetEncapsulate())
	require.Equal(t, storkutil.IPv4, options[0].GetUniverse())
}

// Test that GetSubnet() function returns zero when host reservation is
// not associated with any subnet.
func TestKeaConfigHostInterfaceNoSubnet(t *testing.T) {
	host := &Host{}
	subnetID, err := host.GetSubnetID(1)
	require.NoError(t, err)
	require.Zero(t, subnetID)
}

// Test that daemon information can be populated to the existing
// host instance when LocalHost instances merely contain DaemonID
// values.
func TestPopulateHostDaemons(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps to the database.
	apps := addTestSubnetApps(t, db)

	// Create bare host that lacks Daemon instances but has valid DaemonID values.
	host := &Host{
		LocalHosts: []LocalHost{
			{
				DaemonID: apps[0].Daemons[0].ID,
			},
			{
				DaemonID: apps[1].Daemons[0].ID,
			},
		},
	}
	err := host.PopulateDaemons(db)
	require.NoError(t, err)

	// Make sure that the daemon information was assigned to the host.
	require.Len(t, host.LocalHosts, 2)
	require.NotNil(t, host.LocalHosts[0].Daemon)
	require.EqualValues(t, apps[0].Daemons[0].ID, host.LocalHosts[0].Daemon.ID)
	require.NotNil(t, host.LocalHosts[1].Daemon)
	require.EqualValues(t, apps[1].Daemons[0].ID, host.LocalHosts[1].Daemon.ID)
}

// Test that an attempt to populate daemon information to a host fails when one
// of the daemons does not exist.
func TestPopulateHostDaemonsMissingDaemons(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps and hosts into the database.
	apps := addTestSubnetApps(t, db)
	host := &Host{
		LocalHosts: []LocalHost{
			{
				DaemonID: apps[0].Daemons[0].ID,
			},
			{
				DaemonID: apps[0].Daemons[0].ID + apps[1].Daemons[0].ID,
			},
		},
	}
	err := host.PopulateDaemons(db)
	require.Error(t, err)

	// The host should not be updated because of an error.
	require.Len(t, host.LocalHosts, 2)
	require.Nil(t, host.LocalHosts[0].Daemon)
	require.Nil(t, host.LocalHosts[1].Daemon)
}

// Test that subnet information can be populated to the existing host
// instance when subnet ID is available.
func TestPopulateSubnet(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Insert apps to the database.
	apps := addTestSubnetApps(t, db)

	// Insert a subnet that matches one of the subnets in the apps'
	// configurations.
	subnet := &Subnet{
		ID:     1,
		Prefix: "192.0.2.0/24",
	}
	err := AddSubnet(db, subnet)
	require.NoError(t, err)
	require.NotZero(t, subnet.ID)

	// Associate the subnet with one of the daemons.
	err = AddDaemonToSubnet(db, subnet, apps[0].Daemons[0])
	require.NoError(t, err)

	// Create the host under test and populate the subnet. It should
	// fetch the subnet from the database and assign to the host struct.
	host := &Host{
		SubnetID: 1,
	}
	err = host.PopulateSubnet(db)
	require.NoError(t, err)

	// Make sure the subnet has been assigned and that it contains
	// the association with the daemon.
	require.NotNil(t, host.Subnet)
	require.Len(t, host.Subnet.LocalSubnets, 1)
	require.EqualValues(t, 123, host.Subnet.LocalSubnets[0].LocalSubnetID)
}

// Test that an error is returned when populated subnet doesn't exist.
func TestPopulateNonExistingSubnet(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	host := &Host{
		SubnetID: 1,
	}
	err := host.PopulateSubnet(db)
	require.Error(t, err)
}

// Test that subnet is not populated when subnet ID is 0.
func TestPopulateNoSubnet(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	host := &Host{
		SubnetID: 0,
	}
	err := host.PopulateSubnet(db)
	require.NoError(t, err)
	require.Nil(t, host.Subnet)
}

// Test that the subnet is not populated the second time.
func TestPopulateSubnetAlreadyPopulated(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	host := &Host{
		SubnetID: 1,
		Subnet: &Subnet{
			ID: 1234,
		},
	}
	err := host.PopulateSubnet(db)
	require.NoError(t, err)
	require.NotNil(t, host.Subnet)
	require.EqualValues(t, 1234, host.Subnet.ID)
}
