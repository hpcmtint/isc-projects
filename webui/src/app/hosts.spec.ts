import { TestBed } from '@angular/core/testing'
import { Host } from './backend/model/host'
import { hasDifferentLocalHostData } from './hosts'

describe('hosts', () => {
    beforeEach(() => TestBed.configureTestingModule({}))

    it('detects differences between DHCP options', () => {
        const host: Host = {
            localHosts: [
                {
                    optionsHash: '123',
                    clientClasses: ['foo', 'bar'],
                },
                {
                    optionsHash: '234',
                    clientClasses: ['foo', 'bar'],
                },
                {
                    optionsHash: '123',
                    clientClasses: ['foo', 'bar'],
                },
            ],
        }
        expect(hasDifferentLocalHostData(host)).toBeTrue()
    })

    it('detects differences between client classes', () => {
        const host: Host = {
            localHosts: [
                {
                    optionsHash: '123',
                    clientClasses: ['foo', 'bar'],
                },
                {
                    optionsHash: '123',
                    clientClasses: ['foo'],
                },
                {
                    optionsHash: '123',
                    clientClasses: ['foo', 'bar'],
                },
            ],
        }
        expect(hasDifferentLocalHostData(host)).toBeTrue()
    })

    it('detects that there are no differences', () => {
        const host: Host = {
            localHosts: [
                {
                    optionsHash: '123',
                    clientClasses: ['foo', 'bar'],
                },
                {
                    optionsHash: '123',
                    clientClasses: ['foo', 'bar'],
                },
                {
                    optionsHash: '123',
                    clientClasses: ['foo', 'bar'],
                },
            ],
        }
        expect(hasDifferentLocalHostData(host)).toBeFalse()
    })

    it('detects no differences for all null options hashes', () => {
        const host: Host = {
            localHosts: [
                {
                    optionsHash: null,
                    clientClasses: ['foo', 'bar'],
                },
                {
                    optionsHash: null,
                    clientClasses: ['foo', 'bar'],
                },
                {
                    optionsHash: null,
                    clientClasses: ['foo', 'bar'],
                },
            ],
        }
        expect(hasDifferentLocalHostData(host)).toBeFalse()
    })

    it('detects no differences for all null client', () => {
        const host: Host = {
            localHosts: [
                {
                    clientClasses: null,
                },
                {
                    clientClasses: null,
                },
                {
                    clientClasses: null,
                },
            ],
        }
        expect(hasDifferentLocalHostData(host)).toBeFalse()
    })

    it('detects differences for some null client classes', () => {
        const host: Host = {
            localHosts: [
                {
                    clientClasses: null,
                },
                {
                    clientClasses: ['foo'],
                },
                {
                    clientClasses: ['foo'],
                },
            ],
        }
        expect(hasDifferentLocalHostData(host)).toBeTrue()
    })

    it('detects no differences when there is a single local host', () => {
        const host: Host = {
            localHosts: [
                {
                    optionsHash: '123',
                    clientClasses: ['foo'],
                },
            ],
        }
        expect(hasDifferentLocalHostData(host)).toBeFalse()
    })

    it('detects no differences when there is no local host', () => {
        const host: Host = {
            localHosts: [],
        }
        expect(hasDifferentLocalHostData(host)).toBeFalse()
    })
})
