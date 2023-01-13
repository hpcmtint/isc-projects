import { Pipe, PipeTransform } from '@angular/core'

/**
 * The capitalize pipeline affects only the first character in the provided
 * input.
 */
@Pipe({
    name: 'capitalizeFirst',
})
export class CapitalizeFirstPipe implements PipeTransform {
    /**
     * Changes the first character to uppercase (if applicable).
     */
    transform(value: string): string {
        if (!value) {
            return ''
        }
        return value[0].toUpperCase() + value.slice(1)
    }
}
