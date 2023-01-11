import { clamp, stringToHex, getErrorMessage, humanCount, formatShortExcludedPrefix } from './utils'

describe('utils', () => {
    it('clamps should return return proper number', () => {
        // Integers - in range
        expect(clamp(1, 0, 2)).toBe(1)
        // Integers - on lower bound
        expect(clamp(0, 0, 2)).toBe(0)
        // Integers - below lower bound
        expect(clamp(-1, 0, 2)).toBe(0)
        // Integers - on upper bound
        expect(clamp(2, 0, 2)).toBe(2)
        // Integers - above upper bound
        expect(clamp(3, 0, 2)).toBe(2)

        // Floats - in range
        expect(clamp(1.5, 0.4, 2.22)).toBe(1.5)
        // Floats - on lower bound
        expect(clamp(0.4, 0.4, 2.22)).toBe(0.4)
        // Floats - below lower bound
        expect(clamp(0.1, 0.4, 2.22)).toBe(0.4)
        // Floats - on upper bound
        expect(clamp(2.22, 0.4, 2.22)).toBe(2.22)
        // Floats - above upper bound
        expect(clamp(3.22, 0.4, 2.22)).toBe(2.22)

        // Floats - value as negative infinity
        expect(clamp(Number.NEGATIVE_INFINITY, 0, 1)).toBe(0)
        // Floats - value as positive infinity
        expect(clamp(Number.POSITIVE_INFINITY, 0, 1)).toBe(1)
        // Floats - bounds as infinities
        expect(clamp(3, Number.NEGATIVE_INFINITY, Number.POSITIVE_INFINITY)).toBe(3)
    })

    it('human count should return proper string for any number', () => {
        // Arrange
        const int = 12345678
        const float = 1234567890
        const bigInt = BigInt('1234567890000000000000000000000000')
        const smallInt = 1
        const str = 'foo'
        const nan = Number.NaN
        const boolean = true as any
        const nullValue = null

        // Act
        const strInt = humanCount(int)
        const strFloat = humanCount(float)
        const strBigInt = humanCount(bigInt)
        const strSmallInt = humanCount(smallInt)
        const strStr = humanCount(str)
        const nanStr = humanCount(nan)
        const boolStr = humanCount(boolean)
        const nullStr = humanCount(nullValue)

        // Assert
        expect(strInt).toBe('12.3M')
        expect(strFloat).toBe('1.2G')
        expect(strBigInt).toBe('1234567890Y')
        expect(strSmallInt).toBe('1')
        expect(strStr).toBe('foo')
        expect(nanStr).toBe('NaN')
        expect(boolStr).toBe('true')
        expect(nullStr).toBe('null')
    })

    it('clamps should return return proper number', () => {
        // Integers - in range
        expect(clamp(1, 0, 2)).toBe(1)
        // Integers - on lower bound
        expect(clamp(0, 0, 2)).toBe(0)
        // Integers - below lower bound
        expect(clamp(-1, 0, 2)).toBe(0)
        // Integers - on upper bound
        expect(clamp(2, 0, 2)).toBe(2)
        // Integers - above upper bound
        expect(clamp(3, 0, 2)).toBe(2)

        // Floats - in range
        expect(clamp(1.5, 0.4, 2.22)).toBe(1.5)
        // Floats - on lower bound
        expect(clamp(0.4, 0.4, 2.22)).toBe(0.4)
        // Floats - below lower bound
        expect(clamp(0.1, 0.4, 2.22)).toBe(0.4)
        // Floats - on upper bound
        expect(clamp(2.22, 0.4, 2.22)).toBe(2.22)
        // Floats - above upper bound
        expect(clamp(3.22, 0.4, 2.22)).toBe(2.22)

        // Floats - value as negative infinity
        expect(clamp(Number.NEGATIVE_INFINITY, 0, 1)).toBe(0)
        // Floats - value as positive infinity
        expect(clamp(Number.POSITIVE_INFINITY, 0, 1)).toBe(1)
        // Floats - bounds as infinities
        expect(clamp(3, Number.NEGATIVE_INFINITY, Number.POSITIVE_INFINITY)).toBe(3)
    })

    it('converts text to a string of hexadecimal digits', () => {
        // Lower case.
        expect(stringToHex('abcdefghi')).toBe('61:62:63:64:65:66:67:68:69')
        // Upper case with non-default separator.
        expect(stringToHex('MY OH MY', '-')).toBe('4d-59-20-4f-48-20-4d-59')
        // Empty string.
        expect(stringToHex('')).toBe('')
    })

    it('retrieves the error message properly', () => {
        const testCases: (object | [object, string])[] = [
            // Error wrapper
            {
                error: {
                    message: 'expected message',
                },
                statusText: 'unexpected message',
                status: 500,
                message: 'unexpected message',
                cause: 'unexpected message',
                name: 'unexpected message',
                toString: () => 'unexpected message',
            },
            // HTTP error with status text
            {
                statusText: 'expected message',
                status: 500,
                message: 'unexpected message',
                cause: 'unexpected message',
                name: 'unexpected message',
                toString: () => 'unexpected message',
            },
            // HTTP error without status text
            [
                {
                    status: 500,
                    message: 'unexpected message',
                    cause: 'unexpected message',
                    name: 'unexpected message',
                    toString: () => 'unexpected message',
                },
                'status: 500',
            ],
            // Message container
            {
                message: 'expected message',
                cause: 'unexpected message',
                name: 'unexpected message',
                toString: () => 'unexpected message',
            },
            // Error with a cause
            {
                cause: 'expected message',
                name: 'unexpected message',
                toString: () => 'unexpected message',
            },
            // Error with a name
            {
                name: 'expected message',
                toString: () => 'unexpected message',
            },
            // Any object
            {
                toString: () => 'expected message',
            },
            // An error object
            Error('expected message'),
        ]

        for (const testCase of testCases) {
            let error = testCase
            let expectedMessage = 'expected message'
            if (testCase instanceof Array) {
                error = testCase[0]
                expectedMessage = testCase[1]
            }
            const actualMessage = getErrorMessage(error)
            expect(actualMessage).toBe(expectedMessage, error)
        }
    })

    it('should shorten the excluded prefix if has common part with a prefix', () => {
        const excludedPrefix = 'fe80:42::/96'
        const prefix = 'fe80::/64'
        expect(formatShortExcludedPrefix(prefix, excludedPrefix)).toBe('~:42::/96')
    })

    it('should not shorten if the excluded prefix has no common part with a prefix', () => {
        const excludedPrefix = '3001::/96'
        const prefix = 'fe80::/64'
        expect(formatShortExcludedPrefix(prefix, excludedPrefix)).toBe('3001::/96')
    })

    it('should not shorten if the prefix and excluded prefix has common part but one of them is not in a canonical form', () => {
        const excludedPrefix = 'fe80:42::/96'
        const prefix = 'fe80:0000::/64'
        expect(formatShortExcludedPrefix(prefix, excludedPrefix)).toBe('fe80:42::/96')
    })
})
