import { HttpErrorResponse } from '@angular/common/http'
import { Component, OnDestroy, OnInit } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'
import { MessageService } from 'primeng/api'
import { Subject, Subscription } from 'rxjs'
import { switchMap } from 'rxjs/operators'
import { ServicesService } from '../backend/api/api'
import { ServerDataService } from '../server-data.service'

/**
 * A component providing a dedicated page displaying Kea daemon configuration.
 *
 * It fetches configuration data and displays it in a JSON viewer.
 * The viewer allows for collapsing and expanding selected or all nodes.
 */
@Component({
    selector: 'app-kea-daemon-configuration-page',
    templateUrl: './kea-daemon-configuration-page.component.html',
    styleUrls: ['./kea-daemon-configuration-page.component.sass'],
})
export class KeaDaemonConfigurationPageComponent implements OnInit, OnDestroy {
    breadcrumbs = [
        { label: 'Services' },
        { label: 'Kea Apps', routerLink: '/apps/kea/all' },
        { label: 'App' },
        { label: 'Daemons' },
        { label: 'Daemon' },
        { label: 'Configuration' },
    ]

    // Variables to store values for getters. See specific getter for documentation.
    private _autoExpandNodeCount = 0
    private _configuration = null
    private _daemonId: number = null
    private _downloadFilename = 'data.json'
    private _failedFetch = false

    private changeDaemonId = new Subject<number>()
    private changeAppId = new Subject<number>()
    private subscription = new Subscription()

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private servicesApi: ServicesService,
        private serverData: ServerDataService,
        private msgService: MessageService
    ) {}

    /**
     * Unsubscribe all subscriptions.
     */
    ngOnDestroy(): void {
        this.subscription.unsubscribe()
    }

    /**
     * Component lifecycle hook invoked upon the component initialization.
     *
     * It subscribes for necessary data, i.e. friendly names and daemon configuration JSON.
     *
     * The app friendly name is fetched for the specified app ID query parameter. If the app
     * with the specified ID does not exist or the ID is invalid, a placeholder for app name
     * is displayed.
     *
     * The daemon ID must be valid and must point to an existing daemon. The function uses
     * it to fetch daemon's friendly name and fetch its configuration. If the daemon ID is
     * invalid, the user is redirected to the apps list.
     * application list.
     */
    ngOnInit(): void {
        // Friendly names of daemons
        const DMAP = {
            dhcp4: 'DHCPv4',
            dhcp6: 'DHCPv6',
            d2: 'DDNS',
            ca: 'CA',
            netconf: 'NETCONF',
        }

        // Update friendly names
        this.subscription.add(
            this.changeAppId.pipe(switchMap((appId) => this.servicesApi.getApp(appId))).subscribe((app) => {
                // Find specific daemon
                const daemons = app.details.daemons.filter((d) => d.id === this._daemonId)
                const daemonName = daemons[0]?.name
                const friendlyName = DMAP[daemonName] ?? daemonName ?? this._daemonId + '' ?? 'Unknown'

                // User-friendly download filename
                this._downloadFilename = `${app.name}_${friendlyName}.json`

                // Breadcrumbs
                this.breadcrumbs[2]['label'] = app.name
                this.breadcrumbs[2]['routerLink'] = `/apps/kea/${app.id}`
                this.breadcrumbs[4]['label'] = friendlyName
                this.breadcrumbs[4]['routerLink'] = `/apps/kea/${app.id}?daemon=${daemonName}`
            })
        )

        // Update Kea daemon configuration
        this.subscription.add(
            this.changeDaemonId
                .pipe(switchMap((daemonId) => this.serverData.getDaemonConfiguration(daemonId)))
                .subscribe((res) => {
                    if (res instanceof HttpErrorResponse) {
                        this.msgService.add({
                            severity: 'error',
                            summary: 'Fetching daemon configuration failed',
                            detail: res.error?.message ?? res.message,
                            life: 10000,
                        })
                        this._failedFetch = true
                        this._configuration = null
                    } else {
                        this._failedFetch = false
                        this._configuration = res
                    }
                })
        )

        // Resolve URI parameters
        this.subscription.add(
            this.route.paramMap.subscribe((params) => {
                const appIdStr = params.get('appId')
                const daemonIdStr = params.get('daemonId')

                const appId = parseInt(appIdStr, 10)
                const daemonId = parseInt(daemonIdStr, 10)

                // Daemon ID is required
                if (!Number.isFinite(daemonId)) {
                    this.router.navigate(['/apps/kea/all'])
                }

                this._daemonId = daemonId

                // Ignore App ID if it is incorrect. It is not necessary to display
                // the configuration tree. It is merely used in breadcrumbs.
                if (Number.isFinite(appId)) {
                    this.changeAppId.next(appId)
                }

                this.changeDaemonId.next(daemonId)
            })
        )
    }

    /**
     * Handle click event of toggle collapse/expand nodes button.
     *
     * JSON viewer uses HTML built-in details-summary tags. It means that we cannot
     * directly control collapse/expand state. We can only toggle "open" HTML property
     * to indicate that tag should be initially expanded or not.
     *
     * This function implements an auto expand feature and setting the count of the
     * auto expanded nodes to 0 (collapse) or max integer value (expand).
     */
    onClickToggleNodes() {
        if (this._autoExpandNodeCount === Number.MAX_SAFE_INTEGER) {
            this._autoExpandNodeCount = 0
        } else {
            this._autoExpandNodeCount = Number.MAX_SAFE_INTEGER
        }
    }

    /** Handle click event on refresh button. */
    onClickRefresh() {
        // Reset configuration instance to display loading indicator.
        this._configuration = null
        this._failedFetch = false
        this.serverData.forceReloadDaemonConfiguration(this.daemonId)
    }

    /** Specifies current toggle/expand button state. */
    get currentAction() {
        return this._autoExpandNodeCount === 0 ? 'expand' : 'collapse'
    }

    /** Returns 0 for collapse nodes or maximal integer for expand. */
    get autoExpandNodeCount() {
        return this._autoExpandNodeCount
    }

    /** Kea daemon configuration to display */
    get configuration() {
        return this._configuration
    }

    /** Filename of downloaded Kea daemon configuration */
    get downloadFilename() {
        return this._downloadFilename
    }

    /** Kea daemon ID of current configuration. If not set then it is null. */
    get daemonId() {
        return this._daemonId
    }

    /** Indicates that fetch configuration failed. */
    get failedFetch() {
        return this._failedFetch
    }
}
