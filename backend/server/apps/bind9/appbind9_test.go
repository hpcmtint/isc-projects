package bind9

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test"
)

// Named statistics-channel response.
func mockNamed(callNo int, response interface{}) {
	statsOutput := response.(*NamedStatsGetResponse)
	*statsOutput = NamedStatsGetResponse{
		Views: map[string]*ViewStatsData{
			"_default": {
				Resolver: ResolverData{
					CacheStats: CacheStatsData{
						CacheHits:   40,
						CacheMisses: 10,
						QueryHits:   70,
						QueryMisses: 30,
					},
				},
			},
			"_bind": {
				Resolver: ResolverData{
					CacheStats: CacheStatsData{
						CacheHits:   1,
						CacheMisses: 5,
						QueryHits:   4,
						QueryMisses: 6,
					},
				},
			},
		},
	}
}

// Test retrieving state of BIND 9 app.
func TestGetAppState(t *testing.T) {
	ctx := context.Background()

	fa := agentcommtest.NewFakeAgents(nil, mockNamed)
	fec := &storktest.FakeEventCenter{}

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "127.0.0.1", "abcd", 953)
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointStatistics, "127.0.0.1", "abcd", 8000)
	dbApp := dbmodel.App{
		AccessPoints: accessPoints,
		Machine: &dbmodel.Machine{
			Address:   "192.0.2.0",
			AgentPort: 1111,
		},
	}

	GetAppState(ctx, fa, &dbApp, fec)

	require.Equal(t, "127.0.0.1", fa.RecordedAddress)
	require.EqualValues(t, 953, fa.RecordedPort)
	require.Equal(t, "abcd", fa.RecordedKey)

	require.True(t, dbApp.Active)
	require.Equal(t, dbApp.Meta.Version, "9.9.9")

	require.Len(t, dbApp.Daemons, 1)
	daemon := dbApp.Daemons[0]
	require.NotNil(t, daemon.Bind9Daemon)
	require.True(t, daemon.Active)
	require.Equal(t, "named", daemon.Name)
	require.Equal(t, "9.9.9", daemon.Version)
	reloadedAt, _ := time.Parse(namedLongDateFormat, "Mon, 03 Feb 2020 14:39:36 GMT")
	require.Equal(t, reloadedAt, daemon.ReloadedAt)
	require.EqualValues(t, 5, daemon.Bind9Daemon.Stats.ZoneCount)
	require.EqualValues(t, 96, daemon.Bind9Daemon.Stats.AutomaticZoneCount)

	// Test statistics.
	require.EqualValues(t, 40, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["CacheHits"])
	require.EqualValues(t, 10, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["CacheMisses"])
	require.EqualValues(t, 70, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["QueryHits"])
	require.EqualValues(t, 30, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["QueryMisses"])
}

// Tests that BIND 9 can be added and then updated in the database.
func TestCommitAppIntoDB(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	fec := &storktest.FakeEventCenter{}

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := dbmodel.AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)

	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 1234)
	app := &dbmodel.App{
		ID:           0,
		MachineID:    machine.ID,
		Type:         dbmodel.AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = CommitAppIntoDB(db, app, fec)
	require.NoError(t, err)

	accessPoints = []*dbmodel.AccessPoint{}
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "", "", 2345)
	app.AccessPoints = accessPoints
	err = CommitAppIntoDB(db, app, fec)
	require.NoError(t, err)

	returned, err := dbmodel.GetAppByID(db, app.ID)
	require.NoError(t, err)
	require.NotNil(t, returned)
	require.Len(t, returned.AccessPoints, 1)
	require.EqualValues(t, 2345, returned.AccessPoints[0].Port)
}
