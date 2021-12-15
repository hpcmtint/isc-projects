package configreview

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

// Creates review context from configuration string.
func createReviewContext(t *testing.T, configStr string) *ReviewContext {
	config, err := dbmodel.NewKeaConfigFromJSON(configStr)
	require.NoError(t, err)

	// Configuration must contain one of the keywords that identify the
	// daemon type.
	daemonName := dbmodel.DaemonNameDHCPv4
	if strings.Contains(configStr, "Dhcp6") {
		daemonName = dbmodel.DaemonNameDHCPv6
	}
	// Create the daemon instance and the context.
	ctx := newReviewContext(&dbmodel.Daemon{
		ID:   1,
		Name: daemonName,
		KeaDaemon: &dbmodel.KeaDaemon{
			Config: config,
		},
	}, false, nil)
	require.NotNil(t, ctx)

	return ctx
}

// Tests that the checker checking stat_cmds hooks library presence
// returns nil when the library is loaded.
func TestStatCmdsPresent(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_stat_cmds.so"
                }
            ]
        }
    }`
	report, err := statCmdsPresence(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker checking stat_cmds hooks library presence
// returns the report when the library is not loaded.
func TestStatCmdsAbsent(t *testing.T) {
	configStr := `{"Dhcp4": { }}`
	report, err := statCmdsPresence(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "The Kea Statistics Commands library")
}

// Tests that the checker checking host_cmds hooks library presence
// returns nil when the library is loaded.
func TestHostCmdsPresent(t *testing.T) {
	// The host backend is in use and the library is loaded.
	configStr := `{
        "Dhcp4": {
            "hosts-database": [
                {
                    "type": "mysql"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`
	report, err := hostCmdsPresence(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker checking host_cmds presence takes into
// account whether or not the host-database(s) parameters are
// also specified.
func TestHostCmdsBackendUnused(t *testing.T) {
	// The backend is not used and the library is not loaded.
	// There should be no report.
	configStr := `{
        "Dhcp4": { }
    }`
	report, err := hostCmdsPresence(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker checking host_cmds hooks library presence
// returns the report when the library is not loaded but the
// host-database (singular) parameter is specified.
func TestHostCmdsAbsentHostsDatabase(t *testing.T) {
	// The host backend is in use but the library is not loaded.
	// Expecting the report.
	configStr := `{
        "Dhcp4": {
            "hosts-database": {
                "type": "mysql"
            }
        }
    }`
	report, err := hostCmdsPresence(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "Kea can be configured")
}

// Tests that the checker checking host_cmds hooks library presence
// returns the report when the library is not loaded but the
// hosts-databases (plural) parameter is specified.
func TestHostCmdsAbsentHostsDatabases(t *testing.T) {
	// The host backend is in use but the library is not loaded.
	// Expecting the report.
	configStr := `{
        "Dhcp4": {
            "hosts-databases": [
                {
                    "type": "mysql"
                }
            ]
        }
    }`
	report, err := hostCmdsPresence(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "Kea can be configured")
}

// Tests that the checker finding dispensable shared networks finds
// an empty IPv4 shared network.
func TestSharedNetworkDispensableNoDHCPv4Subnet(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "foo"
                },
                {
                    "name": "bar",
                    "subnet4": [
                        {
                            "subnet": "192.0.2.0/24"
                        },
                        {
                            "subnet": "192.0.3.0/24"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration comprises 1 empty shared network")
}

// Tests that the checker finding dispensable shared networks finds
// an IPv4 shared network with a single subnet.
func TestSharedNetworkDispensableSingleDHCPv4Subnet(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "bar",
                    "subnet4": [
                        {
                            "subnet": "192.0.2.0/24"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration comprises 1 shared network with only a single subnet")
}

// Tests that the checker finding dispensable shared networks finds
// multiple empty IPv4 shared networks and multiple Ipv4 shared networks
// with a single subnet.
func TestSharedNetworkDispensableSomeEmptySomeWithSingleSubnet(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "foo"
                },
                {
                    "name": "bar"
                },
                {
                    "name": "baz",
                    "subnet4": [
                        {
                            "subnet": "192.0.2.0/24"
                        }
                    ]
                },
                {
                    "name": "zab",
                    "subnet4": [
                        {
                            "subnet": "192.0.3.0/24"
                        }
                    ]
                },
                {
                    "name": "bac",
                    "subnet4": [
                        {
                            "subnet": "192.0.4.0/24"
                        },
                        {
                            "subnet": "192.0.5.0/24"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration comprises 2 empty shared networks and 2 shared networks with only a single subnet")
}

// Tests that the checker finding dispensable shared networks does not
// generate a report when there are no empty shared networks nor the
// shared networks with a single subnet.
func TestSharedNetworkDispensableMultipleDHCPv4Subnets(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "bar",
                    "subnet4": [
                        {
                            "subnet": "192.0.2.0/24"
                        },
                        {
                            "subnet": "192.0.3.0/24"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable shared networks finds
// an empty IPv6 shared network.
func TestSharedNetworkDispensableNoDHCPv6Subnet(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "name": "foo"
                },
                {
                    "name": "bar",
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64"
                        },
                        {
                            "subnet": "2001:db8:2::/64"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration comprises 1 empty shared network")
}

// Tests that the checker finding dispensable shared networks finds
// an IPv6 shared network with a single subnet.
func TestSharedNetworkDispensableSingleDHCPv6Subnet(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "name": "bar",
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration comprises 1 shared network with only a single subnet")
}

// Tests that the checker finding dispensable shared networks does not
// generate a report when there are no empty shared networks nor the
// shared networks with a single subnet.
func TestSharedNetworkDispensableMultipleDHCPv6Subnets(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "name": "bar",
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64"
                        },
                        {
                            "subnet": "2001:db8:2::/64"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := sharedNetworkDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets finds the subnets
// that comprise no pools and no reservations.
func TestIPv4SubnetDispensableNoPoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet4": [
                        {
                            "subnet": "192.0.2.0/24"
                        }
                    ]
                }
            ],
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24"
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration comprises 2 subnets without pools and host reservations")
}

// Tests that the checker finding dispensable subnets does not generate
// a report when host_cmds hooks library is used.
func TestIPv4SubnetDispensableNoPoolsNoReservationsHostCmds(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet4": [
                        {
                            "subnet": "192.0.2.0/24"
                        }
                    ]
                }
            ],
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets does not generate
// a report when pools are present.
func TestIPv4SubnetDispensableSomePoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets does not generate
// a report when reservations are present.
func TestIPv4SubnetDispensableNoPoolsSomeReservations(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "reservations": [
                        {
                            "ip-address": "192.0.3.10",
                            "hw-address": "01:02:03:04:05:06"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets finds the subnets
// that comprise no pools and no reservations.
func TestIPv6SubnetDispensableNoPoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64"
                        }
                    ]
                }
            ],
            "subnet6": [
                {
                    "subnet": "2001:db8:2::/64"
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "configuration comprises 2 subnets without pools and host reservations")
}

// Tests that the checker finding dispensable subnets does not generate
// a report when host_cmds hooks library is used.
func TestIPv6SubnetDispensableNoPoolsNoReservationsHostCmds(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "name": "foo",
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64"
                        }
                    ]
                }
            ],
            "subnet6": [
                {
                    "subnet": "2001:db8:2::/64"
                }
            ],
            "hooks-libraries": [
                {
                    "library": "/usr/lib/kea/libdhcp_host_cmds.so"
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets does not generate
// a report when pools are present.
func TestIPv6SubnetDispensableSomePoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::5 - 2001:db8:1::15"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets does not generate
// a report when prefix delegation pools are present.
func TestIPv6SubnetDispensableSomePdPoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pd-pools": [
                        {
                            "prefix": "3001::/16",
                            "prefix-len": 64,
                            "delegated-len": 96
                        }
                    ]
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker finding dispensable subnets does not generate
// a report when reservations are present.
func TestIPv6SubnetDispensableNoPoolsSomeReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "reservations": [
                        {
                            "ip-addresses": [ "2001:db8:1::10" ],
                            "hw-address": "01:02:03:06:05:06"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := subnetDispensable(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the global
// subnets list.
func TestDHCPv4ReservationsOutOfPoolTopLevelSubnet(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ],
                    "reservations": [
                        {
                            "ip-address": "192.0.3.5"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "comprises 1 subnet for which it is recommended to use out-of-pool")
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the shared
// networks.
func TestDHCPv4ReservationsOutOfPoolSharedNetwork(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "shared-networks": [
                {
                    "subnet4": [
                        {
                            "subnet": "192.0.3.0/24",
                            "pools": [
                                {
                                    "pool": "192.0.3.10 - 192.0.3.100"
                                }
                            ],
                            "reservations": [
                                {
                                    "ip-address": "192.0.3.5"
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used respects the out-of-pool mode
// specified at the global level.
func TestDHCPv4ReservationsOutOfPoolEnabledGlobally(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "reservations-out-of-pool": true,
            "shared-networks": [
                {
                    "subnet4": [
                        {
                            "subnet": "192.0.3.0/24",
                            "pools": [
                                {
                                    "pool": "192.0.3.10 - 192.0.3.100"
                                }
                            ],
                            "reservations": [
                                {
                                    "ip-address": "192.0.3.5"
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used respects the out-of-pool mode
// specified at the shared network level.
func TestDHCPv4ReservationsOutOfPoolEnabledAtSharedNetworkLevel(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "reservations-out-of-pool": false,
            "shared-networks": [
                {
                    "reservation-mode": "out-of-pool",
                    "subnet4": [
                        {
                            "subnet": "192.0.3.0/24",
                            "pools": [
                                {
                                    "pool": "192.0.3.10 - 192.0.3.100"
                                }
                            ],
                            "reservations": [
                                {
                                    "ip-address": "192.0.3.5"
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used respects the out-of-pool mode
// specified at the subnet level.
func TestDHCPv4ReservationsOutOfPoolEnabledAtSubnetLevel(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "reservations-out-of-pool": false,
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "reservations-out-of-pool": true,
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ],
                    "reservations": [
                        {
                            "ip-address": "192.0.3.5"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when there are
// no reservations in the subnet.
func TestDHCPv4ReservationsOutOfPoolNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns the report when a subnet has
// reservations but no pools.
func TestDHCPv4ReservationsOutOfPoolNoPools(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "reservations": [
                        {
                            "ip-address": "192.0.3.5"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when a subnet has
// no reservations.
func TestDHCPv4ReservationsOutOfPoolNoPoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24"
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when a subnet has
// reservations but they contain no IP addresses.
func TestDHCPv4ReservationsOutOfPoolNoPoolsNonIPReservations(t *testing.T) {
	configStr := `{
        "Dhcp4": {
            "subnet4": [
                {
                    "subnet": "192.0.3.0/24",
                    "pools": [
                        {
                            "pool": "192.0.3.10 - 192.0.3.100"
                        }
                    ],
                    "reservations": [
                        {
                            "hostname": "myhost123.example.org"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the global
// subnets list.
func TestDHCPv6ReservationsOutOfPoolTopLevelSubnet(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ],
                    "reservations": [
                        {
                            "ip-addresses": [ "2001:db8:1::5" ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "comprises 1 subnet for which it is recommended to use out-of-pool")
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the global
// subnets list. Prefix delegation case.
func TestDHCPv6ReservationsOutOfPDPoolTopLevelSubnet(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pd-pools": [
                        {
                            "prefix": "3000::",
                            "prefix-len": 64,
                            "delegated-len": 96
                        }
                    ],
                    "reservations": [
                        {
                            "prefixes": [ "3001::/96" ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Contains(t, report.content, "comprises 1 subnet for which it is recommended to use out-of-pool")
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when reserved
// IP address is within the pool.
func TestDHCPv6ReservationsOutOfPoolTopLevelSubnetInPool(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ],
                    "reservations": [
                        {
                            "ip-addresses": [ "2001:db8:1::30" ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when reserved
// delegated prefix is within the prefix delegation pool.
func TestDHCPv6ReservationsOutOfPoolTopLevelSubnetInPDPool(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pd-pools": [
                        {
                            "prefix": "3000::",
                            "prefix-len": 64,
                            "delegated-len": 96
                        }
                    ],
                    "reservations": [
                        {
                            "prefixes": [ "3000::/96" ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the shared
// networks.
func TestDHCPv6ReservationsOutOfPoolSharedNetwork(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64",
                            "pools": [
                                {
                                    "pool": "2001:db8:1::10 - 2001:db8:1::100"
                                }
                            ],
                            "reservations": [
                                {
                                    "ip-addresses": [ "2001:db8:1::5" ]
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used finds these subnets in the shared
// networks. Prefix delegation case.
func TestDHCPv6ReservationsOutOfPDPoolSharedNetwork(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "shared-networks": [
                {
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64",
                            "pd-pools": [
                                {
                                    "prefix": "3000::",
                                    "prefix-len": 64,
                                    "delegated-len": 96
                                }
                            ],
                            "reservations": [
                                {
                                    "prefixes": [ "3001::/96" ]
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used respects the out-of-pool mode
// specified at the global level.
func TestDHCPv6ReservationsOutOfPoolEnabledGlobally(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "reservations-out-of-pool": true,
            "shared-networks": [
                {
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64",
                            "pools": [
                                {
                                    "pool": "2001:db8:1::10 - 2001:db8:1::100"
                                }
                            ],
                            "reservations": [
                                {
                                    "ip-addresses": [ "2001:db8:1::5" ]
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used respects the out-of-pool mode
// specified at the shared network level.
func TestDHCPv6ReservationsOutOfPoolEnabledAtSharedNetworkLevel(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "reservations-out-of-pool": false,
            "shared-networks": [
                {
                    "reservation-mode": "out-of-pool",
                    "subnet6": [
                        {
                            "subnet": "2001:db8:1::/64",
                            "pools": [
                                {
                                    "pool": "2001:db8:1::10 - 2001:db8:1::100"
                                }
                            ],
                            "reservations": [
                                {
                                    "ip-addresses": [ "2001:db8:1::5" ]
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used respects the out-of-pool mode
// specified at the subnet level.
func TestDHCPv6ReservationsOutOfPoolEnabledAtSubnetLevel(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "reservations-out-of-pool": false,
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "reservations-out-of-pool": true,
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ],
                    "reservations": [
                        {
                            "ip-addresses": [ "2001:db8:1::5" ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when there are
// no reservations in the subnet.
func TestDHCPv6ReservationsOutOfPoolNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns the report when a subnet has
// reservations but no pools.
func TestDHCPv6ReservationsOutOfPoolNoPools(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "reservations": [
                        {
                            "ip-addresses": [ "2001:db8:1::5" ]
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.NotNil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when a subnet has
// no reservations.
func TestDHCPv6ReservationsOutOfPoolNoPoolsNoReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64"
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}

// Tests that the checker identifying subnets in which out-of-pool
// reservation mode can be used returns no report when a subnet has
// reservations but they contain neither IP addresses nor delegated
// prefixes.
func TestDHCPv6ReservationsOutOfPoolNoPoolsNonIPReservations(t *testing.T) {
	configStr := `{
        "Dhcp6": {
            "subnet6": [
                {
                    "subnet": "2001:db8:1::/64",
                    "pools": [
                        {
                            "pool": "2001:db8:1::10 - 2001:db8:1::100"
                        }
                    ],
                    "reservations": [
                        {
                            "hostname": "myhost123.example.org"
                        }
                    ]
                }
            ]
        }
    }`
	report, err := reservationsOutOfPool(createReviewContext(t, configStr))
	require.NoError(t, err)
	require.Nil(t, report)
}
