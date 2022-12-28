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
})
