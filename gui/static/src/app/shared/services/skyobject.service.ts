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
    type: string;
    tag: string;
}

@Injectable()
export class SkyObjectService {
    api: string = Constants.API_PATH + 'object1/';
    headers: Headers = new Headers();

    constructor(private http: Http) {
        this.headers.append('Content-Type', 'application/x-www-form-urlencoded');
    }

    getStatistic() {
        let self = this;
        return this.http.get(this.api + '_stat', {headers: self.headers})
            .map(res => res.json());
    }

    getSchemaList() {
        let self = this;
        return this.http.get(this.api + '_schemas', {headers: self.headers})
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


    getObjectList(schema: string) {
        let self = this;
        return this.http.get(this.api + schema + '/list', {headers: self.headers})
            .map(res => res.json());
    }

    getObject(schema: string, id: string) {
        let self = this;
        return this.http.get(this.api + schema + '/list/' + id, {headers: self.headers})
            .map(res => res.json());
    }

    syncObject(id: string) {
        let self = this;
        return this.http.post(this.api + '/sync/' + id, {headers: self.headers})
    }

    objectInfo(id: string) {
        let self = this;
        return this.http.get(this.api + '/sync/' + id, {headers: self.headers})
            .map(res => res.json());
    }

    create(schema: string, name: string){
        let self = this;
        return this.http.post(this.api + '/create/' + schema + '/' + name, {headers: self.headers})
            .map(res => res.json());
    }
}
