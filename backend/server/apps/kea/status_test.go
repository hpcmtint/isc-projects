package kea

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test"
)

// Generates a response to the status-get command including two status
// structures, one for DHCPv4 and one for DHCPv6.
func mockGetStatusWithHA(callNo int, cmdResponses []interface{}) {
	daemons, _ := agentcomm.NewKeaDaemons("dhcp4", "dhcp6")
	command, _ := agentcomm.NewKeaCommand("status-get", daemons, nil)
	var json string
	switch callNo {
	case 0:
		json = `[{
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "ha-servers":
                    {
                        "local": {
                            "role": "primary",
                            "scopes": [ "server1" ],
                            "state": "load-balancing"
                        },
                        "remote": {
                            "age": 10,
                            "in-touch": true,
                            "role": "secondary",
                            "last-scopes": [ "server2" ],
                            "last-state": "load-balancing"
                        }
                    }
              }
         },
         {
             "result": 0,
             "text": "Everything is fine",
             "arguments": {
                 "pid": 2345,
                 "uptime": 3333,
                 "reload": 2222,
                 "ha-servers":
                     {
                         "local": {
                             "role": "standby",
                             "scopes": [ ],
                             "state": "hot-standby"
                         },
                         "remote": {
                             "age": 3,
                             "in-touch": true,
                             "role": "primary",
                             "last-scopes": [ "server1" ],
                             "last-state": "waiting"
                         }
                     }
               }
          }]`
	default:
		json = `[{
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "ha-servers":
                    {
                        "local": {
                            "role": "primary",
                            "scopes": [ "server1", "server2" ],
                            "state": "partner-down"
                        },
                        "remote": {
                            "age": 0,
                            "in-touch": false,
                            "role": "secondary",
                            "last-scopes": [ ],
                            "last-state": "unavailable"
                        }
                    }
              }
         },
         {
             "result": 1,
             "text": "Unable to communicate"
          }]`
	}
	_ = agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
}

// Generates a response to the status-get command including two status
// structures, one for DHCPv4 and one for DHCPv6. Format supported by
// Kea 1.7.8 onwards.
func mockGetStatusWithHA178(callNo int, cmdResponses []interface{}) {
	daemons, _ := agentcomm.NewKeaDaemons("dhcp4", "dhcp6")
	command, _ := agentcomm.NewKeaCommand("status-get", daemons, nil)
	var json string
	switch callNo {
	case 0:
		json = `[{
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "high-availability": [
                    {
                        "ha-mode": "load-balancing",
                        "ha-servers":
                            {
                                "local": {
                                "role": "primary",
                                "scopes": [ "server1" ],
                                "state": "load-balancing"
                            },
                            "remote": {
                                "age": 10,
                                "in-touch": true,
                                "role": "secondary",
                                "last-scopes": [ "server2" ],
                                "last-state": "load-balancing",
                                "communication-interrupted": true,
                                "connecting-clients": 1,
                                "unacked-clients": 2,
                                "unacked-clients-left": 3,
                                "analyzed-packets": 10
                            }
                        }
                    }
               ]
         }},
         {
             "result": 0,
             "text": "Everything is fine",
             "arguments": {
                 "pid": 2345,
                 "uptime": 3333,
                 "reload": 2222,
                 "high-availability": [
                     {
                         "ha-mode": "hot-standby",
                         "ha-servers":
                             {
                                 "local": {
                                     "role": "standby",
                                     "scopes": [ ],
                                     "state": "hot-standby"
                                 },
                                 "remote": {
                                     "age": 3,
                                     "in-touch": true,
                                     "role": "primary",
                                     "last-scopes": [ "server1" ],
                                     "last-state": "waiting",
                                     "communication-interrupted": true,
                                     "connecting-clients": 2,
                                     "unacked-clients": 3,
                                     "unacked-clients-left": 4,
                                     "analyzed-packets": 15
                                 }
                             }
                      }
                 ]
          }}]`
	default:
		json = `[{
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "high-availability": [
                    {
                        "ha-mode": "hot-standby",
                        "ha-servers":
                            {
                                "local": {
                                    "role": "primary",
                                    "scopes": [ "server1", "server2" ],
                                    "state": "partner-down"
                                },
                                "remote": {
                                    "age": 0,
                                    "in-touch": false,
                                    "role": "secondary",
                                    "last-scopes": [ ],
                                    "last-state": "unavailable",
                                    "communication-interrupted": true,
                                    "connecting-clients": 2,
                                    "unacked-clients": 3,
                                    "unacked-clients-left": 1,
                                    "analyzed-packets": 20
                                }
                           }
                    }
               ]
         }},
         {
             "result": 1,
             "text": "Unable to communicate"
         }]`
	}
	_ = agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
}

// Generate test response to status-get command including status of the
// HA pair doing load balancing.
func mockGetStatusLoadBalancing(callNo int, cmdResponses []interface{}) {
	daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
	command, _ := agentcomm.NewKeaCommand("status-get", daemons, nil)
	json := `[
        {
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "ha-servers":
                    {
                        "local": {
                            "role": "primary",
                            "scopes": [ "server1" ],
                            "state": "load-balancing"
                        },
                        "remote": {
                            "age": 10,
                            "in-touch": true,
                            "role": "secondary",
                            "last-scopes": [ "server2" ],
                            "last-state": "load-balancing"
                        }
                    }
                }
            }
    ]`
	_ = agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
}

// Generate test response to status-get command including status of the
// HA pair doing load balancing. Format supported by Kea 1.7.8 onwards.
func mockGetStatusLoadBalancing178(callNo int, cmdResponses []interface{}) {
	daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
	command, _ := agentcomm.NewKeaCommand("status-get", daemons, nil)
	json := `[
        {
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111,
                "high-availability": [
                    {
                        "ha-mode": "load-balancing",
                        "ha-servers":
                            {
                                "local": {
                                    "role": "primary",
                                    "scopes": [ "server1" ],
                                    "state": "load-balancing"
                                },
                                "remote": {
                                    "age": 10,
                                    "in-touch": true,
                                    "role": "secondary",
                                    "last-scopes": [ "server2" ],
                                    "last-state": "load-balancing",
                                    "communication-interrupted": true,
                                    "connecting-clients": 1,
                                    "unacked-clients": 2,
                                    "unacked-clients-left": 3,
                                    "analyzed-packets": 10
                                }
                            }
                    }
                ]
            }
        }
    ]`
	_ = agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
}

// Generates test response to status-get command lacking a status of the
// HA pair.
func mockGetStatusNoHA(callNo int, cmdResponses []interface{}) {
	daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
	command, _ := agentcomm.NewKeaCommand("status-get", daemons, nil)
	json := `[
        {
            "result": 0,
            "text": "Everything is fine",
            "arguments": {
                "pid": 1234,
                "uptime": 3024,
                "reload": 1111
            }
        }
    ]`
	_ = agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
}

// Generates test response to status-get command indicating an error and
// lacking argument.s
func mockGetStatusError(callNo int, cmdResponses []interface{}) {
	daemons, _ := agentcomm.NewKeaDaemons("dhcp4")
	command, _ := agentcomm.NewKeaCommand("status-get", daemons, nil)
	json := `[
        {
            "result": 1,
            "text": "unable to communicate with the deamon"
        }
    ]`
	_ = agentcomm.UnmarshalKeaResponseList(command, json, cmdResponses[0])
}

// Test status-get command when HA status is returned.
func TestGetDHCPStatus(t *testing.T) {
	fa := agentcommtest.NewFakeAgents(mockGetStatusLoadBalancing, nil)

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234)

	app := dbmodel.App{
		AccessPoints: accessPoints,
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
	}

	appStatus, err := getDHCPStatus(context.Background(), fa, &app)
	require.NoError(t, err)
	require.NotNil(t, appStatus)

	require.Len(t, appStatus, 1)

	status := appStatus[0]

	// Common fields must be always present.
	require.EqualValues(t, 1234, status.Pid)
	require.EqualValues(t, 3024, status.Uptime)
	require.EqualValues(t, 1111, status.Reload)
	require.Equal(t, "dhcp4", status.Daemon)

	// HA status should have been returned.
	require.NotNil(t, status.HAServers)

	// Test HA status of the server receiving the command.
	local := status.HAServers.Local
	require.Equal(t, "primary", local.Role)
	require.Len(t, local.Scopes, 1)
	require.Contains(t, local.Scopes, "server1")
	require.Equal(t, "load-balancing", local.State)

	// Test HA status of the partner.
	remote := status.HAServers.Remote
	require.Equal(t, "secondary", remote.Role)
	require.Len(t, remote.LastScopes, 1)
	require.Contains(t, remote.LastScopes, "server2")
	require.Equal(t, "load-balancing", remote.LastState)
	require.EqualValues(t, 10, remote.Age)
	require.True(t, remote.InTouch)
}

// Test status-get command when HA status is returned by Kea 1.7.8 or
// later.
func TestGetDHCPStatus178(t *testing.T) {
	fa := agentcommtest.NewFakeAgents(mockGetStatusLoadBalancing178, nil)

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234)

	app := dbmodel.App{
		AccessPoints: accessPoints,
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
	}

	appStatus, err := getDHCPStatus(context.Background(), fa, &app)
	require.NoError(t, err)
	require.NotNil(t, appStatus)

	require.Len(t, appStatus, 1)

	status := appStatus[0]

	// Common fields must be always present.
	require.EqualValues(t, 1234, status.Pid)
	require.EqualValues(t, 3024, status.Uptime)
	require.EqualValues(t, 1111, status.Reload)
	require.Equal(t, "dhcp4", status.Daemon)

	// The HA status should be returned in the high-availability argument.
	require.Nil(t, status.HAServers)

	require.Len(t, status.HA, 1)

	require.Equal(t, "load-balancing", status.HA[0].HAMode)

	// Test HA status of the server receiving the command.
	local := status.HA[0].HAServers.Local
	require.Equal(t, "primary", local.Role)
	require.Len(t, local.Scopes, 1)
	require.Contains(t, local.Scopes, "server1")
	require.Equal(t, "load-balancing", local.State)

	// Test HA status of the partner.
	remote := status.HA[0].HAServers.Remote
	require.Equal(t, "secondary", remote.Role)
	require.Len(t, remote.LastScopes, 1)
	require.Contains(t, remote.LastScopes, "server2")
	require.Equal(t, "load-balancing", remote.LastState)
	require.EqualValues(t, 10, remote.Age)
	require.True(t, remote.InTouch)
	require.NotNil(t, remote.CommInterrupted)
	require.True(t, *remote.CommInterrupted)
	require.EqualValues(t, 1, remote.ConnectingClients)
	require.EqualValues(t, 2, remote.UnackedClients)
	require.EqualValues(t, 3, remote.UnackedClientsLeft)
	require.EqualValues(t, 10, remote.AnalyzedPackets)
}

// Test status-get command when HA status is not returned.
func TestGetDHCPStatusNoHA(t *testing.T) {
	fa := agentcommtest.NewFakeAgents(mockGetStatusNoHA, nil)

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234)

	app := dbmodel.App{
		AccessPoints: accessPoints,
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
	}

	appStatus, err := getDHCPStatus(context.Background(), fa, &app)
	require.NoError(t, err)
	require.NotNil(t, appStatus)

	require.Len(t, appStatus, 1)

	status := appStatus[0]

	// Common fields must be always present.
	require.EqualValues(t, 1234, status.Pid)
	require.EqualValues(t, 3024, status.Uptime)
	require.EqualValues(t, 1111, status.Reload)

	// This time, HA status should not be present.
	require.Nil(t, status.HAServers)
	require.Empty(t, status.HA)
}

// Test the case when the Kea CA is unable to communicate with the
// Kea deamon.
func TestGetDHCPStatusError(t *testing.T) {
	fa := agentcommtest.NewFakeAgents(mockGetStatusError, nil)

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234)

	app := dbmodel.App{
		AccessPoints: accessPoints,
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
	}

	appStatus, err := getDHCPStatus(context.Background(), fa, &app)
	require.NoError(t, err)
	require.NotNil(t, appStatus)

	require.Empty(t, appStatus)
}

// Test that new instance of the puller for fetching HA services status can
// be created and shut down.
func TestNewHAStatusPuller(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// The puller requires fetch interval to be present in the database.
	err := dbmodel.InitializeSettings(db)
	require.NoError(t, err)

	puller, err := NewHAStatusPuller(db, nil)
	require.NoError(t, err)
	require.NotNil(t, puller)
	defer puller.Shutdown()
}

// Test that HA status can be fetched and updated via the HA status puller
// mechanism. This is a generic test which can be used to validate the
// behavior for two different formats of the status-get response, one for
// Kea versions earlier than 1.7.8 and the second for Kea version 1.7.8
// and later.
func testPullHAStatus(t *testing.T, version178 bool) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	// Add a machine.
	m := &dbmodel.Machine{
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, m)
	require.NoError(t, err)

	// Add Kea application to the machine
	var keaPoints []*dbmodel.AccessPoint
	keaPoints = dbmodel.AppendAccessPoint(keaPoints, dbmodel.AccessPointControl, "", "", 1234)
	keaApp := &dbmodel.App{
		ID:           0,
		MachineID:    m.ID,
		Type:         dbmodel.AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
		Daemons: []*dbmodel.Daemon{
			{
				Name: "dhcp4",
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: getHATestConfig("Dhcp4", "server1", "load-balancing",
						"server1", "server2", "server4"),
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
			{
				Name: "dhcp6",
				KeaDaemon: &dbmodel.KeaDaemon{
					Config: getHATestConfig("Dhcp6", "server3", "hot-standby",
						"server1", "server3", "server4"),
					KeaDHCPDaemon: &dbmodel.KeaDHCPDaemon{},
				},
			},
		},
	}

	// This call, apart from adding the app to the machine, will also associate the
	// app with the HA services.
	err = CommitAppIntoDB(db, keaApp, fec, nil)
	require.NoError(t, err)

	// The puller requires fetch interval to be present in the database.
	err = dbmodel.InitializeSettings(db)
	require.NoError(t, err)

	var fa *agentcommtest.FakeAgents

	// Configure the fake control agents to mimic returning a status of
	// two HA services for Kea.
	if version178 {
		fa = agentcommtest.NewFakeAgents(mockGetStatusWithHA178, nil)
	} else {
		fa = agentcommtest.NewFakeAgents(mockGetStatusWithHA, nil)
	}

	// Create the puller which normally fetches the HA status periodically.
	puller, err := NewHAStatusPuller(db, fa)
	require.NoError(t, err)
	require.NotNil(t, puller)

	// No need to wait for the puller to fetch the status.
	count, err := puller.pullData()
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	// We should have two services in the database. One for DHCPv4 and one
	// for DHCPv6.
	services, err := dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)

	// The first one is the DHCPv4 service.
	service := services[0]
	require.NotNil(t, service.HAService)
	// Our app has the primary role in this service.
	require.EqualValues(t, keaApp.Daemons[0].ID, service.HAService.PrimaryID)
	// The status should have been collected for primary and secondary.
	require.False(t, service.HAService.PrimaryStatusCollectedAt.IsZero())
	require.False(t, service.HAService.SecondaryStatusCollectedAt.IsZero())
	// The "age" value indicates how long ago the secondary status have
	// been fetched. Therefore, the timestamp of the seconary status should
	// be earlier than the status of the primary.
	require.True(t, service.HAService.PrimaryStatusCollectedAt.After(service.HAService.SecondaryStatusCollectedAt))
	// Both servers should be in load balancing state.
	require.Equal(t, "load-balancing", service.HAService.PrimaryLastState)
	require.Equal(t, "load-balancing", service.HAService.SecondaryLastState)
	require.ElementsMatch(t, []string{"server1"}, service.HAService.PrimaryLastScopes)
	require.ElementsMatch(t, []string{"server2"}, service.HAService.SecondaryLastScopes)
	require.True(t, service.HAService.PrimaryReachable)
	require.True(t, service.HAService.SecondaryReachable)
	// The failover event hasn't been observed yet.
	require.True(t, service.HAService.PrimaryLastFailoverAt.IsZero())
	require.True(t, service.HAService.SecondaryLastFailoverAt.IsZero())

	// These fields are only available in Kea 1.7.8+.
	if version178 {
		require.NotNil(t, service.HAService.SecondaryCommInterrupted)
		require.True(t, *service.HAService.SecondaryCommInterrupted)
		require.EqualValues(t, 1, service.HAService.SecondaryConnectingClients)
		require.EqualValues(t, 2, service.HAService.SecondaryUnackedClients)
		require.EqualValues(t, 3, service.HAService.SecondaryUnackedClientsLeft)
		require.EqualValues(t, 10, service.HAService.SecondaryAnalyzedPackets)
	}

	// The second service for this app is the DHCPv6 service.
	service = services[1]

	require.NotNil(t, service.HAService)
	require.EqualValues(t, keaApp.Daemons[1].ID, service.HAService.SecondaryID)
	// The status should have been collected for standby and primary.
	require.False(t, service.HAService.PrimaryStatusCollectedAt.IsZero())
	require.False(t, service.HAService.SecondaryStatusCollectedAt.IsZero())
	// The "age" value indicates how long ago the secondary status have
	// been fetched. Therefore, the timestamp of the primary status should
	// be earlier than the status of the primary.
	require.True(t, service.HAService.SecondaryStatusCollectedAt.After(service.HAService.PrimaryStatusCollectedAt))
	require.Equal(t, "waiting", service.HAService.PrimaryLastState)
	require.Equal(t, "hot-standby", service.HAService.SecondaryLastState)
	require.ElementsMatch(t, []string{"server1"}, service.HAService.PrimaryLastScopes)
	require.Empty(t, service.HAService.SecondaryLastScopes)
	require.True(t, service.HAService.PrimaryReachable)
	require.True(t, service.HAService.SecondaryReachable)

	// These fields are only available in Kea 1.7.8+.
	if version178 {
		require.NotNil(t, service.HAService.PrimaryCommInterrupted)
		require.True(t, *service.HAService.PrimaryCommInterrupted)
		require.EqualValues(t, 2, service.HAService.PrimaryConnectingClients)
		require.EqualValues(t, 3, service.HAService.PrimaryUnackedClients)
		require.EqualValues(t, 4, service.HAService.PrimaryUnackedClientsLeft)
		require.EqualValues(t, 15, service.HAService.PrimaryAnalyzedPackets)
	}

	// Pull the data again.
	count, err = puller.pullData()
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	// There should still be two services, one for DHCPv4 and one for DHCPv6.
	services, err = dbmodel.GetDetailedAllServices(db)
	require.NoError(t, err)
	require.Len(t, services, 2)

	// Validate the values of the DHCPv4 service.
	service = services[0]
	require.NotNil(t, service.HAService)
	require.EqualValues(t, keaApp.ID, service.HAService.PrimaryID)
	require.False(t, service.HAService.PrimaryStatusCollectedAt.IsZero())
	require.False(t, service.HAService.SecondaryStatusCollectedAt.IsZero())

	// The primary state is now partner-down and the secondary state is unknown.
	require.Equal(t, "partner-down", service.HAService.PrimaryLastState)
	require.Equal(t, "unavailable", service.HAService.SecondaryLastState)
	require.ElementsMatch(t, []string{"server1", "server2"}, service.HAService.PrimaryLastScopes)
	require.Empty(t, service.HAService.SecondaryLastScopes)
	require.True(t, service.HAService.PrimaryReachable)
	require.False(t, service.HAService.SecondaryReachable)
	// The partner-down state is the indication that the failover took place.
	// This should be recorded for the primary server.
	require.False(t, service.HAService.PrimaryLastFailoverAt.IsZero())
	require.True(t, service.HAService.SecondaryLastFailoverAt.IsZero())

	// These fields are only available in Kea 1.7.8+.
	if version178 {
		require.NotNil(t, service.HAService.SecondaryCommInterrupted)
		require.True(t, *service.HAService.SecondaryCommInterrupted)
		// In the partner-down state they should be all reset.
		require.Zero(t, service.HAService.SecondaryConnectingClients)
		require.Zero(t, service.HAService.SecondaryUnackedClients)
		require.Zero(t, service.HAService.SecondaryUnackedClientsLeft)
		require.Zero(t, service.HAService.SecondaryAnalyzedPackets)
	}

	// The second service for this app is the DHCPv6 service. The status should
	// remain the same for the DHCPv6 server because we were unable to communicate
	// with the server. The state may be overridden if the partner of that server
	// returns a different state for this server.
	service = services[1]
	require.NotNil(t, service.HAService)
	require.EqualValues(t, keaApp.Daemons[1].ID, service.HAService.SecondaryID)
	require.False(t, service.HAService.PrimaryStatusCollectedAt.IsZero())
	require.False(t, service.HAService.SecondaryStatusCollectedAt.IsZero())
	require.True(t, service.HAService.SecondaryStatusCollectedAt.After(service.HAService.PrimaryStatusCollectedAt))
	require.ElementsMatch(t, []string{"server1"}, service.HAService.PrimaryLastScopes)
	require.Empty(t, service.HAService.SecondaryLastScopes)
	require.True(t, service.HAService.PrimaryReachable)
	require.False(t, service.HAService.SecondaryReachable)

	// These fields are only available in Kea 1.7.8+.
	if version178 {
		require.NotNil(t, service.HAService.PrimaryCommInterrupted)
		require.True(t, *service.HAService.PrimaryCommInterrupted)
		require.EqualValues(t, 2, service.HAService.PrimaryConnectingClients)
		require.EqualValues(t, 3, service.HAService.PrimaryUnackedClients)
		require.EqualValues(t, 4, service.HAService.PrimaryUnackedClientsLeft)
		require.EqualValues(t, 15, service.HAService.PrimaryAnalyzedPackets)
	}
}

// Test that HA status can be fetched and updated via the HA status puller
// mechanism.
func TestPullHAStatus(t *testing.T) {
	testPullHAStatus(t, false)
}

// Test that HA status can be fetched and updated via the HA status puller
// mechanism for Kea versions 1.7.8 and later.
func TestPullHAStatus178(t *testing.T) {
	testPullHAStatus(t, true)
}
