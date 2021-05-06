import { Component, Input, OnInit } from '@angular/core'
import { DHCPService } from '../backend/api/api'
import { epochToLocal } from '../utils'

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
export class HostTabComponent implements OnInit {
    Usage = HostReservationUsage

    /**
     * Structure containing host information currently displayed.
     */
    currentHost: any

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

    /**
     * Structure used to build a field set with IP address reservations
     * and delegated prefix reservations from a template.
     */
    ipReservationsStatics = [
        {
            id: 'address-reservations-fieldset',
            legend: 'Address Reservations',
        },
        {
            id: 'prefix-reservations-fieldset',
            legend: 'Prefix Reservations',
        },
    ]

    /**
     * Component constructor.
     *
     * @param dhcpApi service used to communicate with the server over REST API.
     */
    constructor(private dhcpApi: DHCPService) {}

    /**
     * Lifecycle hook triggered during component initialization.
     *
     * Currently no-op.
     */
    ngOnInit(): void {}

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
                console.info(data)
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
                console.info(err)
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
        // If the usage haven't been set yet, or the new usage overrides the current
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
                summary =
                    'Found ' +
                    leaseInfo.leases.length +
                    ' lease' +
                    (m ? 's' : '') +
                    ' for this reservation' +
                    (m ? '. They include a lease' : '') +
                    ' which expired at ' +
                    epochToLocal(leaseInfo.culprit.cltt + leaseInfo.culprit.validLifetime) +
                    '.'
                return summary
            case this.Usage.Declined:
                // Found leases for our client but at least one of them is declined.
                summary =
                    'Found ' +
                    leaseInfo.leases.length +
                    ' lease' +
                    (m ? 's' : '') +
                    ' for this reservation' +
                    (m ? '. They include a declined lease with' : ' which is declined and has') +
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
                    'Found a lease with expiration time at ' +
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
     * Returns the leatest expiration time from the leases held in the
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
}
