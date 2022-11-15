package restservice

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	keactrl "isc.org/stork/appctrl/kea"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	apps "isc.org/stork/server/apps"
	appstest "isc.org/stork/server/apps/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	testutil "isc.org/stork/testutil"
)

func mockStatusError(commandName string, cmdResponses []interface{}) {
	command := keactrl.NewCommand(commandName, []string{"dhcp4"}, nil)
	json := `[
        {
            "result": 1,
            "text": "unable to communicate with the daemon"
        }
    ]`
	_ = keactrl.UnmarshalResponseList(command, []byte(json), cmdResponses[0])
}

// Test that all hosts can be fetched without filtering.
func TestGetHostsNoFiltering(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)
	ctx := context.Background()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts, apps := testutil.AddTestHosts(t, db)

	err = dbmodel.AddDaemonToHost(db, &hosts[0], apps[0].Daemons[0].ID, dbmodel.HostDataSourceConfig)
	require.NoError(t, err)
	err = dbmodel.AddDaemonToHost(db, &hosts[0], apps[1].Daemons[0].ID, dbmodel.HostDataSourceConfig)
	require.NoError(t, err)

	params := dhcp.GetHostsParams{}
	rsp := rapi.GetHosts(ctx, params)
	require.IsType(t, &dhcp.GetHostsOK{}, rsp)
	okRsp := rsp.(*dhcp.GetHostsOK)
	require.Len(t, okRsp.Payload.Items, 5)
	require.EqualValues(t, 5, okRsp.Payload.Total)

	items := okRsp.Payload.Items
	require.NotNil(t, items)

	// There should be a total of 5 hosts, 4 of them including IP address
	// reservations and 1 with a prefix reservation.
	require.Len(t, items, 5)
	for i := range items {
		require.NotNil(t, items[i])
		require.NotZero(t, items[i].ID)
		require.EqualValues(t, hosts[i].SubnetID, items[i].SubnetID)

		// Check that the host identifiers types match.
		require.EqualValues(t, len(items[i].HostIdentifiers), len(hosts[i].HostIdentifiers))
		for j := range items[i].HostIdentifiers {
			require.NotNil(t, items[i].HostIdentifiers[j])
			require.EqualValues(t, hosts[i].HostIdentifiers[j].Type, items[i].HostIdentifiers[j].IDType)
		}

		// The total number of reservations, which includes both address and
		// prefix reservations should be equal to the number of reservations for
		// a given host.
		require.EqualValues(t, len(hosts[i].IPReservations),
			len(items[i].AddressReservations)+len(items[i].PrefixReservations))

		// Walk over the address and prefix reservations for a host.
		for _, ips := range [][]*models.IPReservation{items[i].AddressReservations, items[i].PrefixReservations} {
			for j, resrv := range ips {
				require.NotNil(t, resrv)
				require.EqualValues(t, hosts[i].IPReservations[j].Address, resrv.Address)
			}
		}
	}

	// The identifiers should have been converted to hex values.
	require.EqualValues(t, "01:02:03:04:05:06", items[0].HostIdentifiers[0].IDHexValue)
	require.EqualValues(t, "01:02:03:04", items[0].HostIdentifiers[1].IDHexValue)
	require.EqualValues(t, "02:03:04:05:06:07", items[1].HostIdentifiers[0].IDHexValue)
	require.EqualValues(t, "02:03:04:05", items[1].HostIdentifiers[1].IDHexValue)
	require.EqualValues(t, "01:02:03:04:05:06", items[2].HostIdentifiers[0].IDHexValue)
	require.EqualValues(t, "01:02:03:04", items[3].HostIdentifiers[0].IDHexValue)
	require.EqualValues(t, "02:02:02:02", items[4].HostIdentifiers[0].IDHexValue)

	require.Equal(t, "192.0.2.0/24", items[0].SubnetPrefix)
	require.Empty(t, items[1].SubnetPrefix)
	require.NotNil(t, "2001:db8:1::/64", items[2].SubnetPrefix)
	require.Empty(t, items[3].SubnetPrefix)
	require.Empty(t, items[4].SubnetPrefix)

	// Hosts
	require.Equal(t, "first.example.org", items[0].Hostname)

	// The first host should be associated with two apps.
	require.Len(t, items[0].LocalHosts, 2)
	require.NotNil(t, items[0].LocalHosts[0])
	require.EqualValues(t, apps[0].ID, items[0].LocalHosts[0].AppID)
	require.Equal(t, dbmodel.HostDataSourceConfig.String(), items[0].LocalHosts[0].DataSource)
	require.Equal(t, "dhcp-server0", items[0].LocalHosts[0].AppName)
	require.NotNil(t, items[0].LocalHosts[1])
	require.EqualValues(t, apps[1].ID, items[0].LocalHosts[1].AppID)
	require.Equal(t, dbmodel.HostDataSourceConfig.String(), items[0].LocalHosts[1].DataSource)
	require.Equal(t, "dhcp-server1", items[0].LocalHosts[1].AppName)
}

// Test that hosts can be filtered by subnet ID.
func TestGetHostsBySubnetID(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)
	ctx := context.Background()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	_, _ = testutil.AddTestHosts(t, db)

	subnetID := int64(2)
	params := dhcp.GetHostsParams{
		SubnetID: &subnetID,
	}
	rsp := rapi.GetHosts(ctx, params)
	require.IsType(t, &dhcp.GetHostsOK{}, rsp)
	okRsp := rsp.(*dhcp.GetHostsOK)
	require.Len(t, okRsp.Payload.Items, 1)
	require.EqualValues(t, 1, okRsp.Payload.Total)
}

// Test that hosts can be filtered by text.
func TestGetHostsWithFiltering(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)
	ctx := context.Background()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	_, _ = testutil.AddTestHosts(t, db)

	filteringText := "2001:db"
	params := dhcp.GetHostsParams{
		Text: &filteringText,
	}
	rsp := rapi.GetHosts(ctx, params)
	require.IsType(t, &dhcp.GetHostsOK{}, rsp)
	okRsp := rsp.(*dhcp.GetHostsOK)
	require.Len(t, okRsp.Payload.Items, 2)
	require.EqualValues(t, 2, okRsp.Payload.Total)
}

// Test that host can be fetched by ID over the REST API.
func TestGetHost(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)
	ctx := context.Background()

	// Add four hosts. Two with IPv4 and two with IPv6 reservations.
	hosts, _ := testutil.AddTestHosts(t, db)

	params := dhcp.GetHostParams{
		ID: hosts[0].ID,
	}
	rsp := rapi.GetHost(ctx, params)
	require.IsType(t, &dhcp.GetHostOK{}, rsp)
	okRsp := rsp.(*dhcp.GetHostOK)
	returnedHost := okRsp.Payload
	require.EqualValues(t, hosts[0].ID, returnedHost.ID)
	require.EqualValues(t, hosts[0].SubnetID, returnedHost.SubnetID)
	require.Equal(t, hosts[0].Hostname, returnedHost.Hostname)

	// Get host for non-existing ID should return a default response.
	params = dhcp.GetHostParams{
		ID: 100000000,
	}
	rsp = rapi.GetHost(ctx, params)
	require.IsType(t, &dhcp.GetHostDefault{}, rsp)
}

// Test that fetched host includes DHCP options.
func TestGetHostWithOptions(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	rapi, err := NewRestAPI(dbSettings, db, dbmodel.NewDHCPOptionDefinitionLookup())
	require.NoError(t, err)
	ctx := context.Background()

	// Add hosts.
	hosts, _ := testutil.AddTestHosts(t, db)

	// Add LocalHost instances comprising DHCP options.
	err = dbmodel.AddHostLocalHosts(db, &hosts[4])
	require.NoError(t, err)

	params := dhcp.GetHostParams{
		ID: hosts[4].ID,
	}
	rsp := rapi.GetHost(ctx, params)
	require.IsType(t, &dhcp.GetHostOK{}, rsp)
	okRsp := rsp.(*dhcp.GetHostOK)
	returnedHost := okRsp.Payload
	require.EqualValues(t, hosts[4].ID, returnedHost.ID)
	require.EqualValues(t, hosts[4].SubnetID, returnedHost.SubnetID)
	require.Equal(t, hosts[4].Hostname, returnedHost.Hostname)

	// Validate returned DHCP options and their hashes.
	require.Len(t, returnedHost.LocalHosts, 2)
	hashes := []string{}
	for _, lh := range returnedHost.LocalHosts {
		require.Len(t, lh.Options, 1)
		require.False(t, lh.Options[0].AlwaysSend)
		require.EqualValues(t, 23, lh.Options[0].Code)
		require.Empty(t, lh.Options[0].Encapsulate)
		require.Len(t, lh.Options[0].Fields, 2)
		require.EqualValues(t, 6, lh.Options[0].Universe)
		hashes = append(hashes, lh.OptionsHash)
	}
	require.NotEmpty(t, hashes[0])
	require.Equal(t, hashes[0], hashes[1])
}

// Test the calls for creating transaction and submitting a new host
// reservation.
func TestCreateHostBeginSubmit(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create fake agents receiving reservation-add commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Make sure we have some Kea apps in the database.
	_, apps := testutil.AddTestHosts(t, db)

	// Begin transaction.
	params := dhcp.CreateHostBeginParams{}
	rsp := rapi.CreateHostBegin(ctx, params)
	require.IsType(t, &dhcp.CreateHostBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.CreateHostBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, daemons and subnets.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.Len(t, contents.Daemons, 4)
	require.Len(t, contents.Subnets, 2)

	// Submit transaction.
	params2 := dhcp.CreateHostSubmitParams{
		ID: transactionID,
		Host: &models.Host{
			SubnetID: 1,
			Hostname: "example.org",
			HostIdentifiers: []*models.HostIdentifier{
				{
					IDType:     "hw-address",
					IDHexValue: "010203040506",
				},
			},
			LocalHosts: []*models.LocalHost{
				{
					DaemonID:   apps[0].Daemons[0].ID,
					DataSource: dbmodel.HostDataSourceAPI.String(),
				},
				{
					DaemonID:   apps[1].Daemons[0].ID,
					DataSource: dbmodel.HostDataSourceAPI.String(),
				},
			},
		},
	}
	rsp2 := rapi.CreateHostSubmit(ctx, params2)
	require.IsType(t, &dhcp.CreateHostSubmitOK{}, rsp2)

	// It should result in sending commands to two Kea servers.
	require.Len(t, fa.RecordedCommands, 2)

	for _, c := range fa.RecordedCommands {
		require.JSONEq(t, `{
            "command": "reservation-add",
            "service": ["dhcp4"],
            "arguments": {
                "reservation": {
                    "hw-address": "010203040506",
                    "subnet-id": 111,
                    "hostname": "example.org"
                }
            }
        }`, c.Marshal())
	}

	// Make sure that the transaction is done.
	cctx, _ := cm.RecoverContext(transactionID, int64(user.ID))
	// Remove the context from the config manager before testing that
	// the returned context is nil. If it happens to be non-nil the
	// require.Nil() would otherwise spit out errors about the concurrent
	// access to the context in the manager's goroutine and here.
	if cctx != nil {
		cm.Done(cctx)
	}
	require.Nil(t, cctx)
}

// Test error case when a user attempts to begin new transaction when the
// user has no session.
func TestCreateHostBeginNoSession(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create fake agents receiving reservation-add commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager but do not login the user.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Make sure we have some Kea apps in the database.
	_, _ = testutil.AddTestHosts(t, db)

	// Begin transaction.
	params := dhcp.CreateHostBeginParams{}
	rsp := rapi.CreateHostBegin(ctx, params)
	require.IsType(t, &dhcp.CreateHostBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.CreateHostBeginDefault)
	require.Equal(t, http.StatusForbidden, getStatusCode(*defaultRsp))
}

// Test error case when a user attempts to begin a new transaction when
// there are no servers with host_cmds hook library found.
func TestCreateHostBeginNoServers(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create fake agents receiving reservation-add commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager but do not login the user.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.CreateHostBeginParams{}
	rsp := rapi.CreateHostBegin(ctx, params)
	require.IsType(t, &dhcp.CreateHostBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.CreateHostBeginDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
}

// Test error cases for submitting new host reservation.
func TestCreateHostSubmitError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Setup fake agents that return an error in response to reservation-add
	// command.
	fa := agentcommtest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		mockStatusError("reservation-add", cmdResponses)
	}, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Start session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create a user and simulate logging in.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Make sure we have some Kea apps in the database.
	_, apps := testutil.AddTestHosts(t, db)

	// Begin transaction. It will be needed for the actual part of the
	// test that relies on the existence of the transaction.
	params := dhcp.CreateHostBeginParams{}
	rsp := rapi.CreateHostBegin(ctx, params)
	require.IsType(t, &dhcp.CreateHostBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.CreateHostBeginOK)
	contents := okRsp.Payload

	// Capture transaction ID.
	transactionID := contents.ID
	require.NotZero(t, transactionID)

	// Submit transaction without the host information.
	t.Run("no host", func(t *testing.T) {
		params := dhcp.CreateHostSubmitParams{
			ID:   transactionID,
			Host: nil,
		}
		rsp := rapi.CreateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateHostSubmitDefault)
		require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	})

	// Submit transaction with non-matching transaction ID.
	t.Run("wrong transaction id", func(t *testing.T) {
		params := dhcp.CreateHostSubmitParams{
			ID: transactionID + 1,
			Host: &models.Host{
				LocalHosts: []*models.LocalHost{
					{
						DaemonID: apps[0].Daemons[0].ID,
					},
					{
						DaemonID: apps[1].Daemons[0].ID,
					},
				},
			},
		}
		rsp := rapi.CreateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateHostSubmitDefault)
		require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	})

	// Submit transaction with a host that is not associated with any daemons.
	// It simulates a failure in "apply" step which typically is caused by
	// some internal server problem rather than malformed request.
	t.Run("no daemons in host", func(t *testing.T) {
		params := dhcp.CreateHostSubmitParams{
			ID: transactionID,
			Host: &models.Host{
				LocalHosts: []*models.LocalHost{},
			},
		}
		rsp := rapi.CreateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateHostSubmitDefault)
		require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	})

	// Submit transaction with valid ID and host but expect the agent to
	// return an error code. This is considered a conflict with the state
	// of the Kea servers.
	t.Run("commit failure", func(t *testing.T) {
		params := dhcp.CreateHostSubmitParams{
			ID: transactionID,
			Host: &models.Host{
				LocalHosts: []*models.LocalHost{
					{
						DaemonID:   apps[0].Daemons[0].ID,
						DataSource: dbmodel.HostDataSourceAPI.String(),
					},
				},
			},
		}
		rsp := rapi.CreateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateHostSubmitDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
	})

	// Submit transaction with valid ID and host but the user has no
	// session.
	t.Run("no user session", func(t *testing.T) {
		err = rapi.SessionManager.LogoutHandler(ctx)
		require.NoError(t, err)

		params := dhcp.CreateHostSubmitParams{
			ID: transactionID,
			Host: &models.Host{
				LocalHosts: []*models.LocalHost{
					{
						DaemonID:   apps[0].Daemons[0].ID,
						DataSource: dbmodel.HostDataSourceAPI.String(),
					},
				},
			},
		}
		rsp := rapi.CreateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.CreateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateHostSubmitDefault)
		require.Equal(t, http.StatusForbidden, getStatusCode(*defaultRsp))
	})
}

// Test that the transaction to add a new host can be canceled, resulting
// in the removal of this transaction from the config manager.
func TestCreateHostBeginCancel(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create fake agents receiving reservation-add commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Make sure we have some Kea apps in the database.
	_, _ = testutil.AddTestHosts(t, db)

	// Begin transaction.
	params := dhcp.CreateHostBeginParams{}
	rsp := rapi.CreateHostBegin(ctx, params)
	require.IsType(t, &dhcp.CreateHostBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.CreateHostBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, daemons and subnets.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.Len(t, contents.Daemons, 4)
	require.Len(t, contents.Subnets, 2)

	// Cancel the transaction.
	params2 := dhcp.CreateHostDeleteParams{
		ID: transactionID,
	}
	rsp2 := rapi.CreateHostDelete(ctx, params2)
	require.IsType(t, &dhcp.CreateHostDeleteOK{}, rsp2)

	cctx, _ := cm.RecoverContext(transactionID, int64(user.ID))
	// Remove the context from the config manager before testing that
	// the returned context is nil. If it happens to be non-nil the
	// require.Nil() would otherwise spit out errors about the concurrent
	// access to the context in the manager's goroutine and here.
	if cctx != nil {
		cm.Done(cctx)
	}
	require.Nil(t, cctx)
}

// Test error cases for cancelling new host reservation.
func TestCreateHostDeleteError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Setup fake agents that return an error in response to reservation-add
	// command.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Start session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create a user and simulate logging in.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Make sure we have some Kea apps in the database.
	_, _ = testutil.AddTestHosts(t, db)

	// Begin transaction. It will be needed for the actual part of the
	// test that relies on the existence of the transaction.
	params := dhcp.CreateHostBeginParams{}
	rsp := rapi.CreateHostBegin(ctx, params)
	require.IsType(t, &dhcp.CreateHostBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.CreateHostBeginOK)
	contents := okRsp.Payload

	// Capture transaction ID.
	transactionID := contents.ID
	require.NotZero(t, transactionID)

	// Cancel transaction with non-matching transaction ID.
	t.Run("wrong transaction id", func(t *testing.T) {
		params := dhcp.CreateHostDeleteParams{
			ID: transactionID + 1,
		}
		rsp := rapi.CreateHostDelete(ctx, params)
		require.IsType(t, &dhcp.CreateHostDeleteDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateHostDeleteDefault)
		require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	})

	// Cancel transaction with valid ID and host but the user has no
	// session.
	t.Run("no user session", func(t *testing.T) {
		err = rapi.SessionManager.LogoutHandler(ctx)
		require.NoError(t, err)

		params := dhcp.CreateHostDeleteParams{
			ID: transactionID,
		}
		rsp := rapi.CreateHostDelete(ctx, params)
		require.IsType(t, &dhcp.CreateHostDeleteDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.CreateHostDeleteDefault)
		require.Equal(t, http.StatusForbidden, getStatusCode(*defaultRsp))
	})
}

// Test the calls for creating new transaction and updating a host reservation.
func TestUpdateHostBeginSubmit(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Make sure we have some Kea apps in the database.
	hosts, apps := testutil.AddTestHosts(t, db)
	err = dbmodel.AddDaemonToHost(db, &hosts[0], apps[0].Daemons[0].ID, dbmodel.HostDataSourceAPI)
	require.NoError(t, err)
	err = dbmodel.AddDaemonToHost(db, &hosts[0], apps[1].Daemons[0].ID, dbmodel.HostDataSourceAPI)
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.UpdateHostBeginParams{
		HostID: hosts[0].ID,
	}
	rsp := rapi.UpdateHostBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateHostBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateHostBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, host, daemons and subnets.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.NotNil(t, contents.Host)
	require.Len(t, contents.Daemons, 4)
	require.Len(t, contents.Subnets, 2)

	// Submit transaction.
	params2 := dhcp.UpdateHostSubmitParams{
		ID: transactionID,
		Host: &models.Host{
			SubnetID: 1,
			Hostname: "updated.example.org",
			HostIdentifiers: []*models.HostIdentifier{
				{
					IDType:     "hw-address",
					IDHexValue: "010203040506",
				},
			},
			LocalHosts: []*models.LocalHost{
				{
					DaemonID:   apps[0].Daemons[0].ID,
					DataSource: dbmodel.HostDataSourceAPI.String(),
				},
				{
					DaemonID:   apps[1].Daemons[0].ID,
					DataSource: dbmodel.HostDataSourceAPI.String(),
				},
			},
		},
	}
	rsp2 := rapi.UpdateHostSubmit(ctx, params2)
	require.IsType(t, &dhcp.UpdateHostSubmitOK{}, rsp2)

	// It should result in sending commands to two Kea servers. Each server
	// receives reservation-del and reservation-add commands.
	require.Len(t, fa.RecordedCommands, 4)

	for i, c := range fa.RecordedCommands {
		switch {
		case i < 2:
			require.JSONEq(t, `{
            "command": "reservation-del",
            "service": ["dhcp4"],
            "arguments": {
                "identifier": "010203040506",
                "identifier-type": "hw-address",
                "subnet-id": 111
            }
        }`, c.Marshal())

		default:
			require.JSONEq(t, `{
                "command": "reservation-add",
                "service": ["dhcp4"],
                "arguments": {
                    "reservation": {
                        "hw-address": "010203040506",
                        "subnet-id": 111,
                        "hostname": "updated.example.org"
                    }
                }
             }`, c.Marshal())
		}
	}

	// Make sure that the transaction is done.
	cctx, _ := cm.RecoverContext(transactionID, int64(user.ID))
	// Remove the context from the config manager before testing that
	// the returned context is nil. If it happens to be non-nil the
	// require.Nil() would otherwise spit out errors about the concurrent
	// access to the context in the manager's goroutine and here.
	if cctx != nil {
		cm.Done(cctx)
	}
	require.Nil(t, cctx)
}

// Test error case when a user attempts to begin new transaction when the
// user has no session.
func TestUpdateHostBeginNoSession(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create fake agents receiving reservation-add commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager but do not login the user.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Make sure we have some Kea apps in the database.
	hosts, _ := testutil.AddTestHosts(t, db)

	// Begin transaction.
	params := dhcp.UpdateHostBeginParams{
		HostID: hosts[0].ID,
	}
	rsp := rapi.UpdateHostBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateHostBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.UpdateHostBeginDefault)
	require.Equal(t, http.StatusForbidden, getStatusCode(*defaultRsp))
}

// Test that an error is returned when it is attempted to begin new
// transaction for updating non-existing host reservation.
func TestUpdateHostBeginNonExistingHostID(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Make sure there are some daemons in the database.
	_, _ = testutil.AddTestHosts(t, db)

	// Begin transaction for non-existing host.
	params := dhcp.UpdateHostBeginParams{
		HostID: int64(1024),
	}
	rsp := rapi.UpdateHostBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateHostBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.UpdateHostBeginDefault)
	require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
}

// Test error cases for submitting host reservation update.
func TestUpdateHostSubmitError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Setup fake agents that return an error in response to reservation-del
	// command.
	fa := agentcommtest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		mockStatusError("reservation-del", cmdResponses)
	}, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Start session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create a user and simulate logging in.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Make sure we have some Kea apps an hosts in the database.
	hosts, apps := testutil.AddTestHosts(t, db)

	// Begin transaction. It will be needed for the actual part of the
	// test that relies on the existence of the transaction.
	params := dhcp.UpdateHostBeginParams{
		HostID: hosts[0].ID,
	}
	rsp := rapi.UpdateHostBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateHostBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateHostBeginOK)
	contents := okRsp.Payload

	// Capture transaction ID.
	transactionID := contents.ID
	require.NotZero(t, transactionID)

	// Submit transaction without the host information.
	t.Run("no host", func(t *testing.T) {
		params := dhcp.UpdateHostSubmitParams{
			ID:   transactionID,
			Host: nil,
		}
		rsp := rapi.UpdateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateHostSubmitDefault)
		require.Equal(t, http.StatusBadRequest, getStatusCode(*defaultRsp))
	})

	// Submit transaction with non-matching transaction ID.
	t.Run("wrong transaction id", func(t *testing.T) {
		params := dhcp.UpdateHostSubmitParams{
			ID: transactionID + 1,
			Host: &models.Host{
				HostIdentifiers: []*models.HostIdentifier{
					{
						IDType:     "hw-address",
						IDHexValue: "010203040506",
					},
				},
				LocalHosts: []*models.LocalHost{
					{
						DaemonID: apps[0].Daemons[0].ID,
					},
					{
						DaemonID: apps[1].Daemons[0].ID,
					},
				},
			},
		}
		rsp := rapi.UpdateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateHostSubmitDefault)
		require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	})

	// Submit transaction with a host that is not associated with any daemons.
	// It simulates a failure in "apply" step which typically is caused by
	// some internal server problem rather than malformed request.
	t.Run("no daemons in host", func(t *testing.T) {
		params := dhcp.UpdateHostSubmitParams{
			ID: transactionID,
			Host: &models.Host{
				HostIdentifiers: []*models.HostIdentifier{
					{
						IDType:     "hw-address",
						IDHexValue: "010203040506",
					},
				},
				LocalHosts: []*models.LocalHost{},
			},
		}
		rsp := rapi.UpdateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateHostSubmitDefault)
		require.Equal(t, http.StatusInternalServerError, getStatusCode(*defaultRsp))
	})

	// Submit transaction with valid ID and host but expect the agent to
	// return an error code. This is considered a conflict with the state
	// of the Kea servers.
	t.Run("commit failure", func(t *testing.T) {
		params := dhcp.UpdateHostSubmitParams{
			ID: transactionID,
			Host: &models.Host{
				HostIdentifiers: []*models.HostIdentifier{
					{
						IDType:     "hw-address",
						IDHexValue: "010203040506",
					},
				},
				LocalHosts: []*models.LocalHost{
					{
						DaemonID:   apps[0].Daemons[0].ID,
						DataSource: dbmodel.HostDataSourceAPI.String(),
					},
				},
			},
		}
		rsp := rapi.UpdateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateHostSubmitDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
	})

	// Submit transaction with valid ID and host but the user has no
	// session.
	t.Run("no user session", func(t *testing.T) {
		err = rapi.SessionManager.LogoutHandler(ctx)
		require.NoError(t, err)

		params := dhcp.UpdateHostSubmitParams{
			ID: transactionID,
			Host: &models.Host{
				HostIdentifiers: []*models.HostIdentifier{
					{
						IDType:     "hw-address",
						IDHexValue: "010203040506",
					},
				},
				LocalHosts: []*models.LocalHost{
					{
						DaemonID:   apps[0].Daemons[0].ID,
						DataSource: dbmodel.HostDataSourceAPI.String(),
					},
				},
			},
		}
		rsp := rapi.UpdateHostSubmit(ctx, params)
		require.IsType(t, &dhcp.UpdateHostSubmitDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.UpdateHostSubmitDefault)
		require.Equal(t, http.StatusForbidden, getStatusCode(*defaultRsp))
	})
}

// Test that the transaction to update a host can be canceled, resulting
// in the removal of this transaction from the config manager and allowing
// another user to apply config updates.
func TestUpdateHostBeginCancel(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create fake agents receiving commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Make sure we have some Kea apps in the database.
	hosts, _ := testutil.AddTestHosts(t, db)
	err = dbmodel.AddHostLocalHosts(db, &hosts[0])
	require.NoError(t, err)

	// Begin transaction.
	params := dhcp.UpdateHostBeginParams{
		HostID: hosts[0].ID,
	}
	rsp := rapi.UpdateHostBegin(ctx, params)
	require.IsType(t, &dhcp.UpdateHostBeginOK{}, rsp)
	okRsp := rsp.(*dhcp.UpdateHostBeginOK)
	contents := okRsp.Payload

	// Make sure the server returned transaction ID, daemons and subnets.
	transactionID := contents.ID
	require.NotZero(t, transactionID)
	require.Len(t, contents.Daemons, 4)
	require.Len(t, contents.Subnets, 2)

	// Try to start another session by another user.
	ctx2, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user = &dbmodel.SystemUser{
		ID: 2345,
	}
	err = rapi.SessionManager.LoginHandler(ctx2, user)
	require.NoError(t, err)

	// It should fail because the first session locked the daemons for
	// update.
	rsp = rapi.UpdateHostBegin(ctx2, params)
	require.IsType(t, &dhcp.UpdateHostBeginDefault{}, rsp)
	defaultRsp := rsp.(*dhcp.UpdateHostBeginDefault)
	require.Equal(t, http.StatusLocked, getStatusCode(*defaultRsp))

	// Cancel the transaction.
	params2 := dhcp.UpdateHostDeleteParams{
		ID: transactionID,
	}
	rsp2 := rapi.UpdateHostDelete(ctx, params2)
	require.IsType(t, &dhcp.UpdateHostDeleteOK{}, rsp2)

	cctx, _ := cm.RecoverContext(transactionID, int64(user.ID))
	// Remove the context from the config manager before testing that
	// the returned context is nil. If it happens to be non-nil the
	// require.Nil() would otherwise spit out errors about the concurrent
	// access to the context in the manager's goroutine and here.
	if cctx != nil {
		cm.Done(cctx)
	}
	require.Nil(t, cctx)

	// After we released the lock, another user should be able to apply
	// his changes.
	rsp = rapi.UpdateHostBegin(ctx2, params)
	require.IsType(t, &dhcp.UpdateHostBeginOK{}, rsp)
}

// Test successfully deleting host reservation.
func TestDeleteHost(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Create fake agents receiving reservation-del commands.
	fa := agentcommtest.NewFakeAgents(nil, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create the config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Create session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create user session.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Add test hosts and associate them with the daemons.
	hosts, apps := testutil.AddTestHosts(t, db)
	err = dbmodel.AddDaemonToHost(db, &hosts[0], apps[0].Daemons[0].ID, dbmodel.HostDataSourceAPI)
	require.NoError(t, err)
	err = dbmodel.AddDaemonToHost(db, &hosts[0], apps[1].Daemons[0].ID, dbmodel.HostDataSourceAPI)
	require.NoError(t, err)

	// Attempt to delete the first host.
	params := dhcp.DeleteHostParams{
		ID: hosts[0].ID,
	}
	rsp := rapi.DeleteHost(ctx, params)
	require.IsType(t, &dhcp.DeleteHostOK{}, rsp)

	// The reservation-del commands should be sent to two Kea servers.
	require.Len(t, fa.RecordedCommands, 2)

	for _, c := range fa.RecordedCommands {
		require.JSONEq(t, `{
            "command": "reservation-del",
            "service": ["dhcp4"],
            "arguments": {
                "identifier": "010203040506",
                "identifier-type": "hw-address",
                "subnet-id": 111
            }
        }`, c.Marshal())
	}

	returnedHost, err := dbmodel.GetHost(db, hosts[0].ID)
	require.NoError(t, err)
	require.Nil(t, returnedHost)
}

// Test error cases for deleting a host reservation.
func TestDeleteHostError(t *testing.T) {
	db, dbSettings, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Setup fake agents that return an error in response to reservation-del
	// command.
	fa := agentcommtest.NewFakeAgents(func(callNo int, cmdResponses []interface{}) {
		mockStatusError("reservation-del", cmdResponses)
	}, nil)
	require.NotNil(t, fa)

	lookup := dbmodel.NewDHCPOptionDefinitionLookup()
	require.NotNil(t, lookup)

	// Create config manager.
	cm := apps.NewManager(&appstest.ManagerAccessorsWrapper{
		DB:        db,
		Agents:    fa,
		DefLookup: lookup,
	})
	require.NotNil(t, cm)

	// Create API.
	rapi, err := NewRestAPI(dbSettings, db, fa, cm, lookup)
	require.NoError(t, err)

	// Start session manager.
	ctx, err := rapi.SessionManager.Load(context.Background(), "")
	require.NoError(t, err)

	// Create a user and simulate logging in.
	user := &dbmodel.SystemUser{
		ID: 1234,
	}
	err = rapi.SessionManager.LoginHandler(ctx, user)
	require.NoError(t, err)

	// Make sure we have some Kea apps in the database.
	hosts, apps := testutil.AddTestHosts(t, db)

	err = dbmodel.AddDaemonToHost(db, &hosts[0], apps[0].Daemons[0].ID, dbmodel.HostDataSourceAPI)
	require.NoError(t, err)
	err = dbmodel.AddDaemonToHost(db, &hosts[0], apps[1].Daemons[0].ID, dbmodel.HostDataSourceAPI)
	require.NoError(t, err)

	// Submit transaction with non-matching host ID.
	t.Run("wrong host id", func(t *testing.T) {
		params := dhcp.DeleteHostParams{
			ID: 19809865,
		}
		rsp := rapi.DeleteHost(ctx, params)
		require.IsType(t, &dhcp.DeleteHostDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.DeleteHostDefault)
		require.Equal(t, http.StatusNotFound, getStatusCode(*defaultRsp))
	})

	// Submit transaction with valid ID but expect the agent to return an
	// error code. This is considered a conflict with the state of the
	// Kea servers.
	t.Run("commit failure", func(t *testing.T) {
		params := dhcp.DeleteHostParams{
			ID: hosts[0].ID,
		}
		rsp := rapi.DeleteHost(ctx, params)
		require.IsType(t, &dhcp.DeleteHostDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.DeleteHostDefault)
		require.Equal(t, http.StatusConflict, getStatusCode(*defaultRsp))
	})

	// Submit transaction with valid ID but the user has no session.
	t.Run("no user session", func(t *testing.T) {
		err = rapi.SessionManager.LogoutHandler(ctx)
		require.NoError(t, err)

		params := dhcp.DeleteHostParams{
			ID: hosts[0].ID,
		}
		rsp := rapi.DeleteHost(ctx, params)
		require.IsType(t, &dhcp.DeleteHostDefault{}, rsp)
		defaultRsp := rsp.(*dhcp.DeleteHostDefault)
		require.Equal(t, http.StatusForbidden, getStatusCode(*defaultRsp))
	})
}
