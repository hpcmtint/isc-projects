package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	storkutil "isc.org/stork/util"
)

type Bind9Daemon struct {
	Pid     int32
	Name    string
	Version string
	Active  bool
}

type Bind9State struct {
	Version string
	Active  bool
	Daemon  Bind9Daemon
}

// It holds common and BIND 9 specifc runtime information.
type Bind9App struct {
	BaseApp
	RndcClient     *RndcClient // to communicate with BIND 9 via rndc
}

// Get base information about BIND 9 app.
func (ba *Bind9App) GetBaseApp() *BaseApp {
	return &ba.BaseApp
}

// Detect allowed logs provided by BIND 9.
// TODO: currently it is not implemeneted and not used,
// it returns always empty list and no error.
func (ba *Bind9App) DetectAllowedLogs() ([]string, error) {
	return nil, nil
}

const (
	RndcDefaultPort         = 953
	StatsChannelDefaultPort = 80
)

const (
	defaultNamedConfFile1 = "/etc/bind/named.conf"
	defaultNamedConfFile2 = "/etc/opt/isc/isc-bind/named.conf"
)

const namedCheckconf = "named-checkconf"

// getRndcKey looks for the key with a given `name` in `contents`.
//
// Example key clause:
//
//    key "name" {
//        algorithm "hmac-sha256";
//        secret "OmItW1lOyLVUEuvv+Fme+Q==";
//    };
//
func getRndcKey(contents, name string) (controlKey string) {
	ptrn := regexp.MustCompile(`(?s)keys\s+\"(\S+)\"\s+\{(.*)\}\s*;`)
	keys := ptrn.FindAllStringSubmatch(contents, -1)
	if len(keys) == 0 {
		return ""
	}

	for _, key := range keys {
		if key[1] != name {
			continue
		}
		ptrn = regexp.MustCompile(`(?s)algorithm\s+\"(\S+)\";`)
		algorithm := ptrn.FindStringSubmatch(key[2])
		if len(algorithm) < 2 {
			log.Warnf("no key algorithm found for name %s", name)
			return ""
		}

		ptrn = regexp.MustCompile(`(?s)secret\s+\"(\S+)\";`)
		secret := ptrn.FindStringSubmatch(key[2])
		if len(secret) < 2 {
			log.Warnf("no key secret found for name %s", name)
			return ""
		}

		// this key clause matches the name we are looking for
		controlKey = fmt.Sprintf("%s:%s", algorithm[1], secret[1])
		break
	}

	return controlKey
}

// parseInetSpec parses an inet statement from a named configuration excerpt.
// The inet statement is defined by inet_spec:
//
//    inet_spec = ( ip_addr | * ) [ port ip_port ]
//                allow { address_match_list }
//                keys { key_list };
//
// This function returns the ip_addr, port and the first key that is
// referenced in the key_list.  If instead of an ip_addr, the asterisk (*) is
// specified, this function will return 'localhost' as an address.
func parseInetSpec(config, excerpt string) (address string, port int64, key string) {
	// This pattern is build up like this:
	// - inet\s+                 - inet
	// - (\S+\s*\S*\s*\d*)\s+    - ( ip_addr | * ) [ port ip_port ]
	// - allow\s*                - allow
	// - \{(?:\s*\S+\s*;\s*)+)\} - address_match_list
	// - (.*);                   - keys { key_list }; (pattern matched below)
	ptrn := regexp.MustCompile(`(?s)inet\s+(\S+\s*\S*\s*\d*)\s+allow\s*\{(?:\s*\S+\s*;\s*)+\}(.*);`)
	match := ptrn.FindStringSubmatch(excerpt)
	if len(match) == 0 {
		log.Warnf("cannot parse BIND 9 inet configuration: no match (%+v)", config)
		return "", 0, ""
	}

	inetSpec := regexp.MustCompile(`\s+`).Split(match[1], 3)
	switch len(inetSpec) {
	case 1:
		address = inetSpec[0]
	case 3:
		address = inetSpec[0]
		if inetSpec[1] != "port" {
			log.Warnf("cannot parse BIND 9 control port: bad port statement (%+v)", inetSpec)
			return "", 0, ""
		}

		iPort, err := strconv.Atoi(inetSpec[2])
		if err != nil {
			log.Warnf("cannot parse BIND 9 control port: %+v (%+v)", inetSpec, err)
			return "", 0, ""
		}
		port = int64(iPort)
	case 2:
	default:
		log.Warnf("cannot parse BIND 9 inet_spec configuration: no match (%+v)", inetSpec)
		return "", 0, ""
	}

	if len(match) == 3 {
		// Find a key clause. This pattern is build up like this:
		// keys\s*                - keys
		// \{\s*                  - {
		// \"(\S+)\"\s*;          - key_list (first)
		// (?:\s*\"\S+\"\s*;\s*)* - key_list (remainder)
		// \s}\s*                 - }
		ptrn = regexp.MustCompile(`(?s)keys\s*\{\s*\"(\S+)\"\s*;(?:\s*\"\S+\"\s*;\s*)*\}\s*`)
		keyName := ptrn.FindStringSubmatch(match[2])
		if len(keyName) > 1 {
			key = getRndcKey(config, keyName[1])
		}
	}

	if address == "*" {
		address = "localhost"
	}

	return address, port, key
}

// getCtrlAddressFromBind9Config retrieves the rndc control access address,
// port, and secret key (if configured) from the configuration `text`.
//
// Multiple controls clauses may be configured but currently this function
// only matches the first one.  Multiple access points may be listed inside
// a single controls clause, but this function currently only matches the
// first in the list.  A controls clause may look like this:
//
//    controls {
//        inet 127.0.0.1 allow {localhost;};
//        inet * port 7766 allow {"rndc-users";} keys {"rndc-remote";};
//    };
//
// In this example, "rndc-users" and "rndc-remote" refer to an acl and key
// clauses.
//
// Finding the key is done by looking if the control access point has a
// keys parameter and if so, it looks in `path` for a key clause with the
// same name.
func getCtrlAddressFromBind9Config(text string) (controlAddress string, controlPort int64, controlKey string) {
	// Match the following clause:
	//     controls {
	//         inet inet_spec [inet_spec] ;
	//     };
	ptrn := regexp.MustCompile(`(?s)controls\s*\{\s*(.*)\s*\}\s*;`)
	controls := ptrn.FindStringSubmatch(text)
	if len(controls) == 0 {
		return "", 0, ""
	}

	// We only pick the first match, but the controls clause
	// can list multiple control access points.
	controlAddress, controlPort, controlKey = parseInetSpec(text, controls[1])
	if controlAddress != "" {
		// If no port was provided, use the default rndc port.
		if controlPort == 0 {
			controlPort = RndcDefaultPort
		}
	}
	return controlAddress, controlPort, controlKey
}

// getStatisticsChannelFromBind9Config retrieves the statistics channel access
// address, port, and secret key (if configured) from the configuration `text`.
//
// Multiple statistics-channels clauses may be configured but currently this
// function only matches the first one.  Multiple access points may be listed
// inside a single controls clause, but this function currently only matches
// the first in the list.  A statistics-channels clause may look like this:
//
//    statistics-channels {
//        inet 10.1.10.10 port 8080 allow { 192.168.2.10; 10.1.10.2; };
//        inet 127.0.0.1  port 8080 allow { "stats-clients" };
//    };
//
// In this example, "stats-clients" refers to an acl clause.
//
// Finding the key is done by looking if the control access point has a
// keys parameter and if so, it looks in `path` for a key clause with the
// same name.
func getStatisticsChannelFromBind9Config(text string) (statsAddress string, statsPort int64, statsKey string) {
	// Match the following clause:
	//     statistics-channels {
	//         inet inet_spec [inet_spec] ;
	//     };
	ptrn := regexp.MustCompile(`(?s)statistics-channels\s*\{\s*(.*)\s*\}\s*;`)
	channels := ptrn.FindStringSubmatch(text)
	if len(channels) == 0 {
		return "", 0, ""
	}

	// We only pick the first match, but the statistics-channels clause
	// can list multiple control access points.
	statsAddress, statsPort, statsKey = parseInetSpec(text, channels[1])
	if statsAddress != "" {
		// If no port was provided, use the default statschannel port.
		if statsPort == 0 {
			statsPort = StatsChannelDefaultPort
		}
	}
	return statsAddress, statsPort, statsKey
}

func detectBind9App(match []string, cwd string, cmdr storkutil.Commander) App {
	if len(match) < 3 {
		log.Warnf("problem with parsing BIND 9 cmdline: %s", match[0])
		return nil
	}

	// try to find bind9 config file(s)
	namedDir := match[1]
	bind9Params := match[2]
	bind9ConfPath := ""

	// look for config file in cmd params
	paramsPtrn := regexp.MustCompile(`-c\s+(\S+)`)
	m := paramsPtrn.FindStringSubmatch(bind9Params)
	if m != nil {
		bind9ConfPath = m[1]
		// if path to config is not absolute then join it with CWD of named
		if !strings.HasPrefix(bind9ConfPath, "/") {
			bind9ConfPath = path.Join(cwd, bind9ConfPath)
		}
	} else {
		// config path not found in cmdline params so try to guess its location
		for _, f := range []string{defaultNamedConfFile1, defaultNamedConfFile2} {
			if _, err := os.Stat(f); err == nil {
				bind9ConfPath = f
				break
			}
		}
	}

	// no config file so nothing to do
	if bind9ConfPath == "" {
		log.Warnf("cannot find config file for BIND 9")
		return nil
	}

	// run named-checkconf on main config file and get preprocessed content of whole config
	prog := namedCheckconf
	if namedDir != "" {
		// remove sbin or bin at the end
		baseNamedDir, _ := filepath.Split(strings.TrimRight(namedDir, "/"))

		// and now determine where named-checkconf actually is located
		for _, binDir := range []string{"sbin", "bin"} {
			prog2 := path.Join(baseNamedDir, binDir, prog)
			if _, err := os.Stat(prog2); err == nil {
				prog = prog2
				break
			}
		}
	}
	out, err := cmdr.Output(prog, "-p", bind9ConfPath)
	if err != nil {
		log.Warnf("cannot parse BIND 9 config file %s: %+v; %s", bind9ConfPath, err, out)
		return nil
	}
	cfgText := string(out)

	// look for control address in config
	address, port, key := getCtrlAddressFromBind9Config(cfgText)
	if port == 0 || len(address) == 0 {
		log.Warnf("found BIND 9 config file (%s) but cannot parse controls clause", bind9ConfPath)
		return nil
	}
	accessPoints := []AccessPoint{
		{
			Type:    AccessPointControl,
			Address: address,
			Port:    port,
			Key:     key,
		},
	}

	// look for statistics channel address in config
	address, port, key = getStatisticsChannelFromBind9Config(cfgText)
	if port > 0 && len(address) != 0 {
		accessPoints = append(accessPoints, AccessPoint{
			Type:    AccessPointStatistics,
			Address: address,
			Port:    port,
			Key:     key,
		})
	} else {
		log.Warnf("cannot parse BIND 9 statistics-channels clause")
	}

	// rndc is the command to interface with BIND 9.
	rndc := func(command []string) ([]byte, error) {
		cmd := exec.Command(command[0], command[1:]...) //nolint:gosec
		return cmd.Output()
	}

	bind9App := &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		RndcClient: NewRndcClient(rndc),
	}

	return bind9App
}

func (ba *Bind9App) DetectAllowedLogs() ([]string, error) {
	return nil, nil
}

// CommandExecutor takes an array of strings, with the first element of the
// array being the program to call, followed by its arguments.  It returns
// the command output, and possibly an error (for example if running the
// command failed).
type CommandExecutor func([]string) ([]byte, error)

type RndcClient struct {
	execute CommandExecutor
}

const (
	RndcKeyFile1 = "/etc/bind/rndc.key"
	RndcKeyFile2 = "/etc/opt/isc/isc-bind/rndc.key"
)

const (
	RndcPath1 = "/usr/sbin/rndc"
	RndcPath2 = "/opt/isc/isc-bind/root/usr/sbin/rndc"
)

// Create an rndc client to communicate with BIND 9 named daemon.
func NewRndcClient(ce CommandExecutor) *RndcClient {
	rndcClient := &RndcClient{
		execute: ce,
	}
	return rndcClient
}

func (ba *Bind9App) sendCommand(command []string) (output []byte, err error) {
	ctrl, err := getAccessPoint(ba, AccessPointControl)
	if err != nil {
		return nil, err
	}

	rndcPath := ""
	if _, err := os.Stat(RndcPath1); err == nil {
		rndcPath = RndcPath1
	} else if _, err := os.Stat(RndcPath2); err == nil {
		rndcPath = RndcPath2
	} else {
		rndcPath = "rndc"
	}

	rndcCommand := []string{rndcPath, "-s", ctrl.Address, "-p", fmt.Sprintf("%d", ctrl.Port)}
	if len(ctrl.Key) > 0 {
		rndcCommand = append(rndcCommand, "-y")
		rndcCommand = append(rndcCommand, ctrl.Key)
	} else if _, err := os.Stat(RndcKeyFile1); err == nil {
		rndcCommand = append(rndcCommand, "-k")
		rndcCommand = append(rndcCommand, RndcKeyFile1)
	} else if _, err := os.Stat(RndcKeyFile2); err == nil {
		rndcCommand = append(rndcCommand, "-k")
		rndcCommand = append(rndcCommand, RndcKeyFile2)
	}
	rndcCommand = append(rndcCommand, command...)
	log.Debugf("rndc: %+v", rndcCommand)

	return ba.RndcClient.execute(rndcCommand)
}
