import { Component, Input } from '@angular/core'
import { DelegatedPrefix } from '../backend'

@Component({
    selector: 'app-delegated-prefix-bar',
    templateUrl: './delegated-prefix-bar.component.html',
    styleUrls: ['./delegated-prefix-bar.component.sass'],
})
export class DelegatedPrefixBarComponent {
    @Input() prefix: DelegatedPrefix

    constructor() {}

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

        if (excludedNetwork.startsWith(baseNetwork)) {
            excludedNetwork = excludedNetwork.slice(baseNetwork.length)
            return `~:${excludedNetwork}/${excludedLen}`
        }

        return this.prefix.excludedPrefix
    }

    get hasExcluded() {
        return !!this.prefix.excludedPrefix
    }
}
