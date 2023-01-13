import { Pipe, PipeTransform } from '@angular/core'

/**
 * The pipeline replaces all occurrences of a specific pattern.
 */
@Pipe({
    name: 'replaceAll',
})
export class ReplaceAllPipe implements PipeTransform {
    /**
     * Replaces all occurrences of a given pattern in the input data
     * with a specific placeholder.
     * @param value Input text.
     * @param from The string to replace.
     * @param to The string with replacement value.
     * @returns The changed input string.
     */
    transform(value: string, from: string, to: string): string {
        return value.replace(new RegExp(from, 'g'), to)
    }
}
