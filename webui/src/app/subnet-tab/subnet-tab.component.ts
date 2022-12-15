import { Component, Input, OnInit } from '@angular/core'
import { Subnet } from '../backend'

@Component({
    selector: 'app-subnet-tab',
    templateUrl: './subnet-tab.component.html',
    styleUrls: ['./subnet-tab.component.sass'],
})
export class SubnetTabComponent implements OnInit {
    constructor() {}

    /**
     * Subnet data.
     */
    @Input() subnet: Subnet

    ngOnInit(): void {}
}
