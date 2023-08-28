package restservice

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/apps/kea"
	"isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	storkutil "isc.org/stork/util"
)

// Converts host reservation fetched from the database to the format
// used in REST API.
func (r *RestAPI) convertFromHost(dbHost *dbmodel.Host) *models.Host {
	host := &models.Host{
		ID:       dbHost.ID,
		SubnetID: dbHost.SubnetID,
		Hostname: dbHost.Hostname,
	}
	// Include subnet prefix if this is subnet specific host.
	if dbHost.Subnet != nil {
		host.SubnetPrefix = dbHost.Subnet.Prefix
	}
	// Convert DHCP host identifiers.
	for _, dbHostID := range dbHost.HostIdentifiers {
		hostID := models.HostIdentifier{
			IDType:     dbHostID.Type,
			IDHexValue: dbHostID.ToHex(":"),
		}
		host.HostIdentifiers = append(host.HostIdentifiers, &hostID)
	}
	// Convert IP reservations.
	for _, dbHostIP := range dbHost.IPReservations {
		parsedIP := storkutil.ParseIP(dbHostIP.Address)
		if parsedIP == nil {
			continue
		}
		hostIP := models.IPReservation{
			Address: parsedIP.NetworkAddress,
		}
		if parsedIP.Prefix {
			host.PrefixReservations = append(host.PrefixReservations, &hostIP)
		} else {
			host.AddressReservations = append(host.AddressReservations, &hostIP)
		}
	}
	// Append local hosts containing associations of the host with
	// daemons and DHCP options.
	for _, dbLocalHost := range dbHost.LocalHosts {
		localHost := models.LocalHost{
			AppID:          dbLocalHost.Daemon.AppID,
			AppName:        dbLocalHost.Daemon.App.Name,
			DaemonID:       dbLocalHost.Daemon.ID,
			DataSource:     dbLocalHost.DataSource.String(),
			NextServer:     dbLocalHost.NextServer,
			ServerHostname: dbLocalHost.ServerHostname,
			BootFileName:   dbLocalHost.BootFileName,
			ClientClasses:  dbLocalHost.ClientClasses,
			OptionsHash:    dbLocalHost.DHCPOptionSet.Hash,
		}
		localHost.Options = r.unflattenDHCPOptions(dbLocalHost.DHCPOptionSet.Options, "", 0)
		host.LocalHosts = append(host.LocalHosts, &localHost)
	}
	return host
}

// Convert host reservation from the format used in REST API to a
// database host representation.
func (r *RestAPI) convertToHost(restHost *models.Host) (*dbmodel.Host, error) {
	host := &dbmodel.Host{
		ID:       restHost.ID,
		SubnetID: restHost.SubnetID,
		Hostname: restHost.Hostname,
	}
	// Convert DHCP host identifiers.
	for _, hid := range restHost.HostIdentifiers {
		hostID := dbmodel.HostIdentifier{
			Type:  hid.IDType,
			Value: storkutil.HexToBytes(hid.IDHexValue),
		}
		host.HostIdentifiers = append(host.HostIdentifiers, hostID)
	}
	// Convert IP reservations.
	for _, r := range append(restHost.PrefixReservations, restHost.AddressReservations...) {
		ipr := dbmodel.IPReservation{
			Address: r.Address,
		}
		host.IPReservations = append(host.IPReservations, ipr)
	}
	// Convert local hosts containing associations of the host with daemons.
	for _, lh := range restHost.LocalHosts {
		var ds dbmodel.HostDataSource
		var err error

		if lh.DataSource == "" {
			return nil, errors.Errorf("missing local host data source")
		} else if ds, err = dbmodel.ParseHostDataSource(lh.DataSource); err != nil {
			return nil, errors.WithMessage(err, "invalid local host data source")
		}

		localHost := dbmodel.LocalHost{
			DaemonID:       lh.DaemonID,
			DataSource:     ds,
			ClientClasses:  lh.ClientClasses,
			NextServer:     lh.NextServer,
			ServerHostname: lh.ServerHostname,
			BootFileName:   lh.BootFileName,
		}
		options, err := r.flattenDHCPOptions("", lh.Options, 0)
		if err != nil {
			return nil, err
		}
		localHost.DHCPOptionSet.SetDHCPOptions(options)
		host.SetLocalHost(&localHost)
	}
	return host, nil
}

// Fetches host reservations from the database and converts to the data formats
// used in REST API.
func (r *RestAPI) getHosts(offset, limit, appID int64, subnetID *int64, localSubnetID *int64, filterText *string, global *bool, sortField string, sortDir dbmodel.SortDirEnum) (*models.Hosts, error) {
	// Get the hosts from the database.
	filters := dbmodel.HostsByPageFilters{
		AppID:         &appID,
		SubnetID:      subnetID,
		LocalSubnetID: localSubnetID,
		FilterText:    filterText,
		Global:        global,
	}
	dbHosts, total, err := dbmodel.GetHostsByPage(r.DB, offset, limit, filters, sortField, sortDir)
	if err != nil {
		return nil, err
	}

	hosts := &models.Hosts{
		Total: total,
	}

	// Convert hosts fetched from the database to REST.
	for i := range dbHosts {
		host := r.convertFromHost(&dbHosts[i])
		hosts.Items = append(hosts.Items, host)
	}

	return hosts, nil
}

// Get list of hosts with specifying an offset and a limit. The hosts can be fetched
// for a given subnet and with filtering by search text.
func (r *RestAPI) GetHosts(ctx context.Context, params dhcp.GetHostsParams) middleware.Responder {
	var start int64
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	var appID int64
	if params.AppID != nil {
		appID = *params.AppID
	}

	// get hosts from db
	hosts, err := r.getHosts(start, limit, appID, params.SubnetID, params.LocalSubnetID, params.Text, params.Global, "", dbmodel.SortDirAny)
	if err != nil {
		msg := "Problem fetching hosts from the database"
		log.Error(err)
		rsp := dhcp.NewGetHostsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Everything fine.
	rsp := dhcp.NewGetHostsOK().WithPayload(hosts)
	return rsp
}

// Get a host by ID.
func (r *RestAPI) GetHost(ctx context.Context, params dhcp.GetHostParams) middleware.Responder {
	// Find a host in the database.
	dbHost, err := dbmodel.GetHost(r.DB, params.ID)
	if err != nil {
		// Error while communicating with the database.
		msg := fmt.Sprintf("Problem fetching host reservation with ID %d from db", params.ID)
		log.Error(err)
		rsp := dhcp.NewGetHostDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbHost == nil {
		// Host not found.
		msg := fmt.Sprintf("Cannot find host reservation with ID %d", params.ID)
		rsp := dhcp.NewGetHostDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Host found. Convert it to the format used in REST API.
	host := r.convertFromHost(dbHost)
	rsp := dhcp.NewGetHostOK().WithPayload(host)
	return rsp
}

// Common function for executed when creating a new transaction for when the
// host is created or updated. It fetches available DHCP daemons, subnets and
// client classes. It also creates transaction context. If an error occurs,
// an http error code and message are returned.
func (r *RestAPI) commonCreateOrUpdateHostBegin(ctx context.Context) ([]*models.KeaDaemon, []*models.Subnet, []string, context.Context, int, string) {
	// A list of Kea DHCP daemons will be needed in the user form,
	// so the user can select which servers send the reservation to.
	daemons, err := dbmodel.GetKeaDHCPDaemons(r.DB)
	if err != nil {
		msg := "problem with fetching Kea daemons from the database"
		log.WithError(err).Error(msg)
		return nil, nil, nil, nil, http.StatusInternalServerError, msg
	}
	// Convert daemons list to REST API format and extract their configured
	// client classes.
	respDaemons := []*models.KeaDaemon{}
	respClientClasses := []string{}
	clientClassesMap := make(map[string]bool)
	for i := range daemons {
		if daemons[i].KeaDaemon != nil && daemons[i].KeaDaemon.Config != nil {
			// Filter the daemons with host_cmds hook library.
			if _, _, exists := daemons[i].KeaDaemon.Config.GetHookLibrary("libdhcp_host_cmds"); exists {
				respDaemons = append(respDaemons, keaDaemonToRestAPI(&daemons[i]))
			}
			clientClasses := daemons[i].KeaDaemon.Config.GetClientClasses()
			for _, c := range clientClasses {
				clientClassesMap[c.Name] = true
			}
		}
	}
	// Turn the class map to a slice and sort it by a class name.
	for c := range clientClassesMap {
		respClientClasses = append(respClientClasses, c)
	}
	sort.Strings(respClientClasses)

	// If there are no daemons with host_cmds hooks library loaded there is no way
	// to add new host reservation. In that case, we don't begin a transaction.
	if len(respDaemons) == 0 {
		msg := "unable to begin transaction for adding new host because there are no Kea servers with host_cmds hooks library available"
		log.Error(msg)
		return nil, nil, nil, nil, http.StatusBadRequest, msg
	}
	// Host reservations are typically associated with subnets. The
	// user needs a current list of available subnets.
	subnets, err := dbmodel.GetAllSubnets(r.DB, 0)
	if err != nil {
		msg := "problem with fetching subnets from the database"
		log.WithError(err).Error(msg)
		return nil, nil, nil, nil, http.StatusInternalServerError, msg
	}
	// Convert subnets list to REST API format.
	respSubnets := []*models.Subnet{}
	for i := range subnets {
		respSubnets = append(respSubnets, r.subnetToRestAPI(&subnets[i]))
	}
	// Get the logged user's ID.
	ok, user := r.SessionManager.Logged(ctx)
	if !ok {
		msg := "unable to begin transaction because user is not logged in"
		log.Error("Problem with creating transaction context because user has no session")
		return nil, nil, nil, nil, http.StatusForbidden, msg
	}
	// Create configuration context.
	cctx, err := r.ConfigManager.CreateContext(int64(user.ID))
	if err != nil {
		msg := "problem with creating transaction context"
		log.WithError(err).Error(msg)
		return nil, nil, nil, nil, http.StatusInternalServerError, msg
	}
	return respDaemons, respSubnets, respClientClasses, cctx, 0, ""
}

// Implements the POST call to create new transaction for adding a new host
// reservation (hosts/new/transaction).
func (r *RestAPI) CreateHostBegin(ctx context.Context, params dhcp.CreateHostBeginParams) middleware.Responder {
	// Execute the common part between create and update operations. It retrieves,
	// daemons, subnets, client classes and creates the transaction context.
	respDaemons, respSubnets, respClientClasses, cctx, code, msg := r.commonCreateOrUpdateHostBegin(ctx)
	if code != 0 {
		// Error case.
		rsp := dhcp.NewCreateHostBeginDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Begin host add transaction.
	var err error
	if cctx, err = r.ConfigManager.GetKeaModule().BeginHostAdd(cctx); err != nil {
		msg := "problem with initializing transaction for host creating new host"
		log.Error(msg)
		rsp := dhcp.NewCreateHostBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Retrieve the generated context ID.
	cctxID, ok := config.GetValueAsInt64(cctx, config.ContextIDKey)
	if !ok {
		msg := "problem with retrieving context ID for a transaction"
		log.Error(msg)
		rsp := dhcp.NewCreateHostBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Remember the context, i.e. new transaction has been successfully created.
	_ = r.ConfigManager.RememberContext(cctx, time.Minute*10)

	// Return transaction ID, apps and subnets to the user.
	contents := &models.CreateHostBeginResponse{
		ID:            cctxID,
		Daemons:       respDaemons,
		Subnets:       respSubnets,
		ClientClasses: respClientClasses,
	}
	rsp := dhcp.NewCreateHostBeginOK().WithPayload(contents)
	return rsp
}

// Common function that implements the POST calls to apply and commit a new
// or updated reservation. The ctx parameter is the REST API context. The
// transactionID is the identifier of the current configuration transaction
// used by the function to recover the transaction context. The restHost is
// the pointer to the host reservation specified by the user. It is converted
// by this function to the database model. The applyFunc is the function of
// of the Kea config module that applies the specified reservation. It is
// one of the ApplyHostAdd or ApplyHostUpdate, depending on whether the
// new host is created (via CreateHostSubmit) or updated (via UpdateHostSubmit).
// The apply functions receive the transaction context and a pointer to the
// host reservation. They return the updated context and error. This function
// returns the HTTP error code if an error occurs or 0 when there is no error.
// In addition it returns an error string to be included in the HTTP response
// or an empty string if there is no error.
func (r *RestAPI) commonCreateOrUpdateHostSubmit(ctx context.Context, transactionID int64, restHost *models.Host, applyFunc func(context.Context, *dbmodel.Host) (context.Context, error)) (int, string) {
	// Make sure that the host information is present.
	if restHost == nil {
		msg := "host information not specified"
		log.Errorf(msg)
		return http.StatusBadRequest, msg
	}
	// Get the user ID and recover the transaction context.
	ok, user := r.SessionManager.Logged(ctx)
	if !ok {
		msg := "unable to submit because user is not logged in"
		log.Error("Problem with recovering transaction context because user has no session")
		return http.StatusForbidden, msg
	}
	// Retrieve the context from the config manager.
	cctx, _ := r.ConfigManager.RecoverContext(transactionID, int64(user.ID))
	if cctx == nil {
		msg := "transaction expired"
		log.Errorf("Problem with recovering transaction context for transaction ID %d and user ID %d", transactionID, user.ID)
		return http.StatusNotFound, msg
	}

	// Convert host information from REST API to database format.
	host, err := r.convertToHost(restHost)
	if err != nil {
		msg := "error parsing specified host reservation"
		log.WithError(err).Error(msg)
		return http.StatusBadRequest, msg
	}
	err = host.PopulateDaemons(r.DB)
	if err != nil {
		msg := "specified host is associated with daemons that no longer exist"
		log.WithError(err).Error(msg)
		return http.StatusNotFound, msg
	}
	err = host.PopulateSubnet(r.DB)
	if err != nil {
		msg := "problem with retrieving subnet association with the host"
		log.WithError(err).Error(msg)
		return http.StatusInternalServerError, msg
	}
	// Apply the host information (create Kea commands).
	cctx, err = applyFunc(cctx, host)
	if err != nil {
		msg := "problem with applying host information"
		log.WithError(err).Error(msg)
		return http.StatusInternalServerError, msg
	}
	// Send the commands to Kea servers.
	cctx, err = r.ConfigManager.Commit(cctx)
	if err != nil {
		msg := fmt.Sprintf("problem with committing host information: %s", err)
		log.WithError(err).Error(msg)
		return http.StatusConflict, msg
	}
	// Everything ok. Cleanup and send OK to the client.
	r.ConfigManager.Done(cctx)
	return 0, ""
}

// Implements the POST call to apply and commit host reservation (hosts/new/transaction/{id}/submit).
func (r *RestAPI) CreateHostSubmit(ctx context.Context, params dhcp.CreateHostSubmitParams) middleware.Responder {
	if code, msg := r.commonCreateOrUpdateHostSubmit(ctx, params.ID, params.Host, r.ConfigManager.GetKeaModule().ApplyHostAdd); code != 0 {
		// Error case.
		rsp := dhcp.NewCreateHostSubmitDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := dhcp.NewCreateHostSubmitOK()
	return rsp
}

// Common function that implements the DELETE calls to cancel adding new
// or updating a host reservation. It removes the specified transaction
// from the config manager, if the transaction exists. It  returns the
// HTTP error code if an error occurs or 0 when there is no error.
// In addition it returns an error string to be included in the HTTP response
// or an empty string if there is no error.
func (r *RestAPI) commonCreateOrUpdateHostDelete(ctx context.Context, transactionID int64) (int, string) {
	// Get the user ID and recover the transaction context.
	ok, user := r.SessionManager.Logged(ctx)
	if !ok {
		msg := "unable to cancel transaction because user is not logged in"
		log.Error("Problem with recovering transaction context because user has no session")
		return http.StatusForbidden, msg
	}
	// Retrieve the context from the config manager.
	cctx, _ := r.ConfigManager.RecoverContext(transactionID, int64(user.ID))
	if cctx == nil {
		msg := "transaction expired"
		log.Errorf("Problem with recovering transaction context for transaction ID %d and user ID %d", transactionID, user.ID)
		return http.StatusNotFound, msg
	}
	r.ConfigManager.Done(cctx)
	return 0, ""
}

// Implements the DELETE call to cancel adding new reservation (hosts/new/transaction/{id}). It
// removes the specified transaction from the config manager, if the transaction exists.
func (r *RestAPI) CreateHostDelete(ctx context.Context, params dhcp.CreateHostDeleteParams) middleware.Responder {
	if code, msg := r.commonCreateOrUpdateHostDelete(ctx, params.ID); code != 0 {
		// Error case.
		rsp := dhcp.NewCreateHostDeleteDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := dhcp.NewCreateHostDeleteOK()
	return rsp
}

// Implements the POST call to create new transaction for updating an
// existing host reservation (hosts/{hostId}/transaction).
func (r *RestAPI) UpdateHostBegin(ctx context.Context, params dhcp.UpdateHostBeginParams) middleware.Responder {
	// Execute the common part between create and update operations. It retrieves,
	// daemons, subnets, client classes and creates the transaction context.
	respDaemons, respSubnets, respClientClasses, cctx, code, msg := r.commonCreateOrUpdateHostBegin(ctx)
	if code != 0 {
		// Error case.
		rsp := dhcp.NewUpdateHostBeginDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Begin host update transaction. It retrieves current host information and
	// locks demons for updates.
	var err error
	cctx, err = r.ConfigManager.GetKeaModule().BeginHostUpdate(cctx, params.HostID)
	if err != nil {
		var hostNotFound *config.HostNotFoundError
		switch {
		case errors.As(err, &hostNotFound):
			// Failed to find host.
			msg := err.Error()
			log.WithError(err).Error("Failed to find host")
			rsp := dhcp.NewUpdateHostBeginDefault(http.StatusBadRequest).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		case errors.Is(err, config.ErrLock):
			// Failed to lock daemons.
			msg := err.Error()
			log.WithError(err).Error("Failed to lock daemons for host update")
			rsp := dhcp.NewUpdateHostBeginDefault(http.StatusLocked).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		default:
			// Other error.
			msg := "problem with initializing transaction for host update"
			log.WithError(err).Error(msg)
			rsp := dhcp.NewUpdateHostBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	}
	state, _ := config.GetTransactionState[kea.ConfigRecipe](cctx)
	host := state.Updates[0].Recipe.HostBeforeUpdate

	// Retrieve the generated context ID.
	cctxID, ok := config.GetValueAsInt64(cctx, config.ContextIDKey)
	if !ok {
		msg := "problem with retrieving context ID for a transaction"
		log.Error(msg)
		rsp := dhcp.NewUpdateHostBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Remember the context, i.e. new transaction has been successfully created.
	_ = r.ConfigManager.RememberContext(cctx, time.Minute*10)

	// Return transaction ID, apps and subnets to the user.
	contents := &models.UpdateHostBeginResponse{
		ID:            cctxID,
		Host:          r.convertFromHost(host),
		Daemons:       respDaemons,
		Subnets:       respSubnets,
		ClientClasses: respClientClasses,
	}
	rsp := dhcp.NewUpdateHostBeginOK().WithPayload(contents)
	return rsp
}

// Implements the POST call and commit an updated host reservation (hosts/{hostId}/transaction/{id}/submit).
func (r *RestAPI) UpdateHostSubmit(ctx context.Context, params dhcp.UpdateHostSubmitParams) middleware.Responder {
	if code, msg := r.commonCreateOrUpdateHostSubmit(ctx, params.ID, params.Host, r.ConfigManager.GetKeaModule().ApplyHostUpdate); code != 0 {
		// Error case.
		rsp := dhcp.NewUpdateHostSubmitDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := dhcp.NewUpdateHostSubmitOK()
	return rsp
}

// Implements the DELETE call to cancel updating host reservation (hosts/{hostId}/transaction/{id}).
// It removes the specified transaction from the config manager, if the transaction exists.
func (r *RestAPI) UpdateHostDelete(ctx context.Context, params dhcp.UpdateHostDeleteParams) middleware.Responder {
	if code, msg := r.commonCreateOrUpdateHostDelete(ctx, params.ID); code != 0 {
		// Error case.
		rsp := dhcp.NewUpdateHostDeleteDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := dhcp.NewUpdateHostDeleteOK()
	return rsp
}

// Implements the DELETE call for a host reservation (hosts/{id}). It sends suitable commands
// to the Kea servers owning the reservation. Deleting host reservation is not transactional.
// It could be implemented as a transaction with first REST API call ensuring that the host
// reservation still exists in Stork database and locking configuration changes for the daemons
// owning the reservation. However, it seems to be too much overhead with little gain. If the
// reservation doesn't exist this call will return an error anyway.
func (r *RestAPI) DeleteHost(ctx context.Context, params dhcp.DeleteHostParams) middleware.Responder {
	dbHost, err := dbmodel.GetHost(r.DB, params.ID)
	if err != nil {
		// Error while communicating with the database.
		msg := fmt.Sprintf("Problem fetching host reservation with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := dhcp.NewDeleteHostDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbHost == nil {
		// Host not found.
		msg := fmt.Sprintf("Cannot find host reservation with ID %d", params.ID)
		rsp := dhcp.NewDeleteHostDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Get the logged user's ID.
	ok, user := r.SessionManager.Logged(ctx)
	if !ok {
		msg := "unable to begin transaction because user is not logged in"
		log.Error("Problem with creating transaction context because user has no session")
		rsp := dhcp.NewDeleteHostDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Create configuration context.
	cctx, err := r.ConfigManager.CreateContext(int64(user.ID))
	if err != nil {
		msg := "problem with creating transaction context"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewDeleteHostDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Create Kea commands to delete host reservation.
	cctx, err = r.ConfigManager.GetKeaModule().ApplyHostDelete(cctx, dbHost)
	if err != nil {
		msg := "problem with preparing commands for deleting host reservation"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewDeleteHostDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Send the commands to Kea servers.
	_, err = r.ConfigManager.Commit(cctx)
	if err != nil {
		msg := fmt.Sprintf("problem with deleting host reservation: %s", err)
		log.WithError(err).Error(msg)
		rsp := dhcp.NewDeleteHostDefault(http.StatusConflict).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Send OK to the client.
	rsp := dhcp.NewDeleteHostOK()
	return rsp
}
