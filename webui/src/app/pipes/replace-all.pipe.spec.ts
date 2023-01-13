import { ReplaceAllPipe } from './replace-all.pipe'

describe('ReplaceAllPipe', () => {
    it('create an instance', () => {
        const pipe = new ReplaceAllPipe()
        expect(pipe).toBeTruthy()
    })

    it('should replace a raw string properly', () => {
        const pipe = new ReplaceAllPipe()
        expect(pipe.transform("foo-bar-baz", "ba", "ki")).toBe("foo-kir-kiz")
    })

    it('should not change the string for a pattern that does not occur', () => {
        const pipe = new ReplaceAllPipe()
        expect(pipe.transform('foo', 'bar', 'baz')).toBe('foo')
    })
})
