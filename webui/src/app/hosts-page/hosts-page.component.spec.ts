import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'

import { HostsPageComponent } from './hosts-page.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { FormBuilder, FormsModule, ReactiveFormsModule } from '@angular/forms'
import { MessageService } from 'primeng/api'
import { TableModule } from 'primeng/table'
import { DHCPService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ActivatedRoute, Router, convertToParamMap } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { By } from '@angular/platform-browser'
import { of, throwError, BehaviorSubject } from 'rxjs'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { TabMenuModule } from 'primeng/tabmenu'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { HostTabComponent } from '../host-tab/host-tab.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { TooltipModule } from 'primeng/tooltip'
import { FieldsetModule } from 'primeng/fieldset'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { IdentifierComponent } from '../identifier/identifier.component'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { DropdownModule } from 'primeng/dropdown'
import { MultiSelectModule } from 'primeng/multiselect'
import { HostFormComponent } from '../host-form/host-form.component'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'

class MockParamMap {
    get(name: string): string | null {
        return null
    }
}

describe('HostsPageComponent', () => {
    let component: HostsPageComponent
    let fixture: ComponentFixture<HostsPageComponent>
    let router: Router
    let route: ActivatedRoute
    let dhcpApi: DHCPService
    let messageService: MessageService
    let paramMap: any
    let paramMapSubject: BehaviorSubject<any>
    let paramMapSpy: any

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                providers: [DHCPService, FormBuilder, MessageService],
                imports: [
                    FormsModule,
                    TableModule,
                    HttpClientTestingModule,
                    RouterTestingModule.withRoutes([
                        {
                            path: 'dhcp/hosts',
                            pathMatch: 'full',
                            redirectTo: 'dhcp/hosts/all',
                        },
                        {
                            path: 'dhcp/hosts/:id',
                            component: HostsPageComponent,
                        },
                    ]),
                    TabMenuModule,
                    BreadcrumbModule,
                    OverlayPanelModule,
                    NoopAnimationsModule,
                    TooltipModule,
                    FormsModule,
                    FieldsetModule,
                    ProgressSpinnerModule,
                    ToggleButtonModule,
                    ButtonModule,
                    CheckboxModule,
                    DropdownModule,
                    FieldsetModule,
                    MultiSelectModule,
                    ReactiveFormsModule,
                ],
                declarations: [
                    EntityLinkComponent,
                    HostsPageComponent,
                    BreadcrumbsComponent,
                    HelpTipComponent,
                    HostTabComponent,
                    IdentifierComponent,
                    HostFormComponent,
                    DhcpOptionFormComponent,
                    DhcpOptionSetFormComponent,
                ],
            }).compileComponents()
        })
    )

    beforeEach(() => {
        fixture = TestBed.createComponent(HostsPageComponent)
        component = fixture.componentInstance
        router = fixture.debugElement.injector.get(Router)
        route = fixture.debugElement.injector.get(ActivatedRoute)
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        messageService = fixture.debugElement.injector.get(MessageService)
        paramMap = convertToParamMap({})
        paramMapSubject = new BehaviorSubject(paramMap)
        paramMapSpy = spyOnProperty(route, 'paramMap').and.returnValue(paramMapSubject)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
        expect(component.filterText.length).toBe(0)
    })

    it('host table should have valid app name and app link', () => {
        component.hosts = [{ id: 1, localHosts: [{ appId: 1, appName: 'frog', dataSource: 'config' }] }]
        fixture.detectChanges()
        // Table rows have ids created by appending host id to the host-row- string.
        const row = fixture.debugElement.query(By.css('#host-row-1'))
        // There should be 6 table cells in the row.
        expect(row.children.length).toBe(6)
        // The last one includes the app name.
        const appNameTd = row.children[5]
        // The cell includes a link to the app.
        expect(appNameTd.children.length).toBe(1)
        const appLink = appNameTd.children[0]
        expect(appLink.nativeElement.innerText).toBe('frog config')
        // Verify that the link to the app is correct.
        expect(appLink.properties.hasOwnProperty('pathname')).toBeTrue()
        expect(appLink.properties.pathname).toBe('/apps/kea/1')
    })

    it('should open and close host tabs', () => {
        // Create a list with two hosts.
        component.hosts = [
            {
                id: 1,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '01:02:03:04',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.1',
                    },
                ],
                localHosts: [
                    {
                        appId: 1,
                        appName: 'frog',
                        dataSource: 'config',
                    },
                ],
            },
            {
                id: 2,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '11:12:13:14',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.2',
                    },
                ],
                localHosts: [
                    {
                        appId: 2,
                        appName: 'mouse',
                        dataSource: 'config',
                    },
                ],
            },
        ]
        fixture.detectChanges()

        // Ensure that we don't fetch the host information from the server upon
        // opening a new tab. We should use the information available in the
        // hosts structure.
        spyOn(dhcpApi, 'getHost')

        // Open tab with host with id 1.
        paramMapSubject.next(convertToParamMap({ id: 1 }))
        fixture.detectChanges()
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(1)

        // Open the tab for creating a host.
        paramMapSubject.next(convertToParamMap({ id: 'new' }))
        fixture.detectChanges()
        expect(component.tabs.length).toBe(3)
        expect(component.activeTabIndex).toBe(2)

        // Open tab with host with id 2.
        paramMapSubject.next(convertToParamMap({ id: 2 }))
        fixture.detectChanges()
        expect(component.tabs.length).toBe(4)
        expect(component.activeTabIndex).toBe(3)

        // Navigate back to the hosts list in the first tab.
        paramMapSubject.next(convertToParamMap({}))
        fixture.detectChanges()
        expect(component.tabs.length).toBe(4)
        expect(component.activeTabIndex).toBe(0)

        // Navigate to the existing tab with host with id 1.
        paramMapSubject.next(convertToParamMap({ id: 1 }))
        fixture.detectChanges()
        expect(component.tabs.length).toBe(4)
        expect(component.activeTabIndex).toBe(1)

        // navigate to the existing tab for adding new host.
        paramMapSubject.next(convertToParamMap({ id: 'new' }))
        fixture.detectChanges()
        expect(component.tabs.length).toBe(4)
        expect(component.activeTabIndex).toBe(2)

        // Close the second tab.
        component.closeHostTab(null, 1)
        fixture.detectChanges()
        expect(component.tabs.length).toBe(3)
        expect(component.activeTabIndex).toBe(1)

        // Close the tab for adding new host.
        component.closeHostTab(null, 1)
        fixture.detectChanges()
        expect(component.tabs.length).toBe(2)
        expect(component.activeTabIndex).toBe(0)

        // Close the remaining tab.
        component.closeHostTab(null, 1)
        fixture.detectChanges()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)
    })

    it('should open a tab when hosts have not been loaded', () => {
        const host: any = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.1',
                },
            ],
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'config',
                },
            ],
        }
        // Do not initialize the hosts list. Instead, simulate returning the
        // host information from the server. The component should send the
        // request to the server to get the host.
        spyOn(dhcpApi, 'getHost').and.returnValue(of(host))
        paramMapSubject.next(convertToParamMap({ id: 1 }))
        fixture.detectChanges()
        // There should be two tabs opened. One with the list of hosts and one
        // with the host details.
        expect(component.tabs.length).toBe(2)
    })

    it('should not open a tab when getting host information erred', () => {
        // Simulate the getHost call to return an error.
        spyOn(dhcpApi, 'getHost').and.returnValue(throwError({ status: 404 }))
        spyOn(messageService, 'add')
        paramMapSubject.next(convertToParamMap({ id: 1 }))
        fixture.detectChanges()
        // There should still be one tab open with a list of hosts.
        expect(component.tabs.length).toBe(1)
        // Ensure that the error message was displayed.
        expect(messageService.add).toHaveBeenCalled()
    })

    it('should generate a label from host information', () => {
        const host0 = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.1',
                },
            ],
            prefixReservations: [
                {
                    address: '2001:db8::',
                },
            ],
            hostname: 'mouse.example.org',
        }

        expect(component.getHostLabel(host0)).toBe('192.0.2.1')

        const host1 = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
            prefixReservations: [
                {
                    address: '2001:db8::',
                },
            ],
            hostname: 'mouse.example.org',
        }

        expect(component.getHostLabel(host1)).toBe('2001:db8::')

        const host2 = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
            hostname: 'mouse.example.org',
        }
        expect(component.getHostLabel(host2)).toBe('mouse.example.org')

        const host3 = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
            ],
        }
        expect(component.getHostLabel(host3)).toBe('duid=01:02:03:04')

        const host4 = {
            id: 1,
        }
        expect(component.getHostLabel(host4)).toBe('[1]')
    })

    it('should display well formatted host identifiers', () => {
        // Create a list with three hosts. One host uses a duid convertible
        // to a textual format. Another host uses a hw-address which is
        // by default displayed in the hex format. Third host uses a
        // flex-id which is not convertible to a textual format.
        component.hosts = [
            {
                id: 1,
                hostIdentifiers: [
                    {
                        idType: 'duid',
                        idHexValue: '61:62:63:64',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.1',
                    },
                ],
                localHosts: [
                    {
                        appId: 1,
                        appName: 'frog',
                        dataSource: 'config',
                    },
                ],
            },
            {
                id: 2,
                hostIdentifiers: [
                    {
                        idType: 'hw-address',
                        idHexValue: '51:52:53:54:55:56',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.2',
                    },
                ],
                localHosts: [
                    {
                        appId: 2,
                        appName: 'mouse',
                        dataSource: 'config',
                    },
                ],
            },
            {
                id: 3,
                hostIdentifiers: [
                    {
                        idType: 'flex-id',
                        idHexValue: '10:20:30:40:50',
                    },
                ],
                addressReservations: [
                    {
                        address: '192.0.2.2',
                    },
                ],
                localHosts: [
                    {
                        appId: 3,
                        appName: 'lion',
                        dataSource: 'config',
                    },
                ],
            },
        ]
        fixture.detectChanges()

        // There should be 3 hosts listed.
        const identifierEl = fixture.debugElement.queryAll(By.css('app-identifier'))
        expect(identifierEl.length).toBe(3)

        // Each host identifier should be a link.
        const firstIdEl = identifierEl[0].query(By.css('a'))
        expect(firstIdEl).toBeTruthy()
        // The DUID is convertible to text.
        expect(firstIdEl.nativeElement.textContent).toContain('duid=(abcd)')
        expect(firstIdEl.attributes.href).toBe('/dhcp/hosts/1')

        const secondIdEl = identifierEl[1].query(By.css('a'))
        expect(secondIdEl).toBeTruthy()
        // The HW address is convertible but by default should be in hex format.
        expect(secondIdEl.nativeElement.textContent).toContain('hw-address=(51:52:53:54:55:56)')
        expect(secondIdEl.attributes.href).toBe('/dhcp/hosts/2')

        const thirdIdEl = identifierEl[2].query(By.css('a'))
        expect(thirdIdEl).toBeTruthy()
        // The flex-id is not convertible to text so should be in hex format.
        expect(thirdIdEl.nativeElement.textContent).toContain('flex-id=(10:20:30:40:50)')
        expect(thirdIdEl.attributes.href).toBe('/dhcp/hosts/3')
    })

    it('should close new host tab when form is submitted', fakeAsync(() => {
        const createHostBeginResp: any = {
            id: 123,
            subnets: [
                {
                    id: 1,
                    subnet: '192.0.2.0/24',
                    localSubnets: [
                        {
                            daemonId: 1,
                        },
                    ],
                },
            ],
            daemons: [
                {
                    id: 1,
                    name: 'dhcp4',
                    app: {
                        name: 'first',
                    },
                },
            ],
        }
        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(createHostBeginResp))
        spyOn(dhcpApi, 'createHostDelete').and.returnValue(of(okResp))

        paramMapSubject.next(convertToParamMap({ id: 'new' }))
        tick()
        fixture.detectChanges()

        paramMapSubject.next(convertToParamMap({}))
        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(1)
        expect(component.openedTabs[0].form.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[0].form.transactionId).toBe(123)

        component.onHostFormSubmit(component.openedTabs[0].form)
        tick()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)

        expect(dhcpApi.createHostDelete).not.toHaveBeenCalled()
    }))

    it('should cancel transaction when a tab is closed', fakeAsync(() => {
        const createHostBeginResp: any = {
            id: 123,
            subnets: [
                {
                    id: 1,
                    subnet: '192.0.2.0/24',
                    localSubnets: [
                        {
                            daemonId: 1,
                        },
                    ],
                },
            ],
            daemons: [
                {
                    id: 1,
                    name: 'dhcp4',
                    app: {
                        name: 'first',
                    },
                },
            ],
        }
        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(createHostBeginResp))
        spyOn(dhcpApi, 'createHostDelete').and.returnValue(of(okResp))

        paramMapSubject.next(convertToParamMap({ id: 'new' }))
        tick()
        fixture.detectChanges()

        paramMapSubject.next(convertToParamMap({}))
        tick()
        fixture.detectChanges()

        expect(component.openedTabs.length).toBe(1)
        expect(component.openedTabs[0].form.hasOwnProperty('transactionId')).toBeTrue()
        expect(component.openedTabs[0].form.transactionId).toBe(123)

        component.closeHostTab(null, 1)
        tick()
        fixture.detectChanges()
        expect(component.tabs.length).toBe(1)
        expect(component.activeTabIndex).toBe(0)

        expect(dhcpApi.createHostDelete).toHaveBeenCalled()
    }))
})
