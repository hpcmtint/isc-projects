import { CapitalizeFirstPipe } from './capitalize-first.pipe'

describe('CapitalizeFirstPipe', () => {
    it('create an instance', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe).toBeTruthy()
    })

    it('capitalize null should return an empty string', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform(null)).toBe('')
    })

    it('capitalize an empty string should return an empty string', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform('')).toBe('')
    })

    it('capitalize a number string should return the same number', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform('42')).toBe('42')
    })

    it('capitalize a string should returns a string with first upper character', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform('foo')).toBe('Foo')
    })

    it('capitalize a multi-word string should change only the first word', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform('foo bar BAZ')).toBe('Foo bar BAZ')
    })

    it('capitalize a string that starts with upper case should not change the string', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform('Foo')).toBe('Foo')
    })

    it('capitalize a unicode string should work well', () => {
        const pipe = new CapitalizeFirstPipe()
        expect(pipe.transform('ξσ')).toBe('Ξσ') // Greek Xi and Sigma.
    })
})
