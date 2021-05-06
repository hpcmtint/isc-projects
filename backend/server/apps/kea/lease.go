package kea

import (
	"context"
	"net"
	"reflect"

	errors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	keactrl "isc.org/stork/appctrl/kea"
	keadata "isc.org/stork/appdata/kea"
	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Structure representing a response to a command fetching a
// single DHCPv4 lease from the Kea server.
type Lease4GetResponse struct {
	keactrl.ResponseHeader
	Arguments *dbmodel.Lease `json:"arguments,omitempty"`
}

// Structure representing a response to a command fetching a
// single DHCPv6 lease from the Kea server.
type Lease6GetResponse struct {
	keactrl.ResponseHeader
	Arguments *dbmodel.Lease `json:"arguments,omitempty"`
}

// Structure representing arguments of a response to a command
// fetching multiple DHCP leases from the Kea server.
type LeaseGetMultipleResponseArgs struct {
	Leases []dbmodel.Lease
}

// Structure representing a response to a command fetching multiple
// DHCP leases from the Kea server.
type LeaseGetMultipleResponse struct {
	keactrl.ResponseHeader
	Arguments *LeaseGetMultipleResponseArgs `json:"arguments,omitempty"`
}

// Validates a response from a Kea daemon to the commands fetching
// leases, e.g. lease4-get-by-hw-address. It checks that the response
// comprises the Success status and that arguments map is not nil.
func validateGetLeasesResponse(commandName string, result int, arguments interface{}) error {
	if result == keactrl.ResponseError {
		return errors.Errorf("error returned by Kea in response to %s command", commandName)
	}
	if result == keactrl.ResponseCommandUnsupported {
		return errors.Errorf("%s command unsupported", commandName)
	}
	argumentsType := reflect.TypeOf(arguments)
	if argumentsType != nil && argumentsType.Kind() == reflect.Ptr {
		if reflect.ValueOf(arguments).IsNil() {
			return errors.Errorf("response to %s command lacks arguments", commandName)
		}
	}
	return nil
}

// Sends a lease4-get command with ip-address argument specifying a searched lease.
// If the lease is found, the pointer to it is returned. If the lease does not
// exist, a nil pointer and nil error are returned.
func GetLease4ByIPAddress(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, ipaddress string) (lease *dbmodel.Lease, err error) {
	daemons, err := keactrl.NewDaemons("dhcp4")
	if err != nil {
		return lease, err
	}
	arguments := map[string]interface{}{
		"ip-address": ipaddress,
	}
	command, err := keactrl.NewCommand("lease4-get", daemons, &arguments)
	if err != nil {
		return lease, err
	}
	response := make([]Lease4GetResponse, 1)
	ctx := context.Background()
	respResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []*keactrl.Command{command}, &response)
	if err != nil {
		return lease, err
	}
	if respResult.Error != nil {
		return lease, respResult.Error
	}
	if len(response) == 0 {
		return lease, errors.Errorf("invalid response to lease4-get command received")
	}
	if response[0].Result == keactrl.ResponseEmpty {
		return lease, nil
	}
	if err = validateGetLeasesResponse("lease4-get", response[0].Result, response[0].Arguments); err != nil {
		return lease, err
	}
	lease = response[0].Arguments
	lease.AppID = dbApp.ID
	lease.App = dbApp
	return lease, nil
}

// Sends a lease6-get command with type and ip-address arguments specifying
// searched lease type and IP address. If the lease is found, the pointer to
// it is returned. If the lease does not exist, a nil pointer and nil error
// are returned.
func GetLease6ByIPAddress(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, leaseType, ipaddress string) (lease *dbmodel.Lease, err error) {
	daemons, err := keactrl.NewDaemons("dhcp6")
	if err != nil {
		return lease, err
	}
	arguments := map[string]interface{}{
		"ip-address": ipaddress,
		"type":       leaseType,
	}
	command, err := keactrl.NewCommand("lease6-get", daemons, &arguments)
	if err != nil {
		return lease, err
	}
	response := make([]Lease6GetResponse, 1)
	ctx := context.Background()
	respResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []*keactrl.Command{command}, &response)
	if err != nil {
		return lease, err
	}
	if respResult.Error != nil {
		return lease, respResult.Error
	}
	if len(response) == 0 {
		return lease, errors.Errorf("invalid response to lease6-get command received")
	}
	if response[0].Result == keactrl.ResponseEmpty {
		return lease, nil
	}
	if err = validateGetLeasesResponse("lease6-get", response[0].Result, response[0].Arguments); err != nil {
		return lease, err
	}
	lease = response[0].Arguments
	lease.AppID = dbApp.ID
	lease.App = dbApp
	return lease, nil
}

// This is a generic function querying a Kea server for leases by specified lease
// properties: hw-address, client-id, DUID or hostname. The type of the property
// is unknown to the function and therefore it sends multiple commands to the Kea
// server using the property value as an input to different commands. It is up
// to the caller to decide which commands this function should send to Kea. For
// example, if the property value is 01:01:01:01, the caller should select the
// lease4-get-by-hw-address, lease4-get-by-client-id and lease6-get-by-duid
// commands. The specified commands are combined in a single gRPC transaction
// to minimize the number of roundtrips between the Stork server and an agent.
func getLeasesByProperties(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, propertyValue string, commandNames ...string) (leases []dbmodel.Lease, warns bool, err error) {
	var commands []*keactrl.Command
	for _, commandName := range commandNames {
		var daemons *keactrl.Daemons
		var propertyName string
		sentPropertyValue := propertyValue
		switch commandName {
		case "lease4-get-by-hw-address":
			daemons, err = keactrl.NewDaemons("dhcp4")
			if err != nil {
				return leases, false, err
			}
			// Searching by empty MAC address is allowed when trying to find declined leases.
			// If the value is non-empty, it has to be properly formatted.
			if len(sentPropertyValue) > 0 {
				// When searching by MAC address we must ensure that it has the format
				// expected by Kea, i.e. 01:02:03:04:05:06.
				if formattedPropertyValue, ok := storkutil.FormatMACAddress(sentPropertyValue); ok {
					sentPropertyValue = formattedPropertyValue
				} else {
					return leases, false, errors.Errorf("invalid format of the property %s used to get leases by MAC address from Kea", propertyValue)
				}
			}
			propertyName = "hw-address"
		case "lease4-get-by-client-id":
			daemons, err = keactrl.NewDaemons("dhcp4")
			if err != nil {
				return leases, false, err
			}
			propertyName = "client-id"
		case "lease6-get-by-duid":
			// Kea does not accept empty DUIDs. Empty DUID in Kea is represented by a zero byte.
			if len(sentPropertyValue) == 0 {
				sentPropertyValue = "0"
			}
			daemons, err = keactrl.NewDaemons("dhcp6")
			if err != nil {
				return leases, false, err
			}
			propertyName = "duid"
		case "lease4-get-by-hostname":
			daemons, err = keactrl.NewDaemons("dhcp4")
			if err != nil {
				return leases, false, err
			}
			propertyName = "hostname"
		case "lease6-get-by-hostname":
			daemons, err = keactrl.NewDaemons("dhcp6")
			if err != nil {
				return leases, false, err
			}
			propertyName = "hostname"
		default:
			continue
		}
		arguments := map[string]interface{}{
			propertyName: sentPropertyValue,
		}
		command, err := keactrl.NewCommand(commandName, daemons, &arguments)
		if err != nil {
			return leases, false, err
		}
		commands = append(commands, command)
	}

	// A caller specified no commands or command names were invalid.
	if len(commands) == 0 {
		return leases, false, nil
	}

	// Create container for responses to each command sent.
	var responses []interface{}
	for range commands {
		response := make([]LeaseGetMultipleResponse, 1)
		responses = append(responses, &response)
	}

	ctx := context.Background()

	// Send all commands to Kea via Stork agent.
	respResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, commands, responses...)
	if err != nil {
		return leases, false, err
	}

	if respResult.Error != nil {
		return leases, false, respResult.Error
	}

	// Validate responses to all commands.
	for i, r := range responses {
		response := r.(*[]LeaseGetMultipleResponse)

		// This is rather an impossible condition, so if it occurs something is
		// heavily broken, so let's bail.
		if len(*response) == 0 {
			return []dbmodel.Lease{}, false, errors.Errorf("invalid response received from Kea to the %s command", commands[i].Command)
		}
		// Ignore empty response. It is valid but there are no leases,
		// so there is nothing more to do.
		if (*response)[0].Result != keactrl.ResponseEmpty {
			if err = validateGetLeasesResponse(commands[i].Command, (*response)[0].Result, (*response)[0].Arguments); err != nil {
				// Log an error and continue. Maybe there is a communication problem
				// with one daemon, but the other one is still operational.
				log.Warn(err)
				warns = true
			} else {
				leases = append(leases, (*response)[0].Arguments.Leases...)
			}
		}
	}
	for i := range leases {
		leases[i].AppID = dbApp.ID
		leases[i].App = dbApp
	}
	return leases, warns, nil
}

// Sends lease4-get-by-hw-address command to Kea.
func GetLeases4ByHWAddress(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, hwaddress string) (leases []dbmodel.Lease, err error) {
	leases, _, err = getLeasesByProperties(agents, dbApp, hwaddress, "lease4-get-by-hw-address")
	return leases, err
}

// Sends lease4-get-by-client-id command to Kea.
func GetLeases4ByClientID(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, clientID string) (leases []dbmodel.Lease, err error) {
	leases, _, err = getLeasesByProperties(agents, dbApp, clientID, "lease4-get-by-client-id")
	return leases, err
}

// Sends lease4-get-by-hostname command to Kea.
func GetLeases4ByHostname(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, hostname string) (leases []dbmodel.Lease, err error) {
	leases, _, err = getLeasesByProperties(agents, dbApp, hostname, "lease4-get-by-hostname")
	return leases, err
}

// Sends lease6-get-by-duid command to Kea.
func GetLeases6ByDUID(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, duid string) (leases []dbmodel.Lease, err error) {
	leases, _, err = getLeasesByProperties(agents, dbApp, duid, "lease6-get-by-duid")
	return leases, err
}

// Sends lease6-get-by-hostname command to Kea.
func GetLeases6ByHostname(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, hostname string) (leases []dbmodel.Lease, err error) {
	leases, _, err = getLeasesByProperties(agents, dbApp, hostname, "lease6-get-by-hostname")
	return leases, err
}

// Convenience function checking if a daemon being a part of the specified app
// has the libdhcp_lease_cmds hooks library configured.
func hasLeaseCmdsHook(app *dbmodel.App, daemonName string) bool {
	daemon := app.GetDaemonByName(daemonName)
	if daemon != nil && daemon.KeaDaemon != nil && daemon.KeaDaemon.Config != nil {
		if _, _, ok := daemon.KeaDaemon.Config.GetHooksLibrary("libdhcp_lease_cmds"); ok {
			return true
		}
	}
	return false
}

// Attempts to find a lease on the Kea servers by specified text.
// It expects that the text is an IP address, MAC address, client
// identifier, or hostname matching a lease. The server contacts
// all Kea servers, which may potentially have the lease. If
// multiple servers have the same lease (e.g. in HA configuration),
// it returns all that lease instances. The  Kea servers which
// returned an error response are returned in the second value.
// Such failures do not preclude the function from returning
// leases found on other servers, but the caller becomes aware
// that some leases may not be included due to the communication
// errors with some servers. The third returned value indicates
// a general error, e.g. issues with Stork database communication.
func FindLeases(db *dbops.PgDB, agents agentcomm.ConnectedAgents, text string) (leases []dbmodel.Lease, erredApps []*dbmodel.App, err error) {
	// Recognize if the text comprises an IP address or some identifier,
	// e.g. MAC address or client identifier.
	const (
		ipv4 = iota
		ipv6
		identifier
		hostname
	)
	// By default query by hostname.
	queryType := hostname
	if ip := net.ParseIP(text); ip != nil {
		// It is an IP address. If it converts to 4 bytes, it is
		// an IPv4 address. Otherwise, it is an IPv6 address.
		if ip.To4() != nil {
			queryType = ipv4
		} else {
			queryType = ipv6
		}
	} else if storkutil.IsHexIdentifier(text) {
		// It is a string of hexadecimal digits, so it must be one
		// of the identifiers.
		queryType = identifier
	}

	// Get Kea apps from the database. We will send commands to these
	// apps to find leases.
	apps, err := dbmodel.GetAppsByType(db, dbmodel.AppTypeKea)
	if err != nil {
		err = errors.WithMessagef(err, "failed to fetch Kea apps while searching for leases by %s", text)
		return leases, erredApps, err
	}

	for i := range apps {
		appError := false

		switch queryType {
		case ipv4:
			if hasLeaseCmdsHook(&apps[i], dbmodel.DaemonNameDHCPv4) {
				// This is an IPv4 address, so send the command to the DHCPv4 server.
				lease, err := GetLease4ByIPAddress(agents, &apps[i], text)
				if err != nil {
					appError = true
					log.Warn(err)
				} else if lease != nil {
					leases = append(leases, *lease)
				}
			}
		case ipv6:
			if hasLeaseCmdsHook(&apps[i], dbmodel.DaemonNameDHCPv6) {
				// This is an IPv6 address (or prefix), so send the command to the
				// DHCPv6 server instead.
				for _, leaseType := range []string{"IA_NA", "IA_PD"} {
					lease, err := GetLease6ByIPAddress(agents, &apps[i], leaseType, text)
					if err != nil {
						appError = true
						log.Warn(err)
					} else if lease != nil {
						leases = append(leases, *lease)
						// If we found a lease by IP address there is no reason to
						// query by delegated prefix because the IP address/prefix
						// must be unique in the database.
						break
					}
				}
			}
		default:
			// The remaining cases are to query by identifier or hostname. They share
			// lots of common code, so they are combined in their own switch statement.
			var commands []string
			switch queryType {
			case identifier:
				if hasLeaseCmdsHook(&apps[i], dbmodel.DaemonNameDHCPv4) {
					commands = append(commands, "lease4-get-by-hw-address", "lease4-get-by-client-id")
				}
				if hasLeaseCmdsHook(&apps[i], dbmodel.DaemonNameDHCPv6) {
					commands = append(commands, "lease6-get-by-duid")
				}
			default:
				if hasLeaseCmdsHook(&apps[i], dbmodel.DaemonNameDHCPv4) {
					commands = append(commands, "lease4-get-by-hostname")
				}
				if hasLeaseCmdsHook(&apps[i], dbmodel.DaemonNameDHCPv6) {
					commands = append(commands, "lease6-get-by-hostname")
				}
			}
			// Search for leases by identifier or hostname.
			leasesByProperties, warns, err := getLeasesByProperties(agents, &apps[i], text, commands...)
			appError = warns
			if err != nil {
				appError = true
				log.Warn(err)
			} else {
				leases = append(leases, leasesByProperties...)
			}
		}
		if appError {
			erredApps = append(erredApps, &apps[i])
		}
	}
	return leases, erredApps, nil
}

// Attempts to find declined leases on the Kea servers. Kea provides no
// API to search the leases by state but the declined leases have HW address
// and DUID empty. Thus, this function sends lease4-get-by-hw-address and
// lease6-get-by-duid with empty hw-address and empty duid parameters
// respectively. Next, it removes the leases which are not in the declined
// state from the result. The  Kea servers which returned an error response
// are returned in second value. Such failures do not preclude the function
// from returning leases found on other servers, but the caller becomes
// aware that some leases may not be included due to the communication
// errors with some servers. The third returned value indicates a general
// error, e.g. issues with Stork database communication.
func FindDeclinedLeases(db *dbops.PgDB, agents agentcomm.ConnectedAgents) (leases []dbmodel.Lease, erredApps []*dbmodel.App, err error) {
	// Get all Kea apps.
	apps, err := dbmodel.GetAppsByType(db, dbmodel.AppTypeKea)
	if err != nil {
		err = errors.WithMessagef(err, "failed to fetch Kea apps while searching for declined leases")
		return leases, erredApps, err
	}

	// Send appropriate commands to each app.
	for i := range apps {
		appError := false

		var commands []string
		if hasLeaseCmdsHook(&apps[i], dbmodel.DaemonNameDHCPv4) {
			commands = append(commands, "lease4-get-by-hw-address")
		}
		if hasLeaseCmdsHook(&apps[i], dbmodel.DaemonNameDHCPv6) {
			commands = append(commands, "lease6-get-by-duid")
		}

		// Send these commands with empty hw-address and empty duid.
		leasesByProperties, warns, err := getLeasesByProperties(agents, &apps[i], "", commands...)
		appError = warns
		if err != nil {
			appError = true
			log.Warn(err)
		} else {
			for j := range leasesByProperties {
				// Only return the leases in the declined state.
				if leasesByProperties[j].State == keadata.LeaseStateDeclined {
					leases = append(leases, leasesByProperties[j])
				}
			}
		}
		if appError {
			erredApps = append(erredApps, &apps[i])
		}
	}
	return leases, erredApps, nil
}

// Attempts to find leases for a given host reservation. An error is returned
// only if there is a problem with database communication. If the host doesn't
// exist, no leases are returned. This function will send commands to all
// monitored Kea servers querying for leases assigned to the given host.
// If there is a communication problem with any of the Kea servers, the details
// of the server are recorded in the erredApps slice.
func FindLeasesByHostID(db *dbops.PgDB, agents agentcomm.ConnectedAgents, hostID int64) (leases []dbmodel.Lease, erredApps []*dbmodel.App, err error) {
	host, err := dbmodel.GetHost(db, hostID)
	if err != nil {
		err = errors.WithMessagef(err, "failed to fetch host with ID %d while searching for its leases", hostID)
		return leases, erredApps, err
	}
	if host == nil {
		return leases, erredApps, err
	}

	// Get Kea apps from the database. We will send commands to these
	// apps to find leases.
	apps, err := dbmodel.GetAppsByType(db, dbmodel.AppTypeKea)
	if err != nil {
		err = errors.WithMessagef(err, "failed to fetch Kea apps while searching for leases for host id %d", hostID)
		return leases, erredApps, err
	}

	for i := range apps {
		// Monitor if a daemon returned an error. We stop sending commands to the
		// daemon it first returns an error.
		dhcp4Error := false
		dhcp6Error := false
		appError := false
		// Go over all IP reservations and send appropriate commands to the app
		// for each of them.
		for _, r := range host.IPReservations {
			parsedIP := storkutil.ParseIP(r.Address)
			if parsedIP == nil {
				// This is rather impossible condition, but let's be safe.
				continue
			}
			// Determine if this is IPv4 or IPv6 lease.
			switch parsedIP.Protocol {
			case storkutil.IPv4:
				if !dhcp4Error && hasLeaseCmdsHook(&apps[i], dbmodel.DaemonNameDHCPv4) {
					lease, err := GetLease4ByIPAddress(agents, &apps[i], parsedIP.NetworkPrefix)
					if err != nil {
						dhcp4Error = true
						log.Warn(err)
					} else if lease != nil {
						leases = append(leases, *lease)
					}
				}
			case storkutil.IPv6:
				if !dhcp6Error && hasLeaseCmdsHook(&apps[i], dbmodel.DaemonNameDHCPv6) {
					// These commands distinguish between IA_NA and IA_PD. A caller
					// must specify the lease type.
					leaseType := "IA_NA"
					if parsedIP.Prefix {
						leaseType = "IA_PD"
					}
					lease, err := GetLease6ByIPAddress(agents, &apps[i], leaseType, parsedIP.NetworkPrefix)
					if err != nil {
						dhcp6Error = true
						log.Warn(err)
					} else if lease != nil {
						leases = append(leases, *lease)
					}
				}
			default:
				// Again, this is impossible condition.
				continue
			}
			// The app returned an error. Maybe the server is unavailable. We don't
			// want to send more commands to a daemon returning an error because there
			// is a minimal chance it will reply with success.
			if dhcp4Error || dhcp6Error {
				if !appError {
					// Record an app for which the error was returned.
					erredApps = append(erredApps, &apps[i])
					appError = true
				}
				// If both daemons returned an error, stop sending any commands to
				// this app.
				if dhcp4Error && dhcp6Error {
					break
				}
			}
		}
	}
	return leases, erredApps, err
}
