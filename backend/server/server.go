package server

import (
	"sync"

	"github.com/go-pg/pg/v10"
	flags "github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork"
	keaconfig "isc.org/stork/appcfg/kea"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/apps"
	"isc.org/stork/server/apps/bind9"
	"isc.org/stork/server/apps/kea"
	"isc.org/stork/server/certs"
	"isc.org/stork/server/config"
	"isc.org/stork/server/configreview"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
	"isc.org/stork/server/metrics"
	"isc.org/stork/server/restservice"
	storkutil "isc.org/stork/util"
)

type Command string

const (
	NoneCommand    Command = "none"
	RunCommand     Command = "run"
	HelpCommand    Command = "help"
	VersionCommand Command = "version"
)

// Global Stork Server state.
type StorkServer struct {
	DBSettings dbops.DatabaseSettings
	DB         *dbops.PgDB

	AgentsSettings agentcomm.AgentsSettings
	Agents         agentcomm.ConnectedAgents

	RestAPISettings restservice.RestAPISettings
	RestAPI         *restservice.RestAPI

	Pullers *apps.Pullers

	InitialPullerInterval int64
	EnableMetricsEndpoint bool
	MetricsCollector      metrics.Collector

	EventCenter eventcenter.EventCenter

	ReviewDispatcher configreview.Dispatcher

	ConfigManager              config.Manager
	DHCPOptionDefinitionLookup keaconfig.DHCPOptionDefinitionLookup
	shutdownOnce               sync.Once
}

// Read environment file settings. It's parsed before the above settings.
type EnvironmentFileSettings struct {
	EnvFile *string `long:"env-file" description:"Read the environment variables from the environment file; applicable only if the flag is set" optional:"true" optional-value:"/etc/stork/server.env"`
}

// Global server settings (called application settings in go-flags nomenclature).
type Settings struct {
	EnvironmentFileSettings
	Version               bool  `short:"v" long:"version" description:"Show software version"`
	EnableMetricsEndpoint bool  `short:"m" long:"metrics" description:"Enable Prometheus /metrics endpoint (no auth)" env:"STORK_SERVER_ENABLE_METRICS"`
	InitialPullerInterval int64 `long:"initial-puller-interval" description:"Initial interval used by pullers fetching data from Kea. If not provided the recommended values for each puller are used." env:"STORK_SERVER_INITIAL_PULLER_INTERVAL"`
}

// Parse the command line arguments into GO structures.
// Returns the expected command to run and error.
func (ss *StorkServer) ParseArgs() (command Command, err error) {
	shortDescription := "Stork Server"
	longDescription := "Stork Server is a Kea and BIND 9 dashboard"

	// Process command line flags.
	// Process the environment file flag.
	var envFileSettings EnvironmentFileSettings
	parser := flags.NewParser(&envFileSettings, flags.IgnoreUnknown)
	parser.ShortDescription = shortDescription
	parser.LongDescription = longDescription

	if _, err := parser.Parse(); err != nil {
		err = errors.Wrap(err, "invalid CLI argument")
		return NoneCommand, err
	}

	if envFileSettings.EnvFile != nil {
		err = storkutil.LoadEnvironmentFileToSetter(
			*envFileSettings.EnvFile,
			storkutil.NewProcessEnvironmentVariableSetter(),
		)
		if err != nil {
			err = errors.WithMessagef(err, "invalid environment file: '%s'", *envFileSettings.EnvFile)
			return NoneCommand, err
		}
	}

	// Process the rest of the flags.
	var serverSettings Settings
	parser = flags.NewParser(&serverSettings, flags.Default)
	parser.ShortDescription = shortDescription
	parser.LongDescription = longDescription

	// Process Database specific args.
	_, err = parser.AddGroup("Database ConnectionFlags", "", &ss.DBSettings)
	if err != nil {
		return
	}

	// Process ReST API specific args.
	_, err = parser.AddGroup("ReST Server Flags", "", &ss.RestAPISettings)
	if err != nil {
		return
	}

	// Process agent comm specific args.
	_, err = parser.AddGroup("Agents Communication Flags", "", &ss.AgentsSettings)
	if err != nil {
		return
	}

	// Do args parsing.
	if _, err := parser.Parse(); err != nil {
		var flagsError *flags.Error
		if errors.As(err, &flagsError) {
			if flagsError.Type == flags.ErrHelp {
				return HelpCommand, nil
			}
		}
		return NoneCommand, err
	}

	ss.EnableMetricsEndpoint = serverSettings.EnableMetricsEndpoint
	ss.InitialPullerInterval = serverSettings.InitialPullerInterval

	if serverSettings.Version {
		// If user specified --version or -v, print the version and quit.
		return VersionCommand, nil
	}
	return RunCommand, nil
}

// Init for Stork Server state from the CLI arguments and environment variables.
// Returns the server object (if no error occurs), command to execute and error.
func NewStorkServer() (ss *StorkServer, command Command, err error) {
	ss = &StorkServer{}
	command, err = ss.ParseArgs()
	if err != nil {
		return nil, command, err
	}

	return ss, command, nil
}

// Establishes a database connection, runs the background pullers, and
// prepares the REST API. The reload flag indicates if the server is
// starting up (reload=false) or it is being reloaded (reload=true).
func (ss *StorkServer) Bootstrap(reload bool) (err error) {
	// setup database connection
	ss.DB, err = dbops.NewPgDB(&ss.DBSettings)
	if err != nil {
		return err
	}

	// initialize stork settings
	err = dbmodel.InitializeSettings(ss.DB, ss.InitialPullerInterval)
	if err != nil {
		return err
	}

	// prepare certificates for establish secure connections
	// between server and agents
	caCertPEM, serverCertPEM, serverKeyPEM, err := certs.SetupServerCerts(ss.DB)
	if err != nil {
		return err
	}

	// setup event center
	ss.EventCenter = eventcenter.NewEventCenter(ss.DB)

	// setup connected agents
	ss.Agents = agentcomm.NewConnectedAgents(&ss.AgentsSettings, ss.EventCenter, caCertPEM, serverCertPEM, serverKeyPEM)
	// TODO: if any operation below fails then this Shutdown here causes segfault.
	// I do not know why and do not know how to fix this. Commenting out for now.
	// defer func() {
	// 	if err != nil {
	// 		ss.Agents.Shutdown()
	// 	}
	// }()

	// Setup configuration review dispatcher.
	ss.ReviewDispatcher = configreview.NewDispatcher(ss.DB)
	configreview.RegisterDefaultCheckers(ss.ReviewDispatcher)
	err = configreview.LoadAndValidateCheckerPreferences(ss.DB, ss.ReviewDispatcher)
	if err != nil {
		return err
	}
	ss.ReviewDispatcher.Start()

	// initialize stork statistics
	err = dbmodel.InitializeStats(ss.DB)
	if err != nil {
		return err
	}

	ss.Pullers = &apps.Pullers{}

	// This instance provides functions to search for option definitions, both in the
	// database and among the standard options. It is required by the config manager.
	ss.DHCPOptionDefinitionLookup = dbmodel.NewDHCPOptionDefinitionLookup()

	// setup apps state puller
	ss.Pullers.AppsStatePuller, err = apps.NewStatePuller(ss.DB, ss.Agents, ss.EventCenter, ss.ReviewDispatcher, ss.DHCPOptionDefinitionLookup)
	if err != nil {
		return err
	}

	// setup bind9 stats puller
	ss.Pullers.Bind9StatsPuller, err = bind9.NewStatsPuller(ss.DB, ss.Agents, ss.EventCenter)
	if err != nil {
		return err
	}

	// setup kea stats puller
	ss.Pullers.KeaStatsPuller, err = kea.NewStatsPuller(ss.DB, ss.Agents)
	if err != nil {
		return err
	}

	// Setup Kea hosts puller.
	ss.Pullers.KeaHostsPuller, err = kea.NewHostsPuller(ss.DB, ss.Agents, ss.ReviewDispatcher, ss.DHCPOptionDefinitionLookup)
	if err != nil {
		return err
	}

	// Setup Kea HA status puller.
	ss.Pullers.HAStatusPuller, err = kea.NewHAStatusPuller(ss.DB, ss.Agents)
	if err != nil {
		return err
	}

	if ss.EnableMetricsEndpoint {
		ss.MetricsCollector, err = metrics.NewCollector(ss.DB)
		if err != nil {
			return err
		}
		log.Info("The metrics endpoint is enabled (ensure that it is properly secured)")
	} else {
		log.Warn("The metric endpoint is disabled (it can be enabled with the -m flag)")
	}

	// Create the config manager instance. It takes config.ManagerAccessors interface
	// as a parameter. The manager uses this interface to setup its state. For example,
	// it stores the instance of the DHCP option definition lookup. Note, that it is
	// important to maintain one instance of the lookup because it applies indexing
	// on the option definitions it returns. Indexing should be done only once at
	// server startup.
	ss.ConfigManager = apps.NewManager(ss)

	// setup ReST API service
	r, err := restservice.NewRestAPI(&ss.RestAPISettings, &ss.DBSettings,
		ss.DB, ss.Agents, ss.EventCenter,
		ss.Pullers, ss.ReviewDispatcher, ss.MetricsCollector, ss.ConfigManager,
		ss.DHCPOptionDefinitionLookup)
	if err != nil {
		ss.Pullers.HAStatusPuller.Shutdown()
		ss.Pullers.KeaHostsPuller.Shutdown()
		ss.Pullers.KeaStatsPuller.Shutdown()
		ss.Pullers.Bind9StatsPuller.Shutdown()
		ss.Pullers.AppsStatePuller.Shutdown()
		if ss.MetricsCollector != nil {
			ss.MetricsCollector.Shutdown()
		}

		ss.DB.Close()
		return err
	}
	ss.RestAPI = r

	// Do not issue the event if we're only reloading the server.
	if !reload {
		ss.EventCenter.AddInfoEvent("started Stork Server", "version: "+stork.Version+"\nbuild date: "+stork.BuildDate)
	}

	return nil
}

// Run Stork Server.
func (ss *StorkServer) Serve() {
	// Start listening for requests from ReST API.
	err := ss.RestAPI.Serve()
	if err != nil {
		log.Fatalf("FATAL error: %+v", err)
	}
}

// Shutdown for Stork Server state. The reload flag indicates if the
// Shutdown is called as part of the server reload (reload=true) or
// the process is terminating (reload=false).
func (ss *StorkServer) Shutdown(reload bool) {
	// If this function is called multiple times, make sure we shutdown
	// only once.
	ss.shutdownOnce.Do(func() {
		// Do not issue the event when we're reloading instead of terminating
		// the server process.
		if !reload {
			ss.EventCenter.AddInfoEvent("shutting down Stork Server")
			log.Println("Shutting down Stork Server")
		}
		ss.RestAPI.Shutdown()
		ss.Pullers.HAStatusPuller.Shutdown()
		ss.Pullers.KeaHostsPuller.Shutdown()
		ss.Pullers.KeaStatsPuller.Shutdown()
		ss.Pullers.Bind9StatsPuller.Shutdown()
		ss.Pullers.AppsStatePuller.Shutdown()
		ss.Agents.Shutdown()
		ss.EventCenter.Shutdown()
		ss.ReviewDispatcher.Shutdown()
		if ss.MetricsCollector != nil {
			ss.MetricsCollector.Shutdown()
		}
		ss.DB.Close()

		if !reload {
			log.Println("Stork Server shut down")
		}
	})
}

// Returns an instance of the database handler used by the configuration manager.
func (ss *StorkServer) GetDB() *pg.DB {
	return ss.DB
}

// Returns an interface to the agents the manager communicates with.
func (ss *StorkServer) GetConnectedAgents() agentcomm.ConnectedAgents {
	return ss.Agents
}

// Returns an interface to the instance providing the DHCP option definition
// lookup logic.
func (ss *StorkServer) GetDHCPOptionDefinitionLookup() keaconfig.DHCPOptionDefinitionLookup {
	return ss.DHCPOptionDefinitionLookup
}
