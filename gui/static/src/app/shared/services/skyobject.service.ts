import { Injectable } from '@angular/core';
import { Constants } from '../constants';
import { Headers, Http } from '@angular/http';

declare var _: any;

export class Statistic {
    total: number;
    memory: number;
}

export class Schema {
    name: string;
    fields: SchemaField[];
}

export class SchemaField {
    name: string;
}

@Injectable()
export class SkyObjectService {
    api: string = Constants.API_PATH;
    headers: Headers = new Headers();

    constructor(private http: Http) {
        this.headers.append('Content-Type', 'application/x-www-form-urlencoded');
    }

    getSchemaList() {
        let self = this;
        return this.http.get(this.api + '_schema', {headers: self.headers})
            .map(res => res.json())
            .map((items: Array<Schema>) => {
                return items;
            });
    }

    getSchema(schema: string) {
        let self = this;
        return this.http.get(this.api + schema + '/schema', {headers: self.headers})
            .map(res => res.json());
    }

    getStatistic() {
        let self = this;
        return this.http.get(this.api + '_stat', {headers: self.headers})
            .map(res => res.json());
    }

    getObjectList(schema: string) {
        let self = this;
        return this.http.get(this.api + schema + '/list', {headers: self.headers})
            .map(res => res.json());
    }
}
