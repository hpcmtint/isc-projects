import { Pipe, PipeTransform } from '@angular/core'

@Pipe({
    name: 'humanCount',
})
export class HumanCountPipe implements PipeTransform {
    transform(count: any): string {
        if (typeof count === 'string') {
            try {
                count = BigInt(count)
            } catch {
                // Cannot convert, keep it as is.
            }
        }

        if (count == null || (typeof count !== 'number' && typeof count !== 'bigint') || Number.isNaN(count)) {
            return count + '' // Casting to string safe for null and undefined
        }

        const units = ['k', 'M', 'G', 'T', 'P', 'E', 'Z', 'Y']
        let u = -1
        while (count >= 1000 && u < units.length - 1) {
            if (typeof count === 'number') {
                count /= 1000
            } else {
                count /= BigInt(1000)
            }
            ++u
        }

        let countStr = ''
        if (typeof count === 'number') {
            countStr = count.toFixed(1)
        } else {
            countStr = count.toString()
        }
        return countStr + (u >= 0 ? units[u] : '')
    }
}
