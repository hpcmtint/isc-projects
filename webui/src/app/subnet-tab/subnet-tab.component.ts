import { Component, Input } from '@angular/core'
import { LocalSubnet, Subnet } from '../backend'
import { getGrafanaSubnetTooltip, getGrafanaUrl } from '../utils'

@Component({
    selector: 'app-subnet-tab',
    templateUrl: './subnet-tab.component.html',
    styleUrls: ['./subnet-tab.component.sass'],
})
export class SubnetTabComponent {
    /**
     * Link to Grafana Dashboard.
     */
    @Input() grafanaUrl: string

    /**
     * Subnet data.
     */
    @Input() subnet: Subnet

    // Return a sorted list of the local subnet statistic keys.
    get localSubnetStatisticKeys(): string[] {
        // Convert set to array.
        return Array.from(
            // Make a set of statistic object keys to remove duplicates.
            new Set(
                (this.subnet.localSubnets ?? [])
                    // Take into account only the entries containing the statistics.
                    .filter((l) => l.stats)
                    // Extract all keys from all statistic objects.
                    .map((l) => Object.keys(l.stats))
                    // Merge the keys into a single list.
                    .reduce((acc, val) => {
                        acc.push(...val)
                        return acc
                    }, [])
            )
        ).sort()
    }

    /**
     * Return true if the subnet is IPv6.
     */
    get isIPv6(): boolean {
        return this.subnet.subnet.includes(':')
    }

    /**
     * Create a link to Grafana for a specific local subnet.
     * @param lsn Local subnet
     * @returns Link to the specific entry in the Grafana.
     */
    getGrafanaUrl(lsn: LocalSubnet): string {
        return getGrafanaUrl(
            this.grafanaUrl,
            // DHCPv6 is not supported by the helper function.
            'dhcp4',
            lsn.id,
            lsn.machineHostname
        )
    }

    /**
     * Create a tooltip text for a specific local subnet.
     * @param lsn Local subnet
     * @returns Tooltip content.
     */
    getGrafanaTooltip(lsn: LocalSubnet): string {
        return getGrafanaSubnetTooltip(lsn.id, lsn.machineHostname)
    }
}
