import { Component } from '@angular/core';
import { SkyObjectService, Schema } from '../shared/services/skyobject.service';

@Component({
    selector: 'collection',
    templateUrl: 'collection.component.html'
})
export class CollectionComponent {
    schemaList: Schema[];

    constructor(private skyObject: SkyObjectService) {
    }

    ngOnInit() {
        this.skyObject.getSchemaList().subscribe((schema: any) => {
            this.schemaList = schema;
        });
    }
}
