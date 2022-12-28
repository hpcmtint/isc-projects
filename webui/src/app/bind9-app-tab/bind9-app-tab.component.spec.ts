import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { Bind9AppTabComponent } from './bind9-app-tab.component'
import { RouterLink, Router, RouterModule, ActivatedRoute } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { TooltipModule } from 'primeng/tooltip'
import { TabViewModule } from 'primeng/tabview'
import { MessageService } from 'primeng/api'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { MockLocationStrategy } from '@angular/common/testing'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { of, throwError } from 'rxjs'

import { ServicesService, UsersService } from '../backend'
import { ServerDataService } from '../server-data.service'
import { RenameAppDialogComponent } from '../rename-app-dialog/rename-app-dialog.component'
import { DialogModule } from 'primeng/dialog'
import { FormsModule } from '@angular/forms'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { AppOverviewComponent } from '../app-overview/app-overview.component'
import { PanelModule } from 'primeng/panel'

class Daemon {
    name = 'bind9'
}

class Details {
    daemon: Daemon = new Daemon()
}

class Machine {
    id = 1
}

class App {
    id = 1
    name = ''
    machine = new Machine()
    details = new Details()
}

class AppTab {
    app: App = new App()
}

describe('Bind9AppTabComponent', () => {
    let component: Bind9AppTabComponent
    let fixture: ComponentFixture<Bind9AppTabComponent>
    let servicesApi: ServicesService
    let serverData: ServerDataService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [UsersService, ServicesService, MessageService, MockLocationStrategy],
            imports: [
                HttpClientTestingModule,
                FormsModule,
                RouterTestingModule,
                TooltipModule,
                TabViewModule,
                DialogModule,
                NoopAnimationsModule,
                PanelModule,
            ],
            declarations: [Bind9AppTabComponent, LocaltimePipe, RenameAppDialogComponent, AppOverviewComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(Bind9AppTabComponent)
        component = fixture.componentInstance
        servicesApi = fixture.debugElement.injector.get(ServicesService)
        serverData = fixture.debugElement.injector.get(ServerDataService)
        const appTab = new AppTab()
        component.refreshedAppTab = of(appTab)
        component.appTab = appTab
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display rename dialog', () => {
        const fakeAppsNames = new Map()
        spyOn(serverData, 'getAppsNames').and.returnValue(of(fakeAppsNames))
        const fakeMachinesAddresses = new Set()
        spyOn(serverData, 'getMachinesAddresses').and.returnValue(of(fakeMachinesAddresses))
        expect(component.appRenameDialogVisible).toBeFalse()
        component.showRenameAppDialog()
        expect(serverData.getAppsNames).toHaveBeenCalled()
        expect(serverData.getMachinesAddresses).toHaveBeenCalled()
        // The dialog should be visible after fetching apps names and machines
        // addresses successfully.
        expect(component.appRenameDialogVisible).toBeTrue()
    })

    it('should not display rename dialog when fetching machines fails', () => {
        const fakeAppsNames = new Map()
        spyOn(serverData, 'getAppsNames').and.returnValue(of(fakeAppsNames))
        // Simulate an error while getting machines addresses.
        spyOn(serverData, 'getMachinesAddresses').and.returnValue(throwError({ status: 404 }))
        expect(component.appRenameDialogVisible).toBeFalse()
        component.showRenameAppDialog()
        expect(serverData.getAppsNames).toHaveBeenCalled()
        expect(serverData.getMachinesAddresses).toHaveBeenCalled()
        // The dialog should not be visible because there was an error.
        expect(component.appRenameDialogVisible).toBeFalse()
    })

    it('should not display rename dialog when fetching apps fails', () => {
        // Simulate an error while getting apps names.
        spyOn(serverData, 'getAppsNames').and.returnValue(throwError({ status: 404 }))
        const fakeMachinesAddresses = new Set()
        spyOn(serverData, 'getMachinesAddresses').and.returnValue(of(fakeMachinesAddresses))
        expect(component.appRenameDialogVisible).toBeFalse()
        component.showRenameAppDialog()
        expect(serverData.getAppsNames).toHaveBeenCalled()
        expect(serverData.getMachinesAddresses).toHaveBeenCalled()
        // The dialog should not be visible because there was an error.
        expect(component.appRenameDialogVisible).toBeFalse()
    })

    it('should send app rename request', () => {
        // Prepare fake success response to renameApp call.
        const fakeResponse: any = { data: {} }
        spyOn(servicesApi, 'renameApp').and.returnValue(of(fakeResponse))
        // Simulate submitting the app rename request.
        component.handleRenameDialogSubmitted('bindx@machine3')
        // Make sure that the request to rename the app was submitted.
        expect(servicesApi.renameApp).toHaveBeenCalled()
        // As a result, the app name in the tab should have been updated.
        expect(component.appTab.app.name).toBe('bindx@machine3')
    })

    it('should hide app rename dialog', () => {
        // Show the dialog box.
        component.appRenameDialogVisible = true
        fixture.detectChanges()
        spyOn(servicesApi, 'renameApp')
        // Cancel the dialog box.
        component.handleRenameDialogHidden()
        // Ensure that the dialog box is no longer visible.
        expect(component.appRenameDialogVisible).toBeFalse()
        // A request to rename the app should not be sent.
        expect(servicesApi.renameApp).not.toHaveBeenCalled()
    })
})
