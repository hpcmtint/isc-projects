import { Meta, Story, moduleMetadata } from '@storybook/angular'
import { SubnetTabComponent } from './subnet-tab.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { ReplaceAllPipe } from '../pipes/replace-all.pipe'
import { CapitalizeFirstPipe } from '../pipes/capitalize-first.pipe'
import { HumanCountComponent } from '../human-count/human-count.component'
import { NumberPipe } from '../pipes/number.pipe'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { FieldsetModule } from 'primeng/fieldset'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { DatePipe, KeyValuePipe, PercentPipe } from '@angular/common'
import { RouterTestingModule } from '@angular/router/testing'
import { TooltipModule } from 'primeng/tooltip'
import { Subnet } from '../backend'
import { IdentifierComponent } from '../identifier/identifier.component'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { FormsModule } from '@angular/forms'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { DelegatedPrefixBarComponent } from '../delegated-prefix-bar/delegated-prefix-bar.component'

export default {
    title: 'App/SubnetTab',
    component: SubnetTabComponent,
    decorators: [
        moduleMetadata({
            declarations: [
                EntityLinkComponent,
                ReplaceAllPipe,
                CapitalizeFirstPipe,
                HumanCountComponent,
                NumberPipe,
                SubnetBarComponent,
                IdentifierComponent,
                HumanCountPipe,
                DelegatedPrefixBarComponent,
            ],
            imports: [
                FieldsetModule,
                NoopAnimationsModule,
                PercentPipe,
                KeyValuePipe,
                DatePipe,
                RouterTestingModule,
                TooltipModule,
                ToggleButtonModule,
                FormsModule,
            ],
        }),
    ],
} as Meta

const Template: Story<SubnetTabComponent> = (args: SubnetTabComponent) => ({
    props: args,
})

export const ipv4NoData = Template.bind({})
ipv4NoData.args = {
    subnet: {
        subnet: '10.0.0.0/8',
    } as Subnet,
}

export const ipv6NoData = Template.bind({})
ipv6NoData.args = {
    subnet: {
        subnet: 'fe80::/64',
    } as Subnet,
}

export const ipv4FullData = Template.bind({})
ipv4FullData.args = {
    subnet: {
        id: 55,
        subnet: '10.0.0.0/8',
        addrUtilization: 85,
        clientClass: 'my-class-00',
        hosts: [
            {
                id: 1,
                hostIdentifiers: [
                    {
                        idHexValue: '00:01:02:03:04:05',
                        idType: 'circuit-id',
                    },
                    {
                        idHexValue: '75:76:77:78:79:80:81',
                        idType: 'flex-id',
                    },
                ],
            },
            {
                id: 2,
                hostIdentifiers: [
                    {
                        idHexValue: '76:43:56:57:89',
                        idType: 'hw-address',
                    },
                ],
            },
            {
                id: 3,
                hostIdentifiers: [
                    {
                        idHexValue: '73:30:6d:45:56:61:4c:75:65',
                        idType: 'flex-id',
                    },
                ],
            },
        ],
        localSubnets: [
            {
                appId: 22,
                appName: 'kea@machine1',
                daemonId: 23,
                daemonName: 'dhcp4',
                machineAddress: 'http://machine1',
                machineHostname: 'machine1',
                id: 24,
                stats: {
                    'total-addresses': '100000',
                    'assigned-addresses': '85000',
                    'declined-addresses': '10000',
                    foobar: '42',
                },
                statsCollectedAt: '2022-12-29T17:09:00',
            },
            {
                appId: 25,
                appName: 'kea@machine2',
                daemonId: 26,
                daemonName: 'dhcp4',
                machineAddress: 'http://machine2',
                machineHostname: 'machine2',
                id: 24,
                stats: {
                    'total-addresses': '100000',
                    'assigned-addresses': '85000',
                    'declined-addresses': '10000',
                    foobar: '42',
                },
                statsCollectedAt: '2022-12-29T16:09:00',
            },
        ],
        stats: {
            'total-addresses': '200000',
            'assigned-addresses': '170000',
            'declined-addresses': '20000',
            foobar: '82',
        },
        statsCollectedAt: '2022-12-29T18:09:00',
        pools: [
            '10.1.0.1-10.1.0.100',
            '10.2.0.1-10.2.0.100',
            '10.3.0.1-10.3.0.100',
            '10.4.0.1-10.4.0.100',
            '10.5.0.1-10.5.0.100',
            '10.6.0.1-10.6.0.100',
            '10.7.0.1-10.7.0.100',
            '10.8.0.1-10.8.0.100',
            '10.9.0.1-10.9.0.100',
            '10.10.0.1-10.10.0.100',
        ],
        sharedNetwork: 'frog',
    } as Subnet,
    grafanaUrl: 'http://localhost:3000',
}

export const ipv6FullData = Template.bind({})
ipv6FullData.args = {
    subnet: {
        id: 55,
        subnet: 'fe80::/64',
        addrUtilization: 85,
        pdUtilization: 95,
        clientClass: 'my-class-00',
        hosts: [
            {
                id: 1,
                hostIdentifiers: [
                    {
                        idHexValue: '00:01:02:03:04:05',
                        idType: 'circuit-id',
                    },
                    {
                        idHexValue: '75:76:77:78:79:80:81',
                        idType: 'flex-id',
                    },
                ],
            },
            {
                id: 2,
                hostIdentifiers: [
                    {
                        idHexValue: '76:43:56:57:89',
                        idType: 'hw-address',
                    },
                ],
            },
            {
                id: 3,
                hostIdentifiers: [
                    {
                        idHexValue: '73:30:6d:45:56:61:4c:75:65',
                        idType: 'flex-id',
                    },
                ],
            },
        ],
        localSubnets: [
            {
                appId: 22,
                appName: 'kea@machine1',
                daemonId: 23,
                daemonName: 'dhcp6',
                machineAddress: 'http://machine1',
                machineHostname: 'machine1',
                id: 24,
                stats: {
                    'total-nas': '100000000',
                    'assigned-nas': '85000000',
                    'declined-nas': '10000000',
                    foobar: '42',
                    'total-pds': '10000000',
                    'assigned-pds': '9500000',
                },
                statsCollectedAt: '2022-12-29T17:09:00',
            },
            {
                appId: 25,
                appName: 'kea@machine2',
                daemonId: 26,
                daemonName: 'dhcp6',
                machineAddress: 'http://machine2',
                machineHostname: 'machine2',
                id: 24,
                stats: {
                    'total-nas': '100000000',
                    'assigned-nas': '85000000',
                    'declined-nas': '10000000',
                    'total-pds': '10000000',
                    'assigned-pds': '9500000',
                    foobar: '42',
                },
                statsCollectedAt: '2022-12-29T16:09:00',
            },
        ],
        stats: {
            'total-nas': '200000000',
            'assigned-nas': '170000000',
            'declined-nas': '20000000',
            'total-pds': '20000000',
            'assigned-pds': '19000000',
            foobar: '82',
        },
        statsCollectedAt: '2022-12-29T18:09:00',
        pools: [
            'fe80:1::1-fe80:1::100',
            'fe80:2::1-fe80:2::100',
            'fe80:3::1-fe80:3::100',
            'fe80:4::1-fe80:4::100',
            'fe80:5::1-fe80:5::100',
            'fe80:6::1-fe80:6::100',
            'fe80:7::1-fe80:7::100',
            'fe80:8::1-fe80:8::100',
            'fe80:9::1-fe80:9::100',
            'fe80:10::1-fe80:10::100',
            'fe80:11::1-fe80:11::100',
            'fe80:12::1-fe80:12::100',
            'fe80:13::1-fe80:13::100',
            'fe80:14::1-fe80:14::100',
            'fe80:15::1-fe80:15::100',
            'fe80:16::1-fe80:16::100',
            'fe80:17::1-fe80:17::100',
            'fe80:18::1-fe80:18::100',
            'fe80:19::1-fe80:19::100',
        ],
        sharedNetwork: 'frog',
        prefixDelegationPools: [
            {
                prefix: 'fe80:100:1::/80',
                delegatedLength: 96,
            },
            {
                prefix: 'fe80:100:2::/80',
                delegatedLength: 96,
            },
            {
                prefix: 'fe80:100:3::/80',
                delegatedLength: 96,
            },
            {
                prefix: 'fe80:100:4::/80',
                delegatedLength: 96,
                excludedPrefix: 'fe80:100:4:ffff::/112',
            },
            {
                prefix: 'fe80:100:5::/80',
                delegatedLength: 96,
                excludedPrefix: 'fe80:100:5:ffff::/112',
            },
        ],
    } as Subnet,
    grafanaUrl: 'http://localhost:3000',
}
