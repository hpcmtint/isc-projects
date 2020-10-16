import { Component, OnInit } from '@angular/core'
import { Router } from '@angular/router'
import { Observable } from 'rxjs'

import { MenuItem } from 'primeng/api'

import { GeneralService } from './backend/api/api'
import { AuthService } from './auth.service'
import { LoadingService } from './loading.service'
import { SettingService } from './setting.service'
import { ServerDataService } from './server-data.service'

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.sass'],
})
export class AppComponent implements OnInit {
    storkVersion = 'unknown'
    storkBuildDate = 'unknown'
    currentUser = null
    loadingInProgress = new Observable()
    userMenuItems: MenuItem[]

    menuItems: MenuItem[]

    breadcrumbItems: MenuItem[]

    constructor(
        private router: Router,
        private serverData: ServerDataService,
        protected generalApi: GeneralService,
        private auth: AuthService,
        private loadingService: LoadingService,
        private settingSvc: SettingService
    ) {
        this.initMenus()

        this.breadcrumbItems = [{ label: 'Categories' }]

        this.loadingInProgress = this.loadingService.getState()
    }

    initMenus() {
        this.userMenuItems = [
            {
                label: 'Profile',
                id: 'profile',
                icon: 'fa fa-cog',
                routerLink: '/profile',
            },
        ]

        this.menuItems = [
            {
                label: 'DHCP',
                id: 'dhcp',
                visible: false,
                items: [
                    {
                        label: 'Dashboard',
                        id: 'dashboard',
                        icon: 'fa fa-tachometer-alt',
                        routerLink: '/dashboard',
                    },
                    {
                        label: 'Host Reservations',
                        id: 'host-reservations',
                        icon: 'fa fa-laptop',
                        routerLink: '/dhcp/hosts',
                    },
                    {
                        label: 'Subnets',
                        id: 'subnets',
                        icon: 'fa fa-project-diagram',
                        routerLink: '/dhcp/subnets',
                    },
                    {
                        label: 'Shared Networks',
                        id: 'shared-networks',
                        icon: 'fa fa-network-wired',
                        routerLink: '/dhcp/shared-networks',
                    },
                ],
            },
            {
                label: 'Services',
                id: 'services',
                items: [
                    {
                        label: 'Kea Apps',
                        id: 'kea-apps',
                        visible: false,
                        icon: 'fa fa-server',
                        routerLink: '/apps/kea/all',
                    },
                    {
                        label: 'BIND 9 Apps',
                        id: 'bind9-apps',
                        visible: false,
                        icon: 'fa fa-server',
                        routerLink: '/apps/bind9/all',
                    },
                    {
                        label: 'Machines',
                        id: 'machines',
                        icon: 'fa fa-server',
                        routerLink: '/machines/all',
                    },
                    {
                        label: 'Grafana',
                        id: 'grafana',
                        icon: 'pi pi-chart-line',
                        url: '',
                        visible: false,
                    },
                ],
            },
            {
                label: 'Monitoring',
                id: 'monitoring',
                items: [
                    {
                        label: 'Events',
                        id: 'events',
                        icon: 'fa fa-calendar-times',
                        routerLink: '/events',
                    },
                ],
            },
            {
                label: 'Configuration',
                id: 'configuration',
                items: [
                    {
                        label: 'Users',
                        id: 'users',
                        visible: false,
                        icon: 'fa fa-user',
                        routerLink: '/users',
                    },
                    {
                        label: 'Settings',
                        id: 'settings',
                        icon: 'fa fa-cog',
                        routerLink: '/settings',
                    },
                ],
            },
            {
                label: 'Help',
                id: 'help',
                items: [
                    {
                        label: 'Stork Manual',
                        id: 'stork-manual',
                        icon: 'fa fa-book',
                        url: '/assets/arm/index.html',
                        target: 'blank',
                    },
                    {
                        label: 'Stork API Docs (SwaggerUI)',
                        id: 'stork-API-docs-swagger',
                        icon: 'fa fa-code',
                        routerLink: '/swagger-ui',
                    },
                    {
                        label: 'Stork API Docs (Redoc)',
                        id: 'stork-API-docs-redoc',
                        icon: 'fa fa-code',
                        url: '/api/docs',
                        target: 'blank',
                    },
                    {
                        label: 'BIND 9 Manual',
                        id: 'bind9-manual',
                        icon: 'fa fa-book',
                        url: 'https://downloads.isc.org/isc/bind9/cur/9.16/doc/arm/Bv9ARM.html',
                        target: 'blank',
                    },
                    {
                        label: 'Kea Manual',
                        id: 'kea-manual',
                        icon: 'fa fa-book',
                        url: 'https://kea.readthedocs.io/',
                        target: 'blank',
                    },
                ],
            },
        ]
    }

    /**
     * Get menu item or subitem from Stork menu based on provided name.
     *
     * @param name A menu item name that must exist in Stork menu tree
     *             that is defined in this.menuItems.
     * @returns A reference to found menu item or null if not found.
     */
    getMenuItem(name) {
        for (const menuItem of this.menuItems) {
            if (menuItem['label'] === name) {
                return menuItem
            }
            for (const subMenu of menuItem.items) {
                if (subMenu['label'] === name) {
                    return subMenu
                }
            }
        }
        console.error('menu item not found', name)
        return null
    }

    ngOnInit() {
        this.generalApi.getVersion().subscribe((data) => {
            this.storkVersion = data.version
            this.storkBuildDate = data.date
        })

        this.auth.currentUser.subscribe((x) => {
            this.currentUser = x
            const menuItem = this.getMenuItem('Users')
            if (this.auth.superAdmin()) {
                // super admin can see Configuration/Users menu
                menuItem['visible'] = true
            } else {
                menuItem['visible'] = false
            }

            // Only get the stats and settings when the user is logged in.
            if (this.auth.currentUserValue) {
                this.serverData.getAppsStats().subscribe((data) => {
                    // if there are Kea apps then show Kea related menu items
                    // otherwise hide them
                    const dhcpMenuItem = this.getMenuItem('DHCP')
                    const keaAppsMenuItem = this.getMenuItem('Kea Apps')
                    if (data.keaAppsTotal && data.keaAppsTotal > 0) {
                        dhcpMenuItem.visible = true
                        keaAppsMenuItem['visible'] = true
                    } else {
                        dhcpMenuItem.visible = false
                        keaAppsMenuItem['visible'] = false
                    }
                    // if there are BIND 9 apps then show BIND 9 related menu items
                    // otherwise hide them
                    const bind9AppsMenuItem = this.getMenuItem('BIND 9 Apps')
                    if (data.bind9AppsTotal && data.bind9AppsTotal > 0) {
                        bind9AppsMenuItem['visible'] = true
                    } else {
                        bind9AppsMenuItem['visible'] = false
                    }
                })

                // If Grafana url is not empty, we need to make
                // Services.Grafana menu choice visible and set it's url.
                // Otherwise we need to make sure it's not visible.
                this.settingSvc.getSettings().subscribe((data) => {
                    const grafanaUrl = data['grafana_url']

                    const grafanaMenuItem = this.getMenuItem('Grafana')

                    if (grafanaUrl && grafanaUrl !== '') {
                        grafanaMenuItem['visible'] = true
                        grafanaMenuItem['url'] = grafanaUrl
                    } else {
                        grafanaMenuItem['visible'] = false
                    }
                })
            }
        })
    }

    signOut() {
        this.router.navigate(['/logout'])
    }
}
