import { DatePipe, KeyValuePipe, PercentPipe } from '@angular/common'
import { ComponentFixture, TestBed } from '@angular/core/testing'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { FieldsetModule } from 'primeng/fieldset'
import { CapitalizeFirstPipe } from '../pipes/capitalize-first.pipe'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { ReplaceAllPipe } from '../pipes/replace-all.pipe'

import { SubnetTabComponent } from './subnet-tab.component'
import { HumanCountComponent } from '../human-count/human-count.component'
import { NumberPipe } from '../pipes/number.pipe'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'

describe('SubnetTabComponent', () => {
    let component: SubnetTabComponent
    let fixture: ComponentFixture<SubnetTabComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
                SubnetTabComponent,
                EntityLinkComponent,
                ReplaceAllPipe,
                CapitalizeFirstPipe,
                HumanCountComponent,
                NumberPipe,
                SubnetBarComponent,
            ],
            imports: [FieldsetModule, NoopAnimationsModule, PercentPipe, KeyValuePipe, DatePipe],
        }).compileComponents()

        fixture = TestBed.createComponent(SubnetTabComponent)
        component = fixture.componentInstance
        component.subnet = {
            id: 42,
            subnet: '42.42.42.42',
        }
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should extract unique statistic keys', () => {
        component.subnet.localSubnets = [
            {
                stats: {
                    foo: 1,
                    bar: 2,
                },
            },
            {
                stats: {
                    foo: 3,
                    ada: 4,
                },
            },
            {
                stats: {
                    zoo: 5,
                },
            },
        ]

        expect(component.localSubnetStatisticKeys).toEqual(['ada', 'bar', 'foo', 'zoo'])
    })

    it('should return an empty list of the static keys if local subnets are missing', () => {
        expect(component.localSubnetStatisticKeys).toEqual([])
    })

    it('should detect IPv6 subnet', () => {
        expect(component.isIPv6).toBeFalse()
        component.subnet.subnet = 'fe80::/32'
        expect(component.isIPv6).toBeTrue()
    })
})
