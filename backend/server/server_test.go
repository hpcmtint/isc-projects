package server

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/testutil"
)

// Aux function checks if a list of expected strings is present in the string.
func checkOutput(output string, exp []string, reason string) bool {
	for _, x := range exp {
		fmt.Printf("Checking if %s exists in %s.\n", x, reason)
		if !strings.Contains(output, x) {
			fmt.Printf("ERROR: Expected string [%s] not found in %s.\n", x, reason)
			return false
		}
	}
	return true
}

// This is the list of all parameters we expect to see there.
func getExpectedSwitches() []string {
	return []string{
		"-v", "-m", "--metrics", "--version", "-d", "--db-name", "-u", "--db-user", "--db-host",
		"-p", "--db-port", "--db-trace-queries", "--rest-cleanup-timeout", "--rest-graceful-timeout",
		"--rest-max-header-size", "--rest-host", "--rest-port", "--rest-listen-limit",
		"--rest-keep-alive", "--rest-read-timeout", "--rest-write-timeout", "--rest-tls-certificate",
		"--rest-tls-key", "--rest-tls-ca", "--rest-static-files-dir", "--initial-puller-interval",
		"--env-file",
	}
}

// Location of the stork-server man page.
const Man = "../../doc/man/stork-server.8.rst"

// This test checks if stork-agent -h reports all expected command-line switches.
func TestCommandLineSwitches(t *testing.T) {
	// Arrange
	os.Args = make([]string, 2)
	os.Args[1] = "-h"

	// Act
	ss := &StorkServer{}
	var command Command
	var err error
	stdout, _, _ := testutil.CaptureOutput(func() {
		command, err = ss.ParseArgs()
	})

	// Assert
	require.EqualValues(t, HelpCommand, command)
	require.NoError(t, err)
	// Now check that all expected command-line switches are really there.
	require.True(t, checkOutput(string(stdout), getExpectedSwitches(), "stork-agent -h output"))
}

// This test checks if all expected command-line switches are documented.
func TestCommandLineSwitchesDoc(t *testing.T) {
	// Read the contents of the man page
	file, err := os.Open(Man)
	require.NoError(t, err)
	man, err := io.ReadAll(file)
	require.NoError(t, err)

	// And check that all expected switches are mentioned there.
	require.True(t, checkOutput(string(man), getExpectedSwitches(), "stork-agent.8.rst"))
}

// This test checks if stork-agent --version (and -v) report expected version.
func TestCommandLineVersion(t *testing.T) {
	// Let's repeat the test twice (for -v and then for --version)
	for _, opt := range []string{"-v", "--version"} {
		arg := opt
		t.Run(arg, func(t *testing.T) {
			// Arrange
			os.Args = make([]string, 2)
			os.Args[1] = arg

			// Act
			ss := &StorkServer{}
			command, err := ss.ParseArgs()

			// Assert
			require.NoError(t, err)
			require.EqualValues(t, VersionCommand, command)
		})
	}
}

// Test that the Stork Server is constructed properly.
func TestNewStorkServer(t *testing.T) {
	// Arrange
	os.Args = make([]string, 0)
	os.Args = append(os.Args, "stork-server",
		"-m",
		"-d", "dbname",
		"-u", "dbuser",
		"--db-host", "dbhost",
		"-p", "9876",
		"--db-sslmode", "verify-ca",
		"--db-sslcert", "sslcert",
		"--db-sslkey", "sslkey",
		"--db-sslrootcert", "sslrootcert",
		"--db-trace-queries", "all",
		"--rest-cleanup-timeout", "12s",
		"--rest-graceful-timeout", "34m",
		"--rest-max-header-size", "56",
		"--rest-host", "resthost",
		"--rest-port", "1234",
		"--rest-listen-limit", "78",
		"--rest-keep-alive", "90h",
		"--rest-read-timeout", "98s",
		"--rest-write-timeout", "76s",
		"--rest-tls-certificate", "tlscert",
		"--rest-tls-key", "tlskey",
		"--rest-tls-ca", "tlsca",
		"--rest-static-files-dir", "staticdir",
		"--initial-puller-interval", "54",
	)

	// Act
	ss, command, err := NewStorkServer()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, RunCommand, command)

	require.True(t, ss.EnableMetricsEndpoint)
	require.EqualValues(t, "dbname", ss.DBSettings.DBName)
	require.EqualValues(t, "dbuser", ss.DBSettings.User)
	require.EqualValues(t, "dbhost", ss.DBSettings.Host)
	require.EqualValues(t, 9876, ss.DBSettings.Port)
	require.EqualValues(t, "verify-ca", ss.DBSettings.SSLMode)
	require.EqualValues(t, "sslcert", ss.DBSettings.SSLCert)
	require.EqualValues(t, "sslkey", ss.DBSettings.SSLKey)
	require.EqualValues(t, "sslrootcert", ss.DBSettings.SSLRootCert)
	require.EqualValues(t, "all", ss.DBSettings.TraceSQL)
	require.EqualValues(t, 12*time.Second, ss.RestAPISettings.CleanupTimeout)
	require.EqualValues(t, 34*time.Minute, ss.RestAPISettings.GracefulTimeout)
	require.EqualValues(t, 56, ss.RestAPISettings.MaxHeaderSize)
	require.EqualValues(t, "resthost", ss.RestAPISettings.Host)
	require.EqualValues(t, 1234, ss.RestAPISettings.Port)
	require.EqualValues(t, 78, ss.RestAPISettings.ListenLimit)
	require.EqualValues(t, 90*time.Hour, ss.RestAPISettings.KeepAlive)
	require.EqualValues(t, 98*time.Second, ss.RestAPISettings.ReadTimeout)
	require.EqualValues(t, 76*time.Second, ss.RestAPISettings.WriteTimeout)
	require.EqualValues(t, "tlscert", ss.RestAPISettings.TLSCertificate)
	require.EqualValues(t, "tlskey", ss.RestAPISettings.TLSCertificateKey)
	require.EqualValues(t, "tlsca", ss.RestAPISettings.TLSCACertificate)
	require.EqualValues(t, "staticdir", ss.RestAPISettings.StaticFilesDir)
	require.EqualValues(t, 54, ss.InitialPullerInterval)
}

// Test that the Stork Server is not constructed if the arguments are wrong.
func TestNewStorkServerWithWrongCLIArguments(t *testing.T) {
	// Arrange
	os.Args = make([]string, 0)
	os.Args = append(os.Args, "stork-server", "--foo-bar-baz")

	// Act
	ss, command, err := NewStorkServer()

	// Assert
	require.Error(t, err)
	require.Nil(t, ss)
	require.EqualValues(t, NoneCommand, command)
}

// Test that the Stork Server is constructed if no arguments are provided.
func TestNewStorkServerNoArguments(t *testing.T) {
	// Arrange
	os.Args = []string{"stork-server"}

	// Act
	ss, command, err := NewStorkServer()

	// Assert
	require.NoError(t, err)
	require.NotNil(t, ss)
	require.EqualValues(t, RunCommand, command)
}

// Test that the server is bootstrapped properly.
func TestBootstrap(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Initializes DB.
	machine := &dbmodel.Machine{Address: "localhost", AgentPort: 8080}
	_ = dbmodel.AddMachine(db, machine)
	app := &dbmodel.App{
		Type:      dbmodel.AppTypeKea,
		MachineID: machine.ID,
		Active:    true,
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true),
			dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv6, true),
		},
	}
	daemons, _ := dbmodel.AddApp(db, app)
	_ = dbmodel.CommitCheckerPreferences(db, []*dbmodel.ConfigCheckerPreference{
		dbmodel.NewGlobalConfigCheckerPreference("host_cmds_presence"),
		dbmodel.NewDaemonConfigCheckerPreference(daemons[0].ID, "out_of_pool_reservation", false),
	}, nil)

	// Initializes CMD.
	os.Args = []string{"stork-server"}

	server, _, _ := NewStorkServer()

	// Switches to test database.
	server.DBSettings.DBName = db.Options().Database
	hostAndPort := strings.Split(db.Options().Addr, ":")
	server.DBSettings.Host = hostAndPort[0]
	port, _ := strconv.ParseInt(hostAndPort[1], 10, 0)
	server.DBSettings.Port = int(port)
	password := db.Options().Password
	os.Setenv("STORK_DATABASE_PASSWORD", password)
	server.DBSettings.User = db.Options().User

	// Act
	err := server.Bootstrap(false)
	defer server.Shutdown(false)

	// Assert
	require.NoError(t, err)

	// Check that appropriate events have been generated. Events are added
	// to the database asynchronously, so it may take a few attempts before
	// they appear.
	var events []dbmodel.Event
	require.Eventually(t, func() bool {
		events, _, _ = dbmodel.GetEventsByPage(db, 0, 10, dbmodel.EvInfo, nil, nil, nil, nil, "", dbmodel.SortDirAny)
		return len(events) > 0
	}, 5*time.Second, time.Second)
	require.Len(t, events, 1)
	require.Contains(t, events[0].Text, "started Stork Server")

	// Checks if the config review checker states were loaded from the database.
	configReviewCheckerPreferences, _ := server.ReviewDispatcher.GetCheckersMetadata(daemons[0])
	require.GreaterOrEqual(t, len(configReviewCheckerPreferences), 5)
	require.EqualValues(t, "host_cmds_presence", configReviewCheckerPreferences[3].Name)
	require.False(t, configReviewCheckerPreferences[3].GloballyEnabled)
	require.EqualValues(t, "out_of_pool_reservation", configReviewCheckerPreferences[4].Name)
	require.True(t, configReviewCheckerPreferences[4].GloballyEnabled)
	configReviewCheckerPreferences, _ = server.ReviewDispatcher.GetCheckersMetadata(daemons[1])
	require.EqualValues(t, "host_cmds_presence", configReviewCheckerPreferences[3].Name)
	require.False(t, configReviewCheckerPreferences[3].GloballyEnabled)
	require.EqualValues(t, "out_of_pool_reservation", configReviewCheckerPreferences[4].Name)
	require.True(t, configReviewCheckerPreferences[4].GloballyEnabled)

	// Run Bootstrap again with the reload flag set. It should not emit any new events.
	err = server.Bootstrap(true)
	require.NoError(t, err)
	events, _, _ = dbmodel.GetEventsByPage(db, 0, 10, dbmodel.EvInfo, nil, nil, nil, nil, "", dbmodel.SortDirAny)
	require.Len(t, events, 1)

	// Run actual shutdown. It doesn't matter we have already deferred one Shutdown().
	// It will be executed only once.
	server.Shutdown(false)

	// Clear events before we get them again after shutdown.
	events = []dbmodel.Event{}

	// Make sure that the shutdown event has been added.
	require.Eventually(t, func() bool {
		events, _, _ = dbmodel.GetEventsByPage(db, 0, 10, dbmodel.EvInfo, nil, nil, nil, nil, "", dbmodel.SortDirAny)
		return len(events) > 0
	}, 5*time.Second, time.Second)
	require.Len(t, events, 2)
	require.Contains(t, events[1].Text, "shutting down Stork Server")
}

// Test that the environment file may be considered while initializing the server.
func TestNewStorkServerFromEnvFile(t *testing.T) {
	// Arrange
	restore := testutil.CreateEnvironmentRestorePoint()
	t.Cleanup(restore)

	content := `
		STORK_DATABASE_HOST=foo
		STORK_DATABASE_PORT=42
		STORK_DATABASE_NAME=bar
		STORK_REST_HOST=baz
		STORK_SERVER_ENABLE_METRICS=true
	`
	file, _ := os.CreateTemp("", "stork-test-server-*")
	defer (func() {
		file.Close()
		os.Remove(file.Name())
	})()
	_, _ = file.WriteString(content)

	os.Args = make([]string, 0)
	os.Args = append(os.Args, "stork-server",
		// The environment file path must be specified using the '=' sign.
		// It's limitation of the go-flag library.
		fmt.Sprintf("--env-file=%s", file.Name()),
	)

	// Act
	ss, command, err := NewStorkServer()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, RunCommand, command)

	require.EqualValues(t, "foo", ss.DBSettings.Host)
	require.EqualValues(t, 42, ss.DBSettings.Port)
	require.EqualValues(t, "bar", ss.DBSettings.DBName)
	require.EqualValues(t, "baz", ss.RestAPISettings.Host)
	require.True(t, ss.EnableMetricsEndpoint)
}
