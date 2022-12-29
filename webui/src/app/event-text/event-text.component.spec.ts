import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { RouterTestingModule } from '@angular/router/testing'
import { EntityLinkComponent } from '../entity-link/entity-link.component'

import { EventTextComponent } from './event-text.component'

describe('EventTextComponent', () => {
    let component: EventTextComponent
    let fixture: ComponentFixture<EventTextComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [RouterTestingModule],
            declarations: [EventTextComponent, EntityLinkComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(EventTextComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
