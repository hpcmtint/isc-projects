package bind9

import (
	"testing"

	"github.com/stretchr/testify/require"

	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	storktest "isc.org/stork/server/test"
)

// Check creating and shutting down StatsPuller.
func TestStatsPullerBasic(t *testing.T) {
	// prepare db
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "bind9_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	err := db.Insert(&setting)
	require.NoError(t, err)

	// prepare fake agents and eventcenter
	fa := agentcommtest.NewFakeAgents(nil, nil)
	fec := &storktest.FakeEventCenter{}

	sp, _ := NewStatsPuller(db, fa, fec)
	sp.Shutdown()
}

// Check if pulling stats works.
func TestStatsPullerPullStats(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	bind9Mock := func(callNo int, statsOutput interface{}) {
		json := `{
		    "json-stats-version":"1.2",
		    "views":{
		        "_default":{
		            "resolver":{
		                "cachestats":{
		                    "CacheHits": 60,
		                    "CacheMisses": 40,
		                    "QueryHits": 10,
		                    "QueryMisses": 90
		                }
		            }
		        },
		        "_bind":{
		            "resolver":{
		                "cachestats":{
		                    "CacheHits": 30,
		                    "CacheMisses": 70,
		                    "QueryHits": 20,
		                    "QueryMisses": 80
		                }
		            }
		        }
		    }
		}`

		agentcomm.UnmarshalNamedStatsResponse(json, statsOutput)
	}
	fa := agentcommtest.NewFakeAgents(nil, bind9Mock)
	fec := &storktest.FakeEventCenter{}

	// prepare bind9 apps
	var err error
	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "127.0.0.1", "abcd", 953)
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointStatistics, "127.0.0.1", "abcd", 8000)

	daemon := &dbmodel.Daemon{
		Pid:         0,
		Name:        "named",
		Active:      true,
		Bind9Daemon: &dbmodel.Bind9Daemon{},
	}

	machine1 := &dbmodel.Machine{
		ID:        0,
		Address:   "192.0.1.0",
		AgentPort: 1111,
	}
	err = dbmodel.AddMachine(db, machine1)
	require.NoError(t, err)
	require.NotZero(t, machine1.ID)
	dbApp1 := dbmodel.App{
		Type:         dbmodel.AppTypeBind9,
		AccessPoints: accessPoints,
		MachineID:    machine1.ID,
		Daemons: []*dbmodel.Daemon{
			daemon,
		},
	}
	err = CommitAppIntoDB(db, &dbApp1, fec)
	require.NoError(t, err)

	daemon.ID = 0
	daemon.Bind9Daemon.ID = 0
	machine2 := &dbmodel.Machine{
		ID:        0,
		Address:   "192.0.2.0",
		AgentPort: 2222,
	}
	err = dbmodel.AddMachine(db, machine2)
	require.NoError(t, err)
	require.NotZero(t, machine2.ID)
	dbApp2 := dbmodel.App{
		Type:         dbmodel.AppTypeBind9,
		AccessPoints: accessPoints,
		MachineID:    machine2.ID,
		Daemons: []*dbmodel.Daemon{
			daemon,
		},
	}
	err = CommitAppIntoDB(db, &dbApp2, fec)
	require.NoError(t, err)

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "bind9_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	err = db.Insert(&setting)
	require.NoError(t, err)

	// prepare stats puller
	sp, err := NewStatsPuller(db, fa, fec)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullStats()
	require.NoError(t, err)
	require.Equal(t, 2, appsOkCnt)

	// check collected stats
	app1, err := dbmodel.GetAppByID(db, dbApp1.ID)
	require.NoError(t, err)

	require.Len(t, app1.Daemons, 1)
	require.NotNil(t, app1.Daemons[0].Bind9Daemon)
	daemon = app1.Daemons[0]
	require.EqualValues(t, 60, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["CacheHits"])
	require.EqualValues(t, 40, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["CacheMisses"])
	require.EqualValues(t, 10, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["QueryHits"])
	require.EqualValues(t, 90, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["QueryMisses"])

	app2, err := dbmodel.GetAppByID(db, dbApp2.ID)
	require.NoError(t, err)
	require.Len(t, app2.Daemons, 1)
	require.NotNil(t, app2.Daemons[0].Bind9Daemon)
	daemon = app2.Daemons[0]
	require.EqualValues(t, 60, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["CacheHits"])
	require.EqualValues(t, 40, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["CacheMisses"])
	require.EqualValues(t, 10, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["QueryHits"])
	require.EqualValues(t, 90, daemon.Bind9Daemon.Stats.NamedStats.Views["_default"].Resolver.CacheStats["QueryMisses"])
}

// Check if statistics-channel response is handled correctly when it is empty.
func TestStatsPullerEmptyResponse(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// prepare fake agents
	bind9Mock := func(callNo int, statsOutput interface{}) {
		json := `{
                    "json-stats-version":"1.2"
                }`

		agentcomm.UnmarshalNamedStatsResponse(json, statsOutput)
	}
	fa := agentcommtest.NewFakeAgents(nil, bind9Mock)
	fec := &storktest.FakeEventCenter{}

	// prepare bind9 app
	var err error
	var accessPoints []*dbmodel.AccessPoint
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointControl, "127.0.0.1", "abcd", 953)
	accessPoints = dbmodel.AppendAccessPoint(accessPoints, dbmodel.AccessPointStatistics, "127.0.0.1", "abcd", 8000)

	daemon := &dbmodel.Daemon{
		Pid:         0,
		Name:        "named",
		Active:      true,
		Bind9Daemon: &dbmodel.Bind9Daemon{},
	}

	machine := &dbmodel.Machine{
		ID:        0,
		Address:   "192.0.1.0",
		AgentPort: 1111,
	}
	err = dbmodel.AddMachine(db, machine)
	require.NoError(t, err)
	require.NotZero(t, machine.ID)
	dbApp := dbmodel.App{
		Type:         dbmodel.AppTypeBind9,
		AccessPoints: accessPoints,
		MachineID:    machine.ID,
		Daemons: []*dbmodel.Daemon{
			daemon,
		},
	}
	err = CommitAppIntoDB(db, &dbApp, fec)
	require.NoError(t, err)

	// set one setting that is needed by puller
	setting := dbmodel.Setting{
		Name:    "bind9_stats_puller_interval",
		ValType: dbmodel.SettingValTypeInt,
		Value:   "60",
	}
	err = db.Insert(&setting)
	require.NoError(t, err)

	// prepare stats puller
	sp, err := NewStatsPuller(db, fa, fec)
	require.NoError(t, err)
	// shutdown stats puller at the end
	defer sp.Shutdown()

	// invoke pulling stats
	appsOkCnt, err := sp.pullStats()
	require.NoError(t, err)
	require.Equal(t, 1, appsOkCnt)

	// check collected stats
	app1, err := dbmodel.GetAppByID(db, dbApp.ID)
	require.NoError(t, err)
	require.Len(t, app1.Daemons, 1)
	require.NotNil(t, app1.Daemons[0].Bind9Daemon)
	daemon = app1.Daemons[0]
	require.Empty(t, daemon.Bind9Daemon.Stats.NamedStats.Views)
}
