import { CapitalizeFirstPipe } from './capitalize-first.pipe'

describe('CapitalizeFirstPipe', () => {
    it('create an instance', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe).toBeTruthy()
    })

    it('should capitalize null to an empty string', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform(null)).toBe('')
    })

    it('should capitalize an empty string to an empty string', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform('')).toBe('')
    })

    it('should capitalize a number string to the same number', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform('42')).toBe('42')
    })

    it('should capitalize a string to a string with first upper character', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform('foo')).toBe('Foo')
    })

    it('should change only the first word in a multi-word string', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform('foo bar BAZ')).toBe('Foo bar BAZ')
    })

    it('should not change a string that starts with upper case', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform('Foo')).toBe('Foo')
    })

    it('should capitalize a unicode string properly', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform('ξσ')).toBe('Ξσ') // Greek Xi and Sigma.
    })
})
