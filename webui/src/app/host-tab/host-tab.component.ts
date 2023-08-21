import { Component, EventEmitter, Input, Output } from '@angular/core'

import { ConfirmationService, MessageService } from 'primeng/api'
import { Host, IPReservation, LocalHost } from '../backend'

import { DHCPService } from '../backend/api/api'
import {
    hasDifferentLocalHostBootFields,
    hasDifferentLocalHostClientClasses,
    hasDifferentLocalHostData,
} from '../hosts'
import { durationToString, epochToLocal, getErrorMessage } from '../utils'

enum HostReservationUsage {
    Conflicted = 1,
    Declined,
    Expired,
    Used,
}

/**
 * Component presenting reservation details for a selected host.
 *
 * It is embedded in the apps-page and used in cases when a user
 * selects one or more host reservations. If multiple host tabs are
 * opened, a single instance of this component is used to present
 * information associated with those tabs. Selecting a different
 * tab causes the change of the host input property. This also
 * triggers a REST API call to fetch leases for the reserved
 * addresses and prefixes, if they haven't been fetched yet for
 * the given host. The lease information is cached for all opened
 * tabs. A user willing to refresh the cached lease information must
 * click the refresh button on the selected tab. The lease information
 * is used to present whether the reserved IP address or prefix is in
 * use and whether the lease is assigned to a client which does not
 * have a reservation for it (conflict).
 */
@Component({
    selector: 'app-host-tab',
    templateUrl: './host-tab.component.html',
    styleUrls: ['./host-tab.component.sass'],
})
export class HostTabComponent {
    /**
     * An event emitter notifying a parent that user has clicked the
     * Edit button to modify the host reservation.
     */
    @Output() hostEditBegin = new EventEmitter<any>()

    /**
     * An event emitter notifying a parent that user has clicked the
     * Delete button to delete the host reservation.
     */
    @Output() hostDelete = new EventEmitter<any>()

    Usage = HostReservationUsage

    /**
     * Structure containing host information currently displayed.
     */
    currentHost: Host

    /**
     * Indicates if the boot fields panel should be displayed.
     */
    displayBootFields: boolean

    /**
     * A map caching leases for various hosts.
     *
     * Thanks to this caching, it is possible to switch between the
     * host tabs and avoid fetching lease information for a current
     * host whenever a different tab is selected.
     */
    private _leasesForHosts = new Map()

    /**
     * Leases fetched for currently selected tab (host).
     */
    currentLeases: any

    /**
     * A map of booleans indicating for which hosts leases search
     * is in progress.
     */
    private _leasesSearchStatus = new Map()

    /**
     * List of Kea apps which returned an error during leases search.
     */
    erredApps = []

    hostDeleted = false

    /**
     * Component constructor.
     *
     * @param msgService service displaying error messages upon a communication
     *                   error with the server.
     * @param dhcpApi service used to communicate with the server over REST API.
     */
    constructor(
        private dhcpApi: DHCPService,
        private confirmService: ConfirmationService,
        private msgService: MessageService
    ) {}

    /**
     * Returns information about currently selected host.
     */
    @Input()
    get host() {
        return this.currentHost
    }

    /**
     * Sets a host to be displayed by the component.
     *
     * This setter is called when the user selects one of the host tabs.
     * If leases for this host have not been already gathered, the function
     * queries the server for the leases corresponding to the host. Otherwise,
     * cached lease information is displayed.
     *
     * @param host host information.
     */
    set host(host) {
        // Make the new host current.
        this.currentHost = host
        // The host is null if the tab with a list of hosts is selected.
        if (!this.currentHost) {
            return
        }
        // Check if we already have lease information for this host.
        const leasesForHost = this._leasesForHosts.get(host.id)
        if (leasesForHost) {
            // We have the lease information already. Let's use it.
            this.currentLeases = leasesForHost
        } else {
            // We don't have lease information for this host. Need to
            // fetch it from Kea servers via Stork server.
            this._fetchLeases(host.id)
        }
        this.displayBootFields = !!this.currentHost.localHosts?.some(
            (lh) => lh.nextServer || lh.serverHostname || lh.bootFileName
        )
    }

    /**
     * Returns all IP host reservations (addresses and prefixes).
     */
    get ipReservations(): Array<IPReservation> {
        let reservations: Array<IPReservation> = []
        if (this.host.addressReservations) {
            reservations.push(...this.host.addressReservations)
        }
        if (this.host.prefixReservations) {
            reservations.push(...this.host.prefixReservations)
        }
        return reservations
    }

    /**
     * Returns boolean value indicating if the leases are being searched
     * for the currently displayed host.
     *
     * @returns true if leases are being searched for the currently displayed
     *          host, false otherwise.
     */
    get leasesSearchInProgress() {
        return this._leasesSearchStatus.get(this.host.id) ? true : false
    }

    /**
     * Returns local host grouped by the app ID. 
     */
    get localHostsByAppId(): LocalHost[] {
        const localHostsByAppId = {}
        this.host.localHosts?.forEach((localHost) => {
            if (!localHostsByAppId[localHost.appId]) {
                localHostsByAppId[localHost.appId] = []
            }
            localHostsByAppId[localHost.appId].push(localHost)
        })
        return Object.values(localHostsByAppId)
    }

    /**
     * Fetches leases for the given host from the Stork server.
     *
     * @param hostId host identifier.
     */
    private _fetchLeases(hostId) {
        // Do not search again if the search is already in progress.
        if (this._leasesSearchStatus.get(hostId)) {
            return
        }
        // Indicate that the search is already in progress for that host.
        this._leasesSearchStatus.set(hostId, true)
        this.erredApps = []
        this.dhcpApi.getLeases(undefined, hostId).subscribe(
            (data) => {
                // Finished searching the leases.
                this._leasesSearchStatus.set(hostId, false)
                // Collect the lease information and store it in the cache.
                const leases = new Map()
                if (data.items) {
                    for (const lease of data.items) {
                        this._mergeLease(leases, data.conflicts, lease)
                    }
                }
                this.erredApps = data.erredApps
                this._leasesForHosts.set(hostId, leases)
                this.currentLeases = leases
            },
            (err) => {
                // Finished searching the leases.
                this._leasesSearchStatus.set(hostId, false)
                const msg = getErrorMessage(err)
                this.msgService.add({
                    severity: 'error',
                    summary: 'Error searching leases for the host',
                    detail: 'Error searching by host ID ' + hostId + ': ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Merges lease information into the cache.
     *
     * The cache contains leases gathered for various IP addresses and/or
     * delegated prefixes for a host. There may be multiple leases for a
     * single reserved IP address or delegated prefix.
     *
     * @param leases leases cache for the host.
     * @param conflicts array of conflicting lease ids.
     * @param newLease a lease to be merged to the cache.
     */
    private _mergeLease(leases, conflicts, newLease) {
        // Check if the lease is in conflict with the host reservation.
        if (conflicts) {
            for (const conflictId of conflicts) {
                if (newLease.id === conflictId) {
                    newLease['conflict'] = true
                }
            }
        }
        let reservedLeaseInfo = leases.get(newLease.ipAddress)
        if (reservedLeaseInfo) {
            // There is already some lease cached for this IP address.
            reservedLeaseInfo.leases.push(newLease)
        } else {
            // There is no lease cached for this IP address yet.
            reservedLeaseInfo = { leases: [newLease] }
            let leaseKey = newLease.ipAddress
            if (newLease.prefixLength && newLease.prefixLength !== 0 && newLease.prefixLength !== 128) {
                leaseKey += '/' + newLease.prefixLength
            }
            leases.set(leaseKey, reservedLeaseInfo)
        }
        let newUsage = this.Usage.Used
        if (newLease['conflict']) {
            newUsage = this.Usage.Conflicted
        } else {
            // Depending on the lease state, we should adjust the usage information
            // displayed next to the IP reservation.
            switch (newLease['state']) {
                case 0:
                    newUsage = this.Usage.Used
                    break
                case 1:
                    newUsage = this.Usage.Declined
                    break
                case 2:
                    newUsage = this.Usage.Expired
                    break
                default:
                    break
            }
        }
        // If the usage hasn't been set yet, or the new usage overrides the current
        // usage, update the usage information. The usage values in the enum are ordered
        // by importance: conflicted, declined, expired, used. Even if one of the leases
        // is used, but the other one is conflicted, we mark the usage as conflicted.
        if (!reservedLeaseInfo['usage'] || newUsage < reservedLeaseInfo['usage']) {
            reservedLeaseInfo['usage'] = newUsage
            reservedLeaseInfo['culprit'] = newLease
        }
    }

    /**
     * Returns lease usage text for an enum value.
     *
     * @param usage usage enum indicating if the lease is used, declined, expired
     *        or conflicted.
     * @returns usage text displayed next to the IP address or delegated prefix.
     */
    getLeaseUsageText(usage) {
        switch (usage) {
            case this.Usage.Used:
                return 'in use'
            case this.Usage.Declined:
                return 'declined'
            case this.Usage.Expired:
                return 'expired'
            case this.Usage.Conflicted:
                return 'in conflict'
            default:
                break
        }
        return 'unused'
    }

    /**
     * Returns information displayed in the expanded row for an IP reservation.
     *
     * When user clicks on a chevron icon next to the reserved IP address, a row
     * is expanded displaying the lease state summary for the given IP address.
     * The summary is generated by this function.
     *
     * @param leaseInfo lease information for the specified IP address or
     *        delegated prefix.
     * @returns lease state summary text.
     */
    getLeaseSummary(leaseInfo) {
        let summary = 'Lease information unavailable.'
        if (!leaseInfo.culprit) {
            return summary
        }
        const m = leaseInfo.leases.length > 1
        switch (leaseInfo.usage) {
            case this.Usage.Used:
                // All leases are assigned to the client who has a reservation
                // for it. Simply say how many leases are assigned and when they
                // expire.
                summary =
                    'Found ' +
                    leaseInfo.leases.length +
                    ' assigned lease' +
                    (m ? 's' : '') +
                    ' with the' +
                    (m ? ' latest' : '') +
                    ' expiration time at ' +
                    epochToLocal(this._getLatestLeaseExpirationTime(leaseInfo)) +
                    '.'
                return summary
            case this.Usage.Expired:
                const expirationTime = leaseInfo.culprit.cltt + leaseInfo.culprit.validLifetime
                const expirationDuration = durationToString(new Date().getTime() / 1000 - expirationTime, true)
                summary =
                    'Found ' +
                    leaseInfo.leases.length +
                    ' lease' +
                    (m ? 's' : '') +
                    ' for this reservation' +
                    (m ? '. This includes a lease' : '') +
                    ' that expired at ' +
                    epochToLocal(expirationTime)

                if (expirationDuration) {
                    summary += ' ' + '(' + expirationDuration + ' ago).'
                }
                return summary
            case this.Usage.Declined:
                // Found leases for our client but at least one of them is declined.
                summary =
                    'Found ' +
                    leaseInfo.leases.length +
                    ' lease' +
                    (m ? 's' : '') +
                    ' for this reservation' +
                    (m ? '. This includes a declined lease with' : ' which is declined and has an') +
                    ' expiration time at ' +
                    epochToLocal(leaseInfo.culprit.cltt + leaseInfo.culprit.validLifetime) +
                    '.'
                return summary
            case this.Usage.Conflicted:
                // Found lease assignments to other clients than the one which
                // has a reservation.
                let identifier = ''
                if (leaseInfo.culprit.hwAddress) {
                    identifier = 'MAC address=' + leaseInfo.culprit.hwAddress
                } else if (leaseInfo.culprit.duid) {
                    identifier = 'DUID=' + leaseInfo.culprit.duid
                } else if (leaseInfo.culprit.clientId) {
                    identifier = 'client-id=' + leaseInfo.culprit.clientId
                }
                summary =
                    'Found a lease with an expiration time at ' +
                    epochToLocal(leaseInfo.culprit.cltt + leaseInfo.culprit.validLifetime) +
                    ' assigned to the client with ' +
                    identifier +
                    ', for which it was not reserved.'
                return summary
            default:
                break
        }
        return summary
    }

    /**
     * Returns the latest expiration time from the leases held in the
     * cache for the particular IP address or delegated prefix.
     *
     * @param leaseInfo lease information for a reserved IP address or
     *        delegated prefix.
     *
     * @returns expiration time relative to the epoch or 0 if no lease
     *          is present.
     */
    private _getLatestLeaseExpirationTime(leaseInfo) {
        let latestExpirationTime = 0
        for (const lease of leaseInfo.leases) {
            const expirationTime = lease.cltt + lease.validLifetime
            if (expirationTime > latestExpirationTime) {
                latestExpirationTime = expirationTime
            }
        }
        return latestExpirationTime
    }

    /**
     * Starts leases refresh for a current host.
     */
    refreshLeases() {
        this._fetchLeases(this.host.id)
    }

    /**
     * Event handler called when user begins host editing.
     *
     * It emits an event to the parent component to notify that host is
     * is now edited.
     */
    onHostEditBegin(): void {
        this.hostEditBegin.emit(this.host)
    }

    /*
     * Displays a dialog to confirm host deletion.
     */
    confirmDeleteHost() {
        this.confirmService.confirm({
            message: 'Are you sure that you want to permanently delete this host reservation?',
            header: 'Delete Host',
            icon: 'pi pi-exclamation-triangle',
            accept: () => {
                this.deleteHost()
            },
        })
    }

    /**
     * Sends a request to the server to delete the host reservation.
     */
    deleteHost() {
        // Disable the button for deleting the host to prevent pressing the
        // button multiple times and sending multiple requests.
        this.hostDeleted = true
        this.dhcpApi
            .deleteHost(this.host.id)
            .toPromise()
            .then((data) => {
                // Re-enable the delete button.
                this.hostDeleted = false
                this.msgService.add({
                    severity: 'success',
                    summary: 'Host reservation successfully deleted',
                })
                // Notify the parent that the host was deleted and the tab can be closed.
                this.hostDelete.emit(this.host)
            })
            .catch((err) => {
                // Re-enable the delete button.
                this.hostDeleted = false
                // Issues with deleting the host.
                const msg = getErrorMessage(err)
                this.msgService.add({
                    severity: 'error',
                    summary: 'Cannot delete the host',
                    detail: 'Failed to delete the host host: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Checks if all provided DHCP servers have equal set of DHCP options.
     */
    daemonsHaveEqualDhcpOptions(localHosts: LocalHost[]): boolean {
        return !hasDifferentLocalHostData(localHosts)
    }

    /**
     * Checks if all DHCP servers owning the reservation have equal set of
     * DHCP options.
     *
     * @returns true, if all DHCP servers have equal option set hashes, false
     *          otherwise.
     */
    allDaemonsHaveEqualDhcpOptions(): boolean {
        return this.daemonsHaveEqualDhcpOptions(this.host.localHosts)
    }

    /**
     * Checks if all DHCP servers owning the reservation have equal set of
     * client classes.
     *
     * @returns true, if all DHCP servers have equal set of client classes.
     */
    allDaemonsHaveEqualClientClasses(): boolean {
        return !hasDifferentLocalHostClientClasses(this.host.localHosts)
    }

    /**
     * Checks if all DHCP servers owning the reservation have equal set of
     * boot fields, i.e. next server, server hostname, boot file name.
     *
     * @returns true if, all DHCP servers have equal set of boot fields.
     */
    allDaemonsHaveEqualBootFields(): boolean {
        return !hasDifferentLocalHostBootFields(this.host.localHosts)
    }
}
