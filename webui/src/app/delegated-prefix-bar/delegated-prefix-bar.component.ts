import { Component, Input } from '@angular/core'
import { DelegatedPrefix } from '../backend'

/**
 * Displays the delegated prefix in a bar form.
 * Supports the delegated prefix with an excluded part.
 */
@Component({
    selector: 'app-delegated-prefix-bar',
    templateUrl: './delegated-prefix-bar.component.html',
    styleUrls: ['./delegated-prefix-bar.component.sass'],
})
export class DelegatedPrefixBarComponent {
    @Input() prefix: DelegatedPrefix

    /**
     * Returns the short representation of the excluded prefix.
     * The common octet pairs with the main prefix are replaced by ~.
     *
     * E.g.: for the 'fe80::/64' main prefix and the 'fe80:42::/80' excluded
     * prefix the shorten form is: '~:42::/80'.
     *
     * It isn't any well-known convention, just a simple idea to limit the
     * length of the bar.
     */
    get excludedPrefixShorten(): string {
        if (!this.prefix.excludedPrefix) {
            return ''
        }

        // Split the network and length.
        let [baseNetwork, _] = this.prefix.prefix.split('/')
        let [excludedNetwork, excludedLen] = this.prefix.excludedPrefix.split('/')

        // Trim the leading double colon.
        if (baseNetwork.endsWith('::')) {
            baseNetwork = baseNetwork.slice(0, baseNetwork.length - 1)
        }

        // Check if the excluded prefix starts with the base prefix.
        // It should be always true for valid data.
        if (excludedNetwork.startsWith(baseNetwork)) {
            // Replace the common part with ~.
            excludedNetwork = excludedNetwork.slice(baseNetwork.length)
            return `~:${excludedNetwork}/${excludedLen}`
        }

        // Fallback to full excluded prefix.
        return this.prefix.excludedPrefix
    }

    // Returns true if the excluded prefix is not empty.
    get hasExcluded() {
        return !!this.prefix.excludedPrefix
    }
}
