import { Component } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { SkyObjectService, Schema, SchemaField } from '../shared/services/skyobject.service';


@Component({
    selector: 'SchemaDetails',
    templateUrl: 'details.component.html'
})
export class SchemaDetailsComponent {
    schema: Schema = {name: '', fields: []};
    item: any;

    constructor(public route: ActivatedRoute, private skyObject: SkyObjectService) {
    }

    ngOnInit() {
        let id = this.route.snapshot.params['id'];
        let name = this.route.snapshot.params['name'];

        this.skyObject.getSchema(name).subscribe((data: any) => {
            this.schema = data;

            this.skyObject.getObject(name, id).subscribe((item: any) => {
                this.item = item;
            });
        });


    }

    isLink(field: SchemaField): boolean {
        if (field && field.tag) {
            return true;
        }
        return false;
    }

    getLinkId(field: SchemaField, item: any): string {
        return '';
    }

    displayField(field: SchemaField, item: any): string {
        let result: string = '';
        if (field && item) {
            result += field.name + ':' + item[field.name];
        }
        return result;
    }

}
