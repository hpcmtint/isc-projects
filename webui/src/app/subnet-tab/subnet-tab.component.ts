import { Component, Input } from '@angular/core'
import { Subnet } from '../backend'

@Component({
    selector: 'app-subnet-tab',
    templateUrl: './subnet-tab.component.html',
    styleUrls: ['./subnet-tab.component.sass'],
})
export class SubnetTabComponent {
    constructor() {}

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

    get isIPv6(): boolean {
        return this.subnet.subnet.includes(':')
    }
}
