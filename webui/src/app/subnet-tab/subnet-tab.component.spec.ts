import { DatePipe, KeyValuePipe, PercentPipe, TitleCasePipe, UpperCasePipe } from '@angular/common'
import { ComponentFixture, TestBed } from '@angular/core/testing'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { FieldsetModule } from 'primeng/fieldset'
import { CapitalizeFirstPipe } from '../capitalize-first.pipe'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { ReplaceAllPipe } from '../replace-all.pipe'

import { SubnetTabComponent } from './subnet-tab.component'
import { HumanCountComponent } from '../human-count/human-count.component'

describe('SubnetTabComponent', () => {
    let component: SubnetTabComponent
    let fixture: ComponentFixture<SubnetTabComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [SubnetTabComponent, EntityLinkComponent, ReplaceAllPipe, CapitalizeFirstPipe],
            imports: [FieldsetModule, NoopAnimationsModule, PercentPipe, KeyValuePipe, DatePipe, HumanCountComponent],
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
