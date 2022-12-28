import { Meta, Story, moduleMetadata } from "@storybook/angular"
import { SubnetBarComponent } from "./subnet-bar.component"
import { EntityLinkComponent } from "../entity-link/entity-link.component"
import { Subnet } from "../backend"
import { RouterTestingModule } from "@angular/router/testing"
import { TooltipModule } from "primeng/tooltip"

export default {
    title: 'App/SubnetBar',
    component: SubnetBarComponent,
    decorators: [
        moduleMetadata({
            imports: [RouterTestingModule, TooltipModule],
            declarations: [EntityLinkComponent]
        }),
    ],
} as Meta

const Template: Story<SubnetBarComponent> = (args: SubnetBarComponent) => ({
    props: args,
})

export const IPv4NoStats = Template.bind({})
IPv4NoStats.args = {
    subnet: {
        id: 42,
        subnet: "42.42.0.0/16"
    } as Subnet
}

export const IPv4Stats = Template.bind({})
IPv4Stats.args = {
    subnet: {
        id: 42,
        subnet: "42.42.0.0/16",
        stats: {
            'total-addresses': 50,
            'assigned-addresses': 20,
            'declined-addresses': 5
        },
        statsCollectedAt: '2022-12-28T14:59:00'
    } as Subnet
}

export const IPv4UtilizationLow = Template.bind({})
IPv4UtilizationLow.args = {
    subnet: {
        id: 42,
        subnet: "42.42.0.0/16",
        addrUtilization: 30,
        stats: {
            'total-addresses': 100,
            'assigned-addresses': 30,
            'declined-addresses': 0
        },
        statsCollectedAt: '2022-12-28T14:59:00'
    } as Subnet
}

export const IPv4UtilizationMedium = Template.bind({})
IPv4UtilizationMedium.args = {
    subnet: {
        id: 42,
        subnet: "42.42.0.0/16",
        addrUtilization: 85,
        stats: {
            'total-addresses': 100,
            'assigned-addresses': 85,
            'declined-addresses': 0
        },
        statsCollectedAt: '2022-12-28T14:59:00'
    } as Subnet
}

export const IPv4UtilizationHigh = Template.bind({})
IPv4UtilizationHigh.args = {
    subnet: {
        id: 42,
        subnet: "42.42.0.0/16",
        addrUtilization: 95,
        stats: {
            'total-addresses': 100,
            'assigned-addresses': 95,
            'declined-addresses': 0
        },
        statsCollectedAt: '2022-12-28T14:59:00'
    } as Subnet
}

export const IPv4UtilizationExceed = Template.bind({})
IPv4UtilizationExceed.args = {
    subnet: {
        id: 42,
        subnet: "42.42.0.0/16",
        addrUtilization: 110,
        stats: {
            'total-addresses': 100,
            'assigned-addresses': 110,
            'declined-addresses': 0
        },
        statsCollectedAt: '2022-12-28T14:59:00'
    } as Subnet
}

export const IPv6NoStats = Template.bind({})
IPv6NoStats.args = {
    subnet: {
        id: 42,
        subnet: "3001:1::/64"
    } as Subnet
}

export const IPv6Stats = Template.bind({})
IPv6Stats.args = {
    subnet: {
        id: 42,
        subnet: "3001:1::/64",
        stats: {
            'total-nas': 50,
            'assigned-nas': 20,
            'declined-nas': 5,
            'total-pds': 70,
            'assigned-pds': 30
        },
        statsCollectedAt: '2022-12-28T14:59:00'
    } as Subnet
}

export const IPv6UtilizationAddressLow = Template.bind({})
IPv6UtilizationAddressLow.args = {
    subnet: {
        id: 42,
        subnet: "3001:1::/64",
        stats: {
            'total-nas': 100,
            'assigned-nas': 20,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 80
        },
        addrUtilization: 20,
        statsCollectedAt: '2022-12-28T14:59:00'
    } as Subnet
}

export const IPv6UtilizationAddressMedium = Template.bind({})
IPv6UtilizationAddressMedium.args = {
    subnet: {
        id: 42,
        subnet: "3001:1::/64",
        stats: {
            'total-nas': 100,
            'assigned-nas': 85,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 162
        },
        addrUtilization: 85,
        statsCollectedAt: '2022-12-28T14:59:00'
    } as Subnet
}

export const IPv6UtilizationAddressHigh = Template.bind({})
IPv6UtilizationAddressHigh.args = {
    subnet: {
        id: 42,
        subnet: "3001:1::/64",
        stats: {
            'total-nas': 100,
            'assigned-nas': 95,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 182
        },
        addrUtilization: 95,
        statsCollectedAt: '2022-12-28T14:59:00'
    } as Subnet
}

export const IPv6UtilizationAddressExceed = Template.bind({})
IPv6UtilizationAddressExceed.args = {
    subnet: {
        id: 42,
        subnet: "3001:1::/64",
        stats: {
            'total-nas': 100,
            'assigned-nas': 110,
            'declined-nas': 0,
            'total-pds': 200,
            'assigned-pds': 250
        },
        addrUtilization: 110,
        statsCollectedAt: '2022-12-28T14:59:00'
    } as Subnet
}