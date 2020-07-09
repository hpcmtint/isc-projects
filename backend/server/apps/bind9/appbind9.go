package bind9

import (
	"context"
	"regexp"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
)

// Provide example date format how named returns dates.
const namedLongDateFormat = "Mon, 02 Jan 2006 15:04:05 MST"

type CacheStatsData struct {
	CacheHits   int64 `json:"CacheHits"`
	CacheMisses int64 `json:"CacheMisses"`
	QueryHits   int64 `json:"QueryHits"`
	QueryMisses int64 `json:"QueryMisses"`
}

type ResolverData struct {
	CacheStats CacheStatsData `json:"cachestats"`
}

type ViewStatsData struct {
	Resolver ResolverData `json:"resolver"`
}

type NamedStatsGetResponse struct {
	Views map[string]*ViewStatsData `json:"views,omitempty"`
}

// Get statistics from named daemon using ForwardToNamedStats function.
func GetAppStatistics(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App) {
	// prepare URL to named
	statsChannel, err := dbApp.GetAccessPoint(dbmodel.AccessPointStatistics)
	if err != nil {
		log.Warnf("problem with getting named statistics-channel access point: %s", err)
		return
	}

	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// store all collected details in app db record
	statsOutput := NamedStatsGetResponse{}
	err = agents.ForwardToNamedStats(ctx2, dbApp.Machine.Address, dbApp.Machine.AgentPort, statsChannel.Address, statsChannel.Port, "json/v1", &statsOutput)
	if err != nil {
		log.Warnf("problem with retrieving stats from named: %s", err)
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
}

// Get state of named daemon using ForwardRndcCommand function.
// The state that is stored into dbApp includes: version, number of zones, and
// some runtime state.
func GetAppState(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, eventCenter eventcenter.EventCenter) {
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	command := "status"
	out, err := agents.ForwardRndcCommand(ctx2, dbApp, command)
	if err != nil {
		log.Warnf("problem with getting BIND 9 status: %s", err)
		return
	}

	bind9Daemon := dbmodel.NewBind9Daemon(false)

	// Get version
	pattern := regexp.MustCompile(`version:\s+(.+)\n`)
	match := pattern.FindStringSubmatch(out.Output)
	if match != nil {
		bind9Daemon.Version = match[1]
	} else {
		log.Warnf("cannot get BIND 9 version: unable to find version in output")
	}

	// Is the named daemon running?
	pattern = regexp.MustCompile(`server is up and running`)
	up := pattern.FindString(out.Output)
	if up != "" {
		bind9Daemon.Active = true
	}

	// Up time
	pattern = regexp.MustCompile(`boot time:\s+(.+)`)
	match = pattern.FindStringSubmatch(out.Output)
	if match != nil {
		bootTime, err := time.Parse(namedLongDateFormat, match[1])
		if err != nil {
			log.Warnf("cannot get BIND 9 up time: %s", err.Error())
		}
		now := time.Now()
		elapsed := now.Sub(bootTime)
		bind9Daemon.Uptime = int64(elapsed.Seconds())
	} else {
		log.Warnf("cannot get BIND 9 up time: unable to find boot time in output")
	}

	// Reloaded at
	pattern = regexp.MustCompile(`last configured:\s+(.+)`)
	match = pattern.FindStringSubmatch(out.Output)
	if match != nil {
		reloadTime, err := time.Parse(namedLongDateFormat, match[1])
		if err != nil {
			log.Warnf("cannot get BIND 9 reload time: %s", err.Error())
		}
		bind9Daemon.ReloadedAt = reloadTime
	} else {
		log.Warnf("cannot get BIND 9 reload time: unable to find last configured in output")
	}

	// Number of zones
	pattern = regexp.MustCompile(`number of zones:\s+(\d+)\s+\((\d+) automatic\)`)
	match = pattern.FindStringSubmatch(out.Output)
	if match != nil {
		count, err := strconv.Atoi(match[1])
		if err != nil {
			log.Warnf("cannot get BIND 9 number of zones: %s", err.Error())
		}
		autoCount, err := strconv.Atoi(match[2])
		if err != nil {
			log.Warnf("cannot get BIND 9 number of automatic zones: %s", err.Error())
		}
		bind9Daemon.Bind9Daemon.Stats.ZoneCount = int64(count - autoCount)
		bind9Daemon.Bind9Daemon.Stats.AutomaticZoneCount = int64(autoCount)
	} else {
		log.Warnf("cannot get BIND 9 number of zones: unable to find number of zones in output")
	}

	// Save status
	dbApp.Active = bind9Daemon.Active
	dbApp.Meta.Version = bind9Daemon.Version
	dbApp.Daemons = []*dbmodel.Daemon{
		bind9Daemon,
	}

	// Get statistics
	GetAppStatistics(ctx, agents, dbApp)
}

// Inserts or updates information about BIND 9 app in the database.
func CommitAppIntoDB(db *dbops.PgDB, app *dbmodel.App, eventCenter eventcenter.EventCenter) (err error) {
	if app.ID == 0 {
		_, err = dbmodel.AddApp(db, app)
		eventCenter.AddInfoEvent("added {app}", app.Machine, app)
	} else {
		_, _, err = dbmodel.UpdateApp(db, app)
	}
	// todo: perform any additional actions required after storing the
	// app in the db.
	return err
}
