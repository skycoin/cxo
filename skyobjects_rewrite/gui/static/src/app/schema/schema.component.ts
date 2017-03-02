import { Component } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { SkyObjectService, Schema } from '../shared/services/skyobject.service';

declare var _: any;

@Component({
    selector: 'schema',
    templateUrl: 'schema.component.html'
})
export class SchemaComponent {
    schema: Schema = {name: '', fields: []};
    items: any[];

    constructor(public route: ActivatedRoute, private skyObject: SkyObjectService) {
    }

    ngOnInit() {
        let schemaName = this.route.snapshot.params['name'];
        this.skyObject.getSchema(schemaName).subscribe((data: any) => {
            this.schema = data;
        });

        this.skyObject.getObjectList(schemaName).subscribe((data: any) => {
            this.items = data;
        });
    }

    displayItem(schema: Schema, item: any): string {
        let result: string = '';
        for (let i = 0; i < schema.fields.length; i++) {
            if (schema.fields[i].tag !== '') {
                result += schema.fields[i].tag + ' ';
            } else {
                result += item[schema.fields[i].name] + ';  ';
            }
        }
        return result;
    }

}
