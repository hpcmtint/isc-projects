import { Pipe, PipeTransform } from '@angular/core'
import { humanCount } from '../utils'

/**
 * The pipeline to format a number or a numeric string into a human-readable
 * representation.
 */
@Pipe({
    name: 'humanCount',
})
export class HumanCountPipe implements PipeTransform {
    /**
     * Formats a provided number or numeric string using the metric prefix
     * notation.
     * @param count Number or numeric string.
     * @returns Formatted string.
     */
    transform(count: string | number | bigint | null): string {
        if (typeof count === 'string') {
            try {
                count = BigInt(count)
            } catch {
                // Cannot convert, keep it as is.
            }
        }

        return humanCount(count)
    }
}
