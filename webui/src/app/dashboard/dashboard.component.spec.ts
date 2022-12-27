import { ComponentFixture, fakeAsync, TestBed, waitForAsync } from '@angular/core/testing'

import { DashboardComponent } from './dashboard.component'
import { PanelModule } from 'primeng/panel'
import { ButtonModule } from 'primeng/button'
import { ServicesService, DHCPService, SettingsService, UsersService, DhcpOverview, AppsStats } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { LocationStrategy, PathLocationStrategy } from '@angular/common'
import { of } from 'rxjs'
import { By } from '@angular/platform-browser'
import { ServerDataService } from '../server-data.service'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { RouterTestingModule } from '@angular/router/testing'
import { PaginatorModule } from 'primeng/paginator'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TooltipModule } from 'primeng/tooltip'
import { TableModule } from 'primeng/table'
import { EntityLinkComponent } from '../entity-link/entity-link.component'

describe('DashboardComponent', () => {
    let component: DashboardComponent
    let fixture: ComponentFixture<DashboardComponent>
    let dhcpService: DHCPService
    let dataService: ServerDataService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [
                NoopAnimationsModule,
                PanelModule,
                OverlayPanelModule,
                PaginatorModule,
                TooltipModule,
                ButtonModule,
                RouterTestingModule,
                HttpClientTestingModule,
                TableModule,
            ],
            declarations: [
                DashboardComponent,
                EventsPanelComponent,
                HelpTipComponent,
                SubnetBarComponent,
                EntityLinkComponent,
            ],
            providers: [
                ServicesService,
                LocationStrategy,
                DHCPService,
                MessageService,
                UsersService,
                SettingsService,
                { provide: LocationStrategy, useClass: PathLocationStrategy },
            ],
        })

        dhcpService = TestBed.inject(DHCPService)
        dataService = TestBed.inject(ServerDataService)
    }))

    beforeEach(() => {
        const fakeOverview: DhcpOverview = {
            dhcp4Stats: {
                assignedAddresses: '6553',
                declinedAddresses: '100',
                totalAddresses: '65530',
            },
            dhcp6Stats: {
                assignedNAs: '20',
                assignedPDs: '1',
                declinedNAs: '10',
                totalNAs: '100',
                totalPDs: '2',
            },
            dhcpDaemons: [
                {
                    active: true,
                    agentCommErrors: 6,
                    appId: 27,
                    appName: 'kea@localhost',
                    appVersion: '2.0.0',
                    haFailureAt: '0001-01-01T00:00:00.000Z',
                    machine: 'pc',
                    machineId: 15,
                    monitored: true,
                    name: 'dhcp4',
                    uptime: 3652,
                },
            ],
            sharedNetworks4: {
                items: [
                    {
                        id: 5,
                        addrUtilization: 40,
                        name: 'frog',
                        subnets: [],
                    },
                ],
            },
            sharedNetworks6: {
                items: [
                    {
                        id: 6,
                        addrUtilization: 50,
                        name: 'mouse',
                        subnets: [],
                    },
                ],
            },
            subnets4: {
                items: [
                    {
                        clientClass: 'class-00-00',
                        id: 5,
                        localSubnets: [
                            {
                                appId: 27,
                                appName: 'kea@localhost',
                                id: 1,
                                machineAddress: 'localhost',
                                machineHostname: 'pc',
                            },
                        ],
                        pools: ['1.0.0.4-1.0.255.254'],
                        subnet: '1.0.0.0/16',
                        stats: {
                            'total-addresses': '65530',
                            'assigned-addresses': '6553',
                            'declined-addresses': '100',
                        },
                        statsCollectedAt: '2022-01-19T12:10:22.513Z',
                        addrUtilization: 10,
                    },
                ],
                total: 10496,
            },
            subnets6: {
                items: [
                    {
                        clientClass: 'class-00-00',
                        id: 6,
                        localSubnets: [
                            {
                                appId: 27,
                                appName: 'kea@localhost',
                                id: 2,
                                machineAddress: 'localhost',
                                machineHostname: 'pc',
                            },
                        ],
                        stats: {
                            'total-nas': '100',
                            'assigned-nas': '20',
                            'declined-nas': '10',
                            'total-pds': '2',
                            'assigned-pds': '1',
                        },
                        statsCollectedAt: '2022-01-19T12:10:22.513Z',
                        pools: ['10.3::1-10.3::100'],
                        subnet: '10:3::/64',
                        addrUtilization: 20,
                    },
                ],
            },
        }

        spyOn(dhcpService, 'getDhcpOverview').and.returnValues(of({} as any), of(fakeOverview as any))
        spyOn(dataService, 'getAppsStats').and.returnValue(
            of({
                keaAppsTotal: 1,
                bind9AppsNotOk: 0,
                bind9AppsTotal: 0,
                keaAppsNotOk: 0,
            } as AppsStats)
        )

        fixture = TestBed.createComponent(DashboardComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should indicate that HA is not enabled', () => {
        // This test doesn't check that the state is rendered correctly
        // as HTML, because the table listing daemons is dynamic and
        // finding the right table cell is going to be involved. Instead
        // we test it indirectly by making sure that the functions used
        // to render the content return expected values.
        const daemon = { haState: 'load-balancing', haFailureAt: '2014-06-01T12:00:00Z' }
        expect(component.showHAState(daemon)).toBe('not configured')
        expect(component.showHAFailureTime(daemon)).toBe('')
        expect(component.haStateIcon(daemon)).toBe('ban')

        const daemon2 = { haEnabled: false, haState: 'load-balancing', haFailureAt: '2014-06-01T12:00:00Z' }
        expect(component.showHAState(daemon2)).toBe('not configured')
        expect(component.showHAFailureTime(daemon2)).toBe('')
        expect(component.haStateIcon(daemon2)).toBe('ban')

        const daemon3 = { haEnabled: true, haState: '', haFailureAt: '0001-01-01' }
        expect(component.showHAState(daemon3)).toBe('fetching...')
        expect(component.showHAFailureTime(daemon3)).toBe('')
        expect(component.haStateIcon(daemon3)).toBe('spin pi-spinner')

        const daemon4 = { haEnabled: true, haFailureAt: '0001-01-01' }
        expect(component.showHAState(daemon4)).toBe('fetching...')
        expect(component.showHAFailureTime(daemon4)).toBe('')
        expect(component.haStateIcon(daemon4)).toBe('spin pi-spinner')
    })

    it('should parse integer statistics', async () => {
        await component.refreshDhcpOverview()
        expect(component.overview.subnets4.items[0].stats['total-addresses']).toBe(BigInt(65530))
        expect(component.overview.subnets6.items[0].stats['assigned-nas']).toBe(BigInt(20))
    })

    it('should display utilizations', async () => {
        await component.refreshDhcpOverview()
        fixture.detectChanges()
        await fixture.whenRenderingDone()

        // DHCPv4
        let rows = fixture.debugElement.queryAll(
            By.css('#dashboard-dhcp4 .dashboard-dhcp__subnets .dashboard-section__data .utilization-row')
        )
        expect(rows.length).toBe(1)
        let row = rows[0]
        let cell = row.query(By.css('.utilization-row__value'))
        expect(cell).not.toBeNull()
        let utilization = (cell.nativeElement as HTMLElement).textContent
        expect(utilization.trim()).toBe('10% used')

        // DHCPv6
        rows = fixture.debugElement.queryAll(
            By.css('#dashboard-dhcp6 .dashboard-dhcp__shared-networks .dashboard-section__data .utilization-row')
        )
        expect(rows.length).toBe(1)
        row = rows[0]
        cell = row.query(By.css('.utilization-row__value'))
        expect(cell).not.toBeNull()
        utilization = (cell.nativeElement as HTMLElement).textContent
        expect(utilization.trim()).toBe('50% used')
    })

    it('should display global statistics', async () => {
        await component.refreshDhcpOverview()
        fixture.detectChanges()
        await fixture.whenRenderingDone()

        const rows = fixture.debugElement.queryAll(
            By.css('.dashboard-dhcp__globals .dashboard-section__data .statistics-row')
        )
        const expected = [
            ['Addresses', '6k / 65k (10% used)'],
            ['Declined', '100'],
            ['Addresses', '20 / 100 (20% used)'],
            ['Prefixes', '1 / 2 (50% used)'],
            ['Declined', '10'],
        ]

        expect(rows.length).toBe(expected.length)

        for (let i = 0; i < expected.length; i++) {
            const [expectedLabel, expectedValue] = expected[i]
            const rowElement = rows[i].nativeElement as HTMLElement
            const labelElement = rowElement.querySelector('.statistics-row__label')
            const valueElement = rowElement.querySelector('.statistics-row__value')
            const labelText = labelElement.textContent.trim()
            const valueText = valueElement.textContent.trim()
            expect(labelText).toBe(expectedLabel)
            expect(valueText).toBe(expectedValue)
        }
    })
})
