import { Component, Input } from '@angular/core'
import { DelegatedPrefix } from '../backend'
import { formatShortExcludedPrefix } from '../utils'

/**
 * Displays the delegated prefix in a bar form.
 * Supports the delegated prefix with an excluded part.
 * See: [RFC 6603](https://www.rfc-editor.org/rfc/rfc6603.html).
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
     */
    get excludedPrefixShorten(): string {
        return formatShortExcludedPrefix(this.prefix.prefix, this.prefix.excludedPrefix)
    }

    /**
     * Returns true if the excluded prefix is not empty.
     */
    get hasExcluded() {
        return !!this.prefix.excludedPrefix
    }
}
