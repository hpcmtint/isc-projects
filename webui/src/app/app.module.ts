// Angular modules
import { BrowserModule } from '@angular/platform-browser'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { NgModule } from '@angular/core'
import { HTTP_INTERCEPTORS, HttpClientModule } from '@angular/common/http'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'

// Other 3rd-party modules
import { FontAwesomeModule } from '@fortawesome/angular-fontawesome'

// PrimeNG modules
import { ButtonModule } from 'primeng/button'
import { MenubarModule } from 'primeng/menubar'
import { PanelModule } from 'primeng/panel'
import { TableModule } from 'primeng/table'
import { TabViewModule } from 'primeng/tabview'
import { ProgressBarModule } from 'primeng/progressbar'
import { DialogModule } from 'primeng/dialog'
import { InputTextModule } from 'primeng/inputtext'
import { DropdownModule } from 'primeng/dropdown'
import { ConfirmationService, MessageService } from 'primeng/api'
import { ToastModule } from 'primeng/toast'
import { MessageModule } from 'primeng/message'
import { MessagesModule } from 'primeng/messages'
import { TabMenuModule } from 'primeng/tabmenu'
import { MenuModule } from 'primeng/menu'
import { InplaceModule } from 'primeng/inplace'
import { ChipModule } from 'primeng/chip'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { TooltipModule } from 'primeng/tooltip'
import { PasswordModule } from 'primeng/password'
import { CardModule } from 'primeng/card'
import { SplitButtonModule } from 'primeng/splitbutton'
import { FieldsetModule } from 'primeng/fieldset'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { InputSwitchModule } from 'primeng/inputswitch'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { PaginatorModule } from 'primeng/paginator'
import { SelectButtonModule } from 'primeng/selectbutton'
import { DividerModule } from 'primeng/divider'
import { TagModule } from 'primeng/tag'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { MultiSelectModule } from 'primeng/multiselect'
import { CheckboxModule } from 'primeng/checkbox'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { InputTextareaModule } from 'primeng/inputtextarea'
import { TreeModule } from 'primeng/tree'
import { DataViewModule } from 'primeng/dataview'
import { ChipsModule } from 'primeng/chips'

// Generated API modules
import { ApiModule, BASE_PATH, Configuration, ConfigurationParameters } from './backend'

// Stork modules
import { environment } from './../environments/environment'
import { AppRoutingModule } from './app-routing.module'
import { AppComponent } from './app.component'
import { AuthInterceptor } from './auth-interceptor'
import { AuthService } from './auth.service'
import { LoginScreenComponent } from './login-screen/login-screen.component'
import { DashboardComponent } from './dashboard/dashboard.component'
import { HostsTableComponent } from './hosts-table/hosts-table.component'
import { SwaggerUiComponent } from './swagger-ui/swagger-ui.component'
import { MachinesPageComponent } from './machines-page/machines-page.component'
import { LocaltimePipe } from './localtime.pipe'
import { UsersPageComponent } from './users-page/users-page.component'
import { AppsPageComponent } from './apps-page/apps-page.component'
import { Bind9AppTabComponent } from './bind9-app-tab/bind9-app-tab.component'
import { KeaAppTabComponent } from './kea-app-tab/kea-app-tab.component'
import { PasswordChangePageComponent } from './password-change-page/password-change-page.component'
import { ProfilePageComponent } from './profile-page/profile-page.component'
import { SettingsMenuComponent } from './settings-menu/settings-menu.component'
import { HaStatusComponent } from './ha-status/ha-status.component'
import { SubnetsPageComponent } from './subnets-page/subnets-page.component'
import { SharedNetworksPageComponent } from './shared-networks-page/shared-networks-page.component'
import { SubnetBarComponent } from './subnet-bar/subnet-bar.component'
import { HostsPageComponent } from './hosts-page/hosts-page.component'
import { SettingsPageComponent } from './settings-page/settings-page.component'
import { HelpTipComponent } from './help-tip/help-tip.component'
import { GlobalSearchComponent } from './global-search/global-search.component'
import { HaStatusPanelComponent } from './ha-status-panel/ha-status-panel.component'
import { EventTextComponent } from './event-text/event-text.component'
import { EntityLinkComponent } from './entity-link/entity-link.component'
import { EventsPanelComponent } from './events-panel/events-panel.component'
import { ForbiddenPageComponent } from './forbidden-page/forbidden-page.component'
import { LogViewPageComponent } from './log-view-page/log-view-page.component'
import { AppDaemonsStatusComponent } from './app-daemons-status/app-daemons-status.component'
import { BreadcrumbsComponent } from './breadcrumbs/breadcrumbs.component'
import { EventsPageComponent } from './events-page/events-page.component'
import { RenameAppDialogComponent } from './rename-app-dialog/rename-app-dialog.component'
import { LeaseSearchPageComponent } from './lease-search-page/lease-search-page.component'
import { JsonTreeComponent } from './json-tree/json-tree.component'
import { JsonTreeRootComponent } from './json-tree-root/json-tree-root.component'
import { KeaDaemonConfigurationPageComponent } from './kea-daemon-configuration-page/kea-daemon-configuration-page.component'
import { HostTabComponent } from './host-tab/host-tab.component'
import { ConfigReviewPanelComponent } from './config-review-panel/config-review-panel.component'
import { IdentifierComponent } from './identifier/identifier.component'
import { AppOverviewComponent } from './app-overview/app-overview.component'
import { HostFormComponent } from './host-form/host-form.component'
import { DhcpOptionFormComponent } from './dhcp-option-form/dhcp-option-form.component'
import { DhcpOptionSetFormComponent } from './dhcp-option-set-form/dhcp-option-set-form.component'
import { ConfigCheckerPreferenceUpdaterComponent } from './config-checker-preference-updater/config-checker-preference-updater.component'
import { ConfigCheckerPreferencePickerComponent } from './config-checker-preference-picker/config-checker-preference-picker.component'
import { ConfigCheckerPreferencePageComponent } from './config-checker-preference-page/config-checker-preference-page.component'
import { DhcpOptionSetViewComponent } from './dhcp-option-set-view/dhcp-option-set-view.component'
import { DhcpClientClassSetFormComponent } from './dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { DhcpClientClassSetViewComponent } from './dhcp-client-class-set-view/dhcp-client-class-set-view.component'
import { SubnetTabComponent } from './subnet-tab/subnet-tab.component'
import { KeyValuePipe, PercentPipe } from '@angular/common'

export function cfgFactory() {
    const params: ConfigurationParameters = {
        apiKeys: {},
        withCredentials: true,
    }
    return new Configuration(params)
}

@NgModule({
    declarations: [
        AppComponent,
        LoginScreenComponent,
        DashboardComponent,
        HostsTableComponent,
        SwaggerUiComponent,
        MachinesPageComponent,
        LocaltimePipe,
        UsersPageComponent,
        AppsPageComponent,
        Bind9AppTabComponent,
        KeaAppTabComponent,
        PasswordChangePageComponent,
        ProfilePageComponent,
        SettingsMenuComponent,
        HaStatusComponent,
        SubnetsPageComponent,
        SharedNetworksPageComponent,
        SubnetBarComponent,
        HostsPageComponent,
        SettingsPageComponent,
        HelpTipComponent,
        GlobalSearchComponent,
        HaStatusPanelComponent,
        EventTextComponent,
        EntityLinkComponent,
        EventsPanelComponent,
        ForbiddenPageComponent,
        LogViewPageComponent,
        AppDaemonsStatusComponent,
        BreadcrumbsComponent,
        EventsPageComponent,
        RenameAppDialogComponent,
        LeaseSearchPageComponent,
        JsonTreeComponent,
        JsonTreeRootComponent,
        KeaDaemonConfigurationPageComponent,
        HostTabComponent,
        ConfigReviewPanelComponent,
        IdentifierComponent,
        AppOverviewComponent,
        HostFormComponent,
        DhcpOptionFormComponent,
        DhcpOptionSetFormComponent,
        DhcpOptionSetViewComponent,
        ConfigCheckerPreferencePickerComponent,
        ConfigCheckerPreferenceUpdaterComponent,
        ConfigCheckerPreferencePageComponent,
        DhcpClientClassSetFormComponent,
        DhcpClientClassSetViewComponent,
        SubnetTabComponent,
    ],
    imports: [
        BrowserModule,
        BrowserAnimationsModule,
        HttpClientModule,
        AppRoutingModule,
        FormsModule,
        ReactiveFormsModule,

        FontAwesomeModule,

        ApiModule.forRoot(cfgFactory),

        ButtonModule,
        MenubarModule,
        PanelModule,
        TableModule,
        TabViewModule,
        ProgressBarModule,
        DialogModule,
        InputTextModule,
        DropdownModule,
        ToastModule,
        MessageModule,
        MessagesModule,
        TabMenuModule,
        MenuModule,
        InplaceModule,
        ProgressSpinnerModule,
        TooltipModule,
        PasswordModule,
        CardModule,
        SplitButtonModule,
        FieldsetModule,
        OverlayPanelModule,
        InputSwitchModule,
        BreadcrumbModule,
        PaginatorModule,
        SelectButtonModule,
        DividerModule,
        TagModule,
        ToggleButtonModule,
        MultiSelectModule,
        CheckboxModule,
        ConfirmDialogModule,
        InputTextareaModule,
        TreeModule,
        ChipModule,
        ChipsModule,
        DataViewModule,
        ToggleButtonModule,
        PercentPipe,
        KeyValuePipe
    ],
    providers: [
        {
            provide: HTTP_INTERCEPTORS,
            useClass: AuthInterceptor,
            multi: true,
        },
        {
            provide: BASE_PATH,
            useValue: environment.apiUrl,
        },
        ConfirmationService,
        MessageService,
    ],
    bootstrap: [AppComponent],
})
export class AppModule {}
