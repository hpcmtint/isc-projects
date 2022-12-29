import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { SubnetsPageComponent } from './subnets-page.component'
import { FormsModule } from '@angular/forms'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { TooltipModule } from 'primeng/tooltip'
import { RouterModule, ActivatedRoute, Router, convertToParamMap } from '@angular/router'
import { DHCPService, SettingsService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { of } from 'rxjs'
import { MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { DelegatedPrefixBarComponent } from '../delegated-prefix-bar/delegated-prefix-bar.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { NumberPipe } from '../pipes/number.pipe'

class MockParamMap {
    get(name: string): string | null {
        return null
    }
}

describe('SubnetsPageComponent', () => {
    let component: SubnetsPageComponent
    let fixture: ComponentFixture<SubnetsPageComponent>
    let dhcpService: DHCPService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                DHCPService,
                UsersService,
                MessageService,
                SettingsService,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        snapshot: { queryParamMap: new MockParamMap() },
                        queryParamMap: of(new MockParamMap()),
                    },
                },
                {
                    provide: Router,
                    useValue: {},
                },
            ],
            imports: [
                FormsModule,
                DropdownModule,
                TableModule,
                TooltipModule,
                RouterModule,
                HttpClientTestingModule,
                BreadcrumbModule,
                OverlayPanelModule,
                NoopAnimationsModule,
            ],
            declarations: [
                SubnetsPageComponent,
                SubnetBarComponent,
                BreadcrumbsComponent,
                HelpTipComponent,
                DelegatedPrefixBarComponent,
                HumanCountComponent,
                NumberPipe,
            ],
        })
        dhcpService = TestBed.inject(DHCPService)
    }))

    beforeEach(() => {
        const fakeResponses: any = [
            {
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
                                machineHostname: 'lv-pc',
                            },
                            {
                                appId: 28,
                                appName: 'kea2@localhost',
                                id: 2,
                                machineAddress: 'host',
                                machineHostname: 'lv-pc2',
                            },
                        ],
                        stats: {
                            'assigned-addresses':
                                '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                            'total-addresses':
                                '1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890',
                            'declined-addresses': '-2',
                        },
                        statsCollectedAt: '2022-01-19T12:10:22.513Z',
                        pools: ['1.0.0.4-1.0.255.254'],
                        subnet: '1.0.0.0/16',
                    },
                ],
                total: 10496,
            },
            {
                items: [
                    {
                        clientClass: 'class-00-00',
                        id: 5,
                        localSubnets: [
                            {
                                appId: 28,
                                appName: 'kea2@localhost',
                                id: 2,
                                machineAddress: 'host',
                                machineHostname: 'lv-pc2',
                            },
                        ],
                        statsCollectedAt: '1970-01-01T12:00:00.0Z',
                        pools: ['1.0.0.4-1.0.255.254'],
                        subnet: '1.0.0.0/16',
                    },
                ],
                total: 10496,
            },
        ]
        spyOn(dhcpService, 'getSubnets').and.returnValues(
            // The subnets are fetched twice before the unit test starts.
            of(fakeResponses[0]),
            of(fakeResponses[0]),
            of(fakeResponses[1])
        )

        fixture = TestBed.createComponent(SubnetsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should convert statistics to big integers', async () => {
        // Act
        await fixture.whenStable()

        // Assert
        const stats: { [key: string]: BigInt } = component.subnets[0].stats as any
        expect(stats['assigned-addresses']).toBe(
            BigInt('12345678901234567890123456789012345678901234567890123456789012345678901234567890')
        )
        expect(stats['total-addresses']).toBe(
            BigInt(
                '1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890'
            )
        )
        expect(stats['declined-addresses']).toBe(BigInt('-2'))
    })

    it('should not fail on empty statistics', async () => {
        // Act
        component.loadSubnets({})
        await fixture.whenStable()

        // Assert
        expect(component.subnets[0].stats).toBeUndefined()
        // No throw
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('DHCP')
        expect(breadcrumbsComponent.items[1].label).toEqual('Subnets')
    })
})
