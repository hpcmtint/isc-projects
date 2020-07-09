package bind9

import (
	"context"

	"github.com/go-pg/pg/v9"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

type StatsPuller struct {
	*agentcomm.PeriodicPuller
	EventCenter eventcenter.EventCenter
}

// Create a StatsPuller object that in background pulls BIND 9 statistics.
// Beneath it spawns a goroutine that pulls stats periodically from the BIND 9
// statistics-channel.
func NewStatsPuller(db *pg.DB, agents agentcomm.ConnectedAgents, eventCenter eventcenter.EventCenter) (*StatsPuller, error) {
	statsPuller := &StatsPuller{
		EventCenter: eventCenter,
	}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "BIND 9 Stats", "bind9_stats_puller_interval",
		statsPuller.pullStats)
	if err != nil {
		return nil, err
	}
	statsPuller.PeriodicPuller = periodicPuller
	return statsPuller, nil
}

// Shutdown StatsPuller. It stops goroutine that pulls stats.
func (statsPuller *StatsPuller) Shutdown() {
	statsPuller.PeriodicPuller.Shutdown()
}

// Pull stats periodically for all BIND 9 apps which Stork is monitoring.
// The function returns a number of apps for which the stats were successfully
// pulled and last encountered error.
func (statsPuller *StatsPuller) pullStats() (int, error) {
	// get list of all bind9 apps from database
	dbApps, err := dbmodel.GetAppsByType(statsPuller.Db, dbmodel.AppTypeBind9)
	if err != nil {
		return 0, err
	}

	// get stats from each bind9 app
	var lastErr error
	appsOkCnt := 0
	for _, dbApp := range dbApps {
		dbApp2 := dbApp
		err := statsPuller.getStatsFromApp(&dbApp2)
		if err != nil {
			lastErr = err
			log.Errorf("error occurred while getting stats from app %+v: %+v", dbApp, err)
		} else {
			appsOkCnt++
		}
	}
	log.Printf("completed pulling stats from BIND 9 apps: %d/%d succeeded", appsOkCnt, len(dbApps))
	return appsOkCnt, lastErr
}

// Get stats from given bind9 app.
func (statsPuller *StatsPuller) getStatsFromApp(dbApp *dbmodel.App) error {
	// if app or daemon not active then do nothing
	if len(dbApp.Daemons) > 0 && !dbApp.Daemons[0].Active {
		return nil
	}
	// prepare URL to statistics-channel
	statsChannel, err := dbApp.GetAccessPoint(dbmodel.AccessPointStatistics)
	if err != nil {
		return err
	}

	statsOutput := NamedStatsGetResponse{}
	ctx := context.Background()
	err = statsPuller.Agents.ForwardToNamedStats(ctx, dbApp.Machine.Address, dbApp.Machine.AgentPort, statsChannel.Address, statsChannel.Port, "json/v1", &statsOutput)
	if err != nil {
		return err
	}

	namedStats := &dbmodel.Bind9NamedStats{}

	if statsOutput.Views != nil {
		viewStats := make(map[string]*dbmodel.Bind9StatsView)

		for name, view := range statsOutput.Views {
			// Only deal with the default view for now.
			if name != "_default" {
				continue
			}

			cacheStats := make(map[string]int64)
			cacheStats["CacheHits"] = view.Resolver.CacheStats.CacheHits
			cacheStats["CacheMisses"] = view.Resolver.CacheStats.CacheMisses
			cacheStats["QueryHits"] = view.Resolver.CacheStats.QueryHits
			cacheStats["QueryMisses"] = view.Resolver.CacheStats.QueryMisses

			viewStats[name] = &dbmodel.Bind9StatsView{
				Resolver: &dbmodel.Bind9StatsResolver{
					CacheStats: cacheStats,
				},
			}
			break
		}

		namedStats.Views = viewStats
	}

	dbApp.Daemons[0].Bind9Daemon.Stats.NamedStats = namedStats
	return dbmodel.UpdateDaemon(statsPuller.Db, dbApp.Daemons[0])
}
