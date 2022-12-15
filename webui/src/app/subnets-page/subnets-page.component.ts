import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'

import { Table } from 'primeng/table'

import { DHCPService } from '../backend/api/api'
import { humanCount, getGrafanaUrl, extractKeyValsAndPrepareQueryParams, getGrafanaSubnetTooltip, getErrorMessage } from '../utils'
import { getTotalAddresses, getAssignedAddresses, parseSubnetsStatisticValues } from '../subnets'
import { SettingService } from '../setting.service'
import { concat, of, Subscription } from 'rxjs'
import { filter, map, take } from 'rxjs/operators'
import { Subnet } from '../backend'
import { MenuItem, MessageService } from 'primeng/api'

/**
 * Component for presenting DHCP subnets.
 */
@Component({
    selector: 'app-subnets-page',
    templateUrl: './subnets-page.component.html',
    styleUrls: ['./subnets-page.component.sass'],
})
export class SubnetsPageComponent implements OnInit, OnDestroy {
    private subscriptions = new Subscription()
    breadcrumbs = [{ label: 'DHCP' }, { label: 'Subnets' }]

    @ViewChild('subnetsTable') subnetsTable: Table

    // subnets
    subnets: Subnet[] = []
    totalSubnets = 0

    // filters
    filterText = ''
    queryParams = {
        text: null,
        dhcpVersion: null,
        appId: null,
    }

    // Tab menu

    /**
     * Array of tab menu items with subnet information.
     *
     * The first tab is always present and displays the subnet list.
     * 
     * Note: we cannot use the URL with no segment for the list tab. It causes
     * the first tab is always marked as active. The Tab Menu has a built-in
     * feature to highlight items based on the current route. It seems that it
     * matches by a prefix instead of an exact value (the "/foo/bar" URL
     * matches the menu item with the "/foo" URL).
     */
    tabs: MenuItem[] = [{ label: "Subnets", routerLink: "/dhcp/subnets/all" }]

    /**
     * Selected tab menu index.
     *
     * The first tab has an index of 0.
     */
    activeTabIndex = 0

    /**
     * Holds the information about specific subnets presented in the tabs.
     *
     * The entry corresponding to subnets list is not related to any specific
     * subnet, its ID is represented by the 0 value.
     */
    openedSubnets: Subnet[] = [{ id: 0 }]

    grafanaUrl: string

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private dhcpApi: DHCPService,
        private settingSvc: SettingService,
        private messageService: MessageService
    ) {}

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    ngOnInit() {
        // ToDo: Silent error catching
        this.subscriptions.add(
            this.settingSvc.getSettings().subscribe(
                (data) => {
                    this.grafanaUrl = data['grafana_url']
                },
                (error) => {
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot fetch server settings',
                        detail: getErrorMessage(error),
                    })
                }
            )
        )

        // handle initial query params
        const ssParams = this.route.snapshot.queryParamMap
        this.updateFilterText(ssParams)
        this.updateOurQueryParams(ssParams)

        // subscribe to subsequent changes to query params
        this.subscriptions.add(
            this.route.queryParamMap.subscribe(
                (params) => {
                    this.updateFilterText(params)
                    this.updateOurQueryParams(params)
                    let event = { first: 0, rows: 10 }
                    if (this.subnetsTable) {
                        event = this.subnetsTable.createLazyLoadMetadata()
                    }
                    this.loadSubnets(event)
                },
                (error) => {
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot process URL query parameters',
                        detail: getErrorMessage(error),
                    })
                }
            )
        )

        // Apply to the changes of the host id, e.g. from /dhcp/subnets/all to
        // /dhcp/hosts/1. Those changes are triggered by switching between the
        // tabs.
        this.subscriptions.add(
            this.route.paramMap.subscribe(
                (params) => {
                    // Get subnet id.
                    const id = params.get('id')
                    let numericId = parseInt(id, 10)
                    if (Number.isNaN(numericId)) {
                        numericId = 0
                    }
                    this.openTabBySubnetId(numericId)
                },
                (error) => {
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot process URL segment parameters',
                        detail: getErrorMessage(error),
                    })
                }
            )
        )
    }

    // ToDo: Silent error catching
    updateOurQueryParams(params) {
        if (['4', '6'].includes(params.get('dhcpVersion'))) {
            this.queryParams.dhcpVersion = params.get('dhcpVersion')
        }
        this.queryParams.text = params.get('text')
        this.queryParams.appId = params.get('appId')
    }

    // Set the value of the filter text using the URL query parameters.
    updateFilterText(params) {
        let text = ''
        if (params.get('text')) {
            text += ' ' + params.get('text')
        }
        if (params.get('appId')) {
            text += ' appId:' + params.get('appId')
        }
        this.filterText = text.trim()
    }

    /**
     * Loads subnets from the database into the component.
     *
     * @param event Event object containing index of the first row, maximum number
     *              of rows to be returned, dhcp version and text for subnets filtering.
     */
    loadSubnets(event) {
        const params = this.queryParams

        this.dhcpApi
            .getSubnets(event.first, event.rows, params.appId, params.dhcpVersion, params.text)
            // Custom parsing for statistics
            .pipe(
                map((subnets) => {
                    if (subnets.items) {
                        parseSubnetsStatisticValues(subnets.items)
                    }
                    return subnets
                })
            )
            .toPromise()
            .then((data) => {
                this.subnets = data.items
                this.totalSubnets = data.total
            })
            .catch((error) => {
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot load subnets',
                    detail: getErrorMessage(error),
                })
            })
    }

    /**
     * Filters list of subnets by DHCP versions. Filtering is realized server-side.
     */
    filterByDhcpVersion() {
        this.router.navigate(['/dhcp/subnets'], {
            queryParams: { dhcpVersion: this.queryParams.dhcpVersion },
            queryParamsHandling: 'merge',
        })
    }

    /**
     * Filters list of subnets by text. The text may contain key=val
     * pairs allowing filtering by various keys. Filtering is realized
     * server-side.
     */
    keyupFilterText(event) {
        if (this.filterText.length >= 2 || event.key === 'Enter') {
            const queryParams = extractKeyValsAndPrepareQueryParams(this.filterText, ['appId'], null)
            this.router.navigate(['/dhcp/subnets'], {
                queryParams,
                queryParamsHandling: 'merge',
            })
        }
    }

    /**
     * Prepare count for presenting in a column that it is easy to grasp by humans.
     */
    humanCount(count) {
        if (count < 1000001) {
            return count.toLocaleString('en-US')
        }

        return humanCount(count)
    }

    /**
     * Prepare count for presenting in tooltip by adding ',' separator to big numbers, eg. 1,243,342.
     */
    tooltipCount(count) {
        if (count === '?') {
            return 'No data collected yet'
        }
        return count.toLocaleString('en-US')
    }

    /**
     * Builds a tooltip explaining what the link is for.
     * @param subnet an identifier of the subnet
     * @param machine an identifier of the machine the subnet is configured on
     */
    getGrafanaTooltip(subnet: number, machine: string) {
        return getGrafanaSubnetTooltip(subnet, machine)
    }

    /**
     * Get total number of addresses in a subnet.
     */
    getTotalAddresses(subnet: Subnet) {
        if (subnet.stats) {
            return getTotalAddresses(subnet)
        } else {
            return '?'
        }
    }

    /**
     * Get assigned number of addresses in a subnet.
     */
    getAssignedAddresses(subnet: Subnet) {
        if (subnet.stats) {
            return getAssignedAddresses(subnet)
        } else {
            return '?'
        }
    }

    /**
     * Build URL to Grafana dashboard
     */
    getGrafanaUrl(name, subnet, instance) {
        return getGrafanaUrl(this.grafanaUrl, name, subnet, instance)
    }

    /**
     * Open a subnet tab. If the tab already exists, switch to it without
     * re-fetching the data. Otherwise, fetch the subnet from API and create a
     * new tab.
     * @param subnetId Subnet ID or a NaN for subnet list.
     */
    openTabBySubnetId(subnetId: number) {
        const tabIndex = this.openedSubnets.map(t => t.id).indexOf(subnetId)

        if (tabIndex < 0) {
            this._createTab(subnetId).then(() => {
                this._switchToTab(this.openedSubnets.length - 1)
            })
        } else {
            this._switchToTab(tabIndex)
        }
    }

    /**
     * Close a menu tab by index.
     * @param index Tab index.
     * @param event Event related to closing.
     */
    closeTabByIndex(index: number, event?: Event, ) {
        if (index == 0) {
            return
        }

        this.openedSubnets.splice(index, 1)
        this.tabs.splice(index, 1)

        if (index <= this.activeTabIndex) {
            this._switchToTab(this.activeTabIndex - 1)
        }

        if (event) {
            event.preventDefault()
        }
    }

    /**
     * Create a new tab for a given subnet ID. It's responsible for fetching
     * the subnet data.
     * @param subnetId Subnet Id.
     */
    private _createTab(subnetId: number): Promise<void> {
        return concat(
            // Existing entry or undefined.
            of(this.subnets.filter(s => s.id == subnetId)[0])
                // Drop an undefined value if the entry was not found.
                .pipe(filter(s => !!s)),
            // Fetch data from API.
            this.dhcpApi.getSubnet(subnetId)
        )
        // Take 1 item (return existing entry if exist, otherwise fetch the API).
        .pipe(take(1))
        // Execute and use.
        .toPromise()
        .then(data => {
            this._appendTab(data)
        })
        .catch(error => {
            const msg = getErrorMessage(error)
            this.messageService.add({
                severity: 'error',
                summary: 'Cannot get subnet',
                detail: `Error getting subnet with ID ${subnetId}: ${msg}`,
                life: 10000,
            })
        })
    }

    /**
     * Append the tab metadata to all appropriate places.
     * @param subnet Subnet data.
     */
    private _appendTab(subnet: Subnet) {
        this.openedSubnets.push(subnet)
        this.tabs.push({
            label: subnet.subnet,
            routerLink: `/dhcp/subnets/${subnet.id}`
        })
    }

    /**
     * Switch to tab by tab index.
     * @param index Tab index.
     */
    private _switchToTab(index: number) {
        if (this.activeTabIndex == index) {
            return
        }
        this.activeTabIndex = index
    }


}
