package kea

import (
	"context"
	"fmt"
	"time"

	errors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
	storkutil "isc.org/stork/util"
)

const (
	dhcp4 = "dhcp4"
	dhcp6 = "dhcp6"
	d2    = "d2"
)

// Get list of hooks for all DHCP daemons of the given Kea application.
func GetDaemonHooks(dbApp *dbmodel.App) map[string][]string {
	hooksByDaemon := make(map[string][]string)

	// go through response list with configs from each daemon and retrieve their hooks lists
	for _, dmn := range dbApp.Daemons {
		if dmn.KeaDaemon == nil || dmn.KeaDaemon.Config == nil {
			continue
		}

		libraries := dmn.KeaDaemon.Config.GetHooksLibraries()
		hooks := []string{}
		for _, library := range libraries {
			hooks = append(hooks, library.Library)
		}
		hooksByDaemon[dmn.Name] = hooks
	}

	return hooksByDaemon
}

// === CA config-get response structs ================================================

type SocketData struct {
	SocketName string `json:"socket-name"`
	SocketType string `json:"socket-type"`
}

type ControlSocketsData struct {
	D2      *SocketData
	Dhcp4   *SocketData
	Dhcp6   *SocketData
	NetConf *SocketData
}

type ControlAgentData struct {
	ControlSockets *ControlSocketsData `json:"control-sockets"`
}

type CAConfigGetRespArgs struct {
	ControlAgent *ControlAgentData `json:"Control-agent"`
}

type CAConfigGetResponse struct {
	agentcomm.KeaResponseHeader
	Arguments *CAConfigGetRespArgs
}

// === version-get response structs ===============================================

type VersionGetRespArgs struct {
	Extended string
}

type VersionGetResponse struct {
	agentcomm.KeaResponseHeader
	Arguments *VersionGetRespArgs `json:"arguments,omitempty"`
}

// Get state of Kea application Control Agent using ForwardToKeaOverHTTP function.
// The state, that is stored into dbApp, includes: version and config of CA.
// It also returns:
// - list of all Kea daemons
// - list of DHCP daemons (dhcpv4 and/or dhcpv6)
func getStateFromCA(ctx context.Context, agents agentcomm.ConnectedAgents, caAddress string, caPort int64, dbApp *dbmodel.App, daemonsMap map[string]*dbmodel.Daemon, daemonsErrors map[string]string) (agentcomm.KeaDaemons, agentcomm.KeaDaemons, error) {
	// prepare the command to get config and version from CA
	cmds := []*agentcomm.KeaCommand{
		{
			Command: "version-get",
		},
		{
			Command: "config-get",
		},
	}

	// get version and config from CA
	versionGetResp := []VersionGetResponse{}
	caConfigGetResp := []CAConfigGetResponse{}

	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, cmds, &versionGetResp, &caConfigGetResp)
	if err != nil {
		return nil, nil, err
	}
	if cmdsResult.Error != nil {
		return nil, nil, cmdsResult.Error
	}

	// process the response from CA

	// first find old record of CA daemon in old daemons assigned to the app
	found := false
	for _, dmn := range dbApp.Daemons {
		if dmn.Name == "ca" {
			dmnCopy := *dmn
			daemonsMap["ca"] = &dmnCopy
			found = true
		}
	}
	// if not found then prepare new record
	if !found {
		daemonsMap["ca"] = &dbmodel.Daemon{
			Name:      "ca",
			Active:    true,
			Monitored: true,
		}
	}

	// if no error in the version-get response then copy retrieved info about CA to its record
	dmn := daemonsMap["ca"]
	err = cmdsResult.CmdsErrors[0]
	if err != nil || versionGetResp == nil || len(versionGetResp) == 0 || versionGetResp[0].Result != 0 {
		dmn.Active = false
		errStr := "problem with version-get response from CA: "
		if err != nil {
			errStr += fmt.Sprintf("%s", err)
		} else if versionGetResp == nil || len(versionGetResp) == 0 {
			errStr += fmt.Sprintf("empty response")
		} else {
			errStr += fmt.Sprintf("result == %d, msg: %s", versionGetResp[0].Result, versionGetResp[0].Result)
		}
		log.Warnf(errStr)
		daemonsErrors["ca"] = errStr
		return nil, nil, err
	}

	dmn.Version = versionGetResp[0].Text
	dbApp.Meta.Version = versionGetResp[0].Text
	if versionGetResp[0].Arguments != nil {
		dmn.ExtendedVersion = versionGetResp[0].Arguments.Extended
	}

	// if no error in the config-get response then copy retrieved info about available daemons
	if caConfigGetResp == nil || len(caConfigGetResp) == 0 || caConfigGetResp[0].Result != 0 || caConfigGetResp[0].Arguments == nil || caConfigGetResp[0].Arguments.ControlAgent == nil || caConfigGetResp[0].Arguments.ControlAgent.ControlSockets == nil {
		dmn.Active = false
		errStr := "problem with config-get response from CA: "
		if caConfigGetResp == nil || len(caConfigGetResp) == 0 || caConfigGetResp[0].Arguments == nil || caConfigGetResp[0].Arguments.ControlAgent == nil || caConfigGetResp[0].Arguments.ControlAgent.ControlSockets == nil {
			errStr += "response is empty"
		} else {
			errStr += fmt.Sprintf("result == %d, msg: %s", caConfigGetResp[0].Result, caConfigGetResp[0].Result)
		}
		log.Warnf(errStr)
		daemonsErrors["ca"] = errStr
		return nil, nil, err
	}

	allDaemons := make(agentcomm.KeaDaemons)
	dhcpDaemons := make(agentcomm.KeaDaemons)
	if caConfigGetResp[0].Arguments.ControlAgent.ControlSockets != nil {
		if caConfigGetResp[0].Arguments.ControlAgent.ControlSockets.Dhcp4 != nil {
			allDaemons[dhcp4] = true
			dhcpDaemons[dhcp4] = true
		}
		if caConfigGetResp[0].Arguments.ControlAgent.ControlSockets.Dhcp6 != nil {
			allDaemons[dhcp6] = true
			dhcpDaemons[dhcp6] = true
		}
		if caConfigGetResp[0].Arguments.ControlAgent.ControlSockets.D2 != nil {
			allDaemons[d2] = true
		}
	}

	return allDaemons, dhcpDaemons, nil
}

// Get state of Kea application daemons (beside Control Agent) using ForwardToKeaOverHTTP function.
// The state, that is stored into dbApp, includes: version, config and runtime state of indicated Kea daemons.
func getStateFromDaemons(ctx context.Context, agents agentcomm.ConnectedAgents, caAddress string, caPort int64, dbApp *dbmodel.App, daemonsMap map[string]*dbmodel.Daemon, allDaemons agentcomm.KeaDaemons, dhcpDaemons agentcomm.KeaDaemons, daemonsErrors map[string]string) error {
	now := storkutil.UTCNow()

	// issue 3 commands to Kea daemons at once to get their state
	cmds := []*agentcomm.KeaCommand{
		{
			Command: "version-get",
			Daemons: &allDaemons,
		},
		{
			Command: "status-get",
			Daemons: &dhcpDaemons,
		},
		{
			Command: "config-get",
			Daemons: &allDaemons,
		},
	}

	versionGetResp := []VersionGetResponse{}
	statusGetResp := []StatusGetResponse{}
	configGetResp := []agentcomm.KeaResponse{}

	cmdsResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, cmds, &versionGetResp, &statusGetResp, &configGetResp)
	if err != nil {
		return err
	}
	if cmdsResult.Error != nil {
		return cmdsResult.Error
	}

	// first find old records of daemons in old daemons assigned to the app
	for name := range allDaemons {
		found := false
		for _, dmn := range dbApp.Daemons {
			if dmn.Name == name {
				dmnCopy := *dmn

				// at first all daemons are marked as active; active state will be updated below
				dmnCopy.Active = true

				daemonsMap[name] = &dmnCopy
				found = true
			}
		}
		// if not found then prepare new record
		if !found {
			daemonsMap[name] = dbmodel.NewKeaDaemon(name, true)
		}
	}

	// process version-get responses
	err = cmdsResult.CmdsErrors[0]
	if err != nil {
		return errors.WithMessage(err, "problem with version-get response")
	} else {
		for _, vRsp := range versionGetResp {
			dmn, ok := daemonsMap[vRsp.Daemon]
			if !ok {
				log.Warn("unrecognized daemon in version-get response: %v", vRsp)
				continue
			}
			if vRsp.Result != 0 {
				dmn.Active = false
				errStr := fmt.Sprintf("problem with version-get and kea daemon %s: %s", vRsp.Daemon, vRsp.Text)
				log.Warnf(errStr)
				daemonsErrors[dmn.Name] = errStr
				continue
			}

			dmn.Version = vRsp.Text
			if vRsp.Arguments != nil {
				dmn.ExtendedVersion = vRsp.Arguments.Extended
			}
		}
	}

	// process status-get responses
	err = cmdsResult.CmdsErrors[1]
	if err != nil {
		return errors.WithMessage(err, "problem with status-get response")
	} else {
		for _, sRsp := range statusGetResp {
			dmn, ok := daemonsMap[sRsp.Daemon]
			if !ok {
				log.Warn("unrecognized daemon in status-get response: %v", sRsp)
				continue
			}
			if sRsp.Result != 0 {
				dmn.Active = false
				errStr := fmt.Sprintf("problem with status-get and kea daemon %s: %s", sRsp.Daemon, sRsp.Text)
				log.Warnf(errStr)
				daemonsErrors[dmn.Name] = errStr
				continue
			}

			if sRsp.Arguments != nil {
				dmn.Uptime = sRsp.Arguments.Uptime
				dmn.ReloadedAt = now.Add(time.Second * time.Duration(-sRsp.Arguments.Reload))
			}
		}
	}

	// process config-get responses
	err = cmdsResult.CmdsErrors[2]
	if err != nil {
		return errors.WithMessage(err, "problem with config-get response")
	} else {
		for _, cRsp := range configGetResp {
			dmn, ok := daemonsMap[cRsp.Daemon]
			if !ok {
				log.Warn("unrecognized daemon in config-get response: %v", cRsp)
				continue
			}
			if cRsp.Result != 0 {
				dmn.Active = false
				log.Warnf("problem with config-get and kea daemon %s: %s", cRsp.Daemon, cRsp.Text)
				continue
			}

			dmn.KeaDaemon.Config = dbmodel.NewKeaConfig(cRsp.Arguments)
		}
	}

	return nil
}

// Get state of Kea application daemons using ForwardToKeaOverHTTP function.
// The state that is stored into dbApp includes: version, config and runtime state of indicated Kea daemons.
func GetAppState(ctx context.Context, agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, eventCenter eventcenter.EventCenter) []*dbmodel.Event {
	// prepare URL to CA
	ctrlPoint, err := dbApp.GetAccessPoint(dbmodel.AccessPointControl)
	if err != nil {
		log.Warnf("problem with getting kea access control point: %s", err)
		return nil
	}

	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// get state from CA
	daemonsMap := map[string]*dbmodel.Daemon{}
	daemonsErrors := map[string]string{}
	allDaemons, dhcpDaemons, err := getStateFromCA(ctx2, agents, ctrlPoint.Address, ctrlPoint.Port, dbApp, daemonsMap, daemonsErrors)
	if err != nil {
		log.Warnf("problem with getting state from kea CA: %s", err)
		errStr, ok := daemonsErrors["ca"]
		if !ok {
			errStr = fmt.Sprintf("%s", err)
		}
		eventCenter.AddErrorEvent("cannot get state from Kea CA from {app} on {machine}", errStr, dbApp.Machine, dbApp)
	}

	// if no problems then now get state from the rest of Kea daemons
	err = getStateFromDaemons(ctx2, agents, ctrlPoint.Address, ctrlPoint.Port, dbApp, daemonsMap, allDaemons, dhcpDaemons, daemonsErrors)
	if err != nil {
		log.Warnf("problem with getting state from kea daemons: %s", err)
		errStr := fmt.Sprintf("%s", err)
		eventCenter.AddErrorEvent("cannot get state from Kea CA from {app} on {machine}", errStr, dbApp.Machine, dbApp)
	}

	// store all collected details in app db record
	newActive := true
	var newDaemons []*dbmodel.Daemon
	var events []*dbmodel.Event

	// If app already existed then...
	if dbApp.ID != 0 {
		newCADmn, ok := daemonsMap["ca"]
		if !ok || !newCADmn.Active {
			for _, oldDmn := range dbApp.Daemons {
				if oldDmn.Active {
					// add ref to app in daemon so it is available in CreateEvent
					oldDmn.App = dbApp
					oldDmn.Active = false
					errStr, ok := daemonsErrors[oldDmn.Name]
					if !ok {
						errStr = ""
					}
					ev := eventcenter.CreateEvent(dbmodel.EvError, "{daemon} is down", errStr, dbApp.Machine, dbApp, oldDmn)
					events = append(events, ev)
				}
			}
			if dbApp.Active {
				dbApp.Active = false
				ev := eventcenter.CreateEvent(dbmodel.EvError, "{app} is down", dbApp.Machine, dbApp)
				events = append(events, ev)
			}
			return events
		}

		for name := range daemonsMap {
			dmn := daemonsMap[name]
			// if all daemons are active then whole app is active
			newActive = newActive && dmn.Active

			// if this is a new app then just add detected daemon
			newDaemons = append(newDaemons, dmn)

			// Determine changes in app daemons state and store them as events.
			// Later this events will be passed to EventCenter when all the changes
			// are stored in database.
			for _, oldDmn := range dbApp.Daemons {
				if dmn.Name == oldDmn.Name {
					// add ref to app in daemon so it is available in CreateEvent
					oldDmn.App = dbApp
					// check if daemon changed Active state
					if dmn.Active != oldDmn.Active {
						lvl := dbmodel.EvWarning
						text := "{daemon} is "
						if dmn.Active && !oldDmn.Active {
							text += "up"
						} else if !dmn.Active && oldDmn.Active {
							text += "down"
							lvl = dbmodel.EvError
						}
						errStr, ok := daemonsErrors[oldDmn.Name]
						if !ok {
							errStr = ""
						}
						ev := eventcenter.CreateEvent(lvl, text, errStr, dbApp.Machine, dbApp, oldDmn)
						events = append(events, ev)

					// check if daemon has been restarted
					} else if dmn.Uptime < oldDmn.Uptime {
						text := "{daemon} has been restarted"
						ev := eventcenter.CreateEvent(dbmodel.EvWarning, text, dbApp.Machine, dbApp, oldDmn)
						events = append(events, ev)
					}

					// check if daemon changed Version
					if dmn.Version != oldDmn.Version {
						text := fmt.Sprintf("{daemon} version changed from %s to %s",
							oldDmn.Version, dmn.Version)
						ev := eventcenter.CreateEvent(dbmodel.EvWarning, text, dbApp.Machine, dbApp, oldDmn)
						events = append(events, ev)
					}
					break
				}
			}
		}
	} else {
		for name := range daemonsMap {
			dmn := daemonsMap[name]
			// if all daemons are active then whole app is active
			newActive = newActive && dmn.Active

			// if this is new daemon and it is not active then disable its monitoring
			if !dmn.Active {
				dmn.Monitored = false
			}

			// if this is a new app then just add detected daemon
			newDaemons = append(newDaemons, dmn)
		}
	}

	// update app state
	dbApp.Active = newActive
	dbApp.Daemons = newDaemons

	return events
}

// Inserts or updates information about Kea app in the database. Next, it extracts
// Kea's configurations and uses to either update or create new shared networks,
// subnets and pools. Finally, the relations between the subnets and the Kea app
// are created. Note that multiple apps can be associated with the same subnet.
func CommitAppIntoDB(db *dbops.PgDB, app *dbmodel.App, eventCenter eventcenter.EventCenter, changeEvents []*dbmodel.Event) error {
	// Go over the shared networks and subnets stored in the Kea configuration
	// and match them with the existing entires in the database. If some of
	// the shared networks or subnets do not exist they are instantiated and
	// returned here.
	networks, subnets, err := DetectNetworks(db, app)
	if err != nil {
		err = errors.Wrapf(err, "unable to detect subnets and shared networks for Kea app with id %d", app.ID)
		return err
	}

	// Go over the global reservations stored in the Kea configuration and
	// match them with the existing global hosts.
	globalHosts, err := detectGlobalHostsFromConfig(db, app)
	if err != nil {
		err = errors.Wrapf(err, "unable to detect global host reservations for Kea app with id %d", app.ID)
		return err
	}

	// Begin transaction.
	tx, rollback, commit, err := dbops.Transaction(db)
	if err != nil {
		return err
	}
	defer rollback()

	newApp := false
	var addedDaemons, deletedDaemons []*dbmodel.Daemon
	if app.ID == 0 {
		// New app, insert it.
		addedDaemons, err = dbmodel.AddApp(tx, app)
		newApp = true
	} else {
		// Existing app, update it if needed.
		addedDaemons, deletedDaemons, err = dbmodel.UpdateApp(tx, app)
	}

	if err != nil {
		return err
	}

	if newApp {
		eventCenter.AddInfoEvent("added {app} on {machine}", app.Machine, app)
	}

	for _, dmn := range deletedDaemons {
		dmn.App = app
		eventCenter.AddInfoEvent("removed {daemon} from {app}", app.Machine, app, dmn)
	}
	for _, dmn := range addedDaemons {
		dmn.App = app
		eventCenter.AddInfoEvent("added {daemon} to {app}", app.Machine, app, dmn)
	}
	for _, ev := range changeEvents {
		eventCenter.AddEvent(ev)
	}

	// For the given app, iterate over the networks and subnets and update their
	// global instances accordingly in the database.
	addedSubnets, err := dbmodel.CommitNetworksIntoDB(tx, networks, subnets, app, 1)
	if err != nil {
		return err
	}
	if len(addedSubnets) > 0 {
		// add event per subnet only if there is not more than 10 subnets
		if len(addedSubnets) < 10 {
			for _, sn := range addedSubnets {
				eventCenter.AddInfoEvent("added {subnet} to {app}", app, sn)
			}
		}
		t := fmt.Sprintf("added %d subnets to {app}", len(addedSubnets))
		eventCenter.AddInfoEvent(t, app)
	}

	// For the given app, iterate over the global hosts and update their instances
	// in the database or insert them into the database.
	err = dbmodel.CommitGlobalHostsIntoDB(tx, globalHosts, app, "config", 1)
	if err != nil {
		return err
	}

	for _, daemon := range app.Daemons {
		// Check what HA services the daemon belongs to.
		services := DetectHAServices(db, daemon)

		// For the given daemon, iterate over the services and add/update them in the
		// database.
		err = dbmodel.CommitServicesIntoDB(tx, services, daemon)
		if err != nil {
			return err
		}
	}

	// Commit the changes if everything went fine.
	err = commit()
	return err
}
