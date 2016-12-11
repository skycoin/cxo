import { Injectable } from '@angular/core';
import { Constants } from '../constants';
import { Headers, Http } from '@angular/http';

export class Subscription {
    ip: string;
    port: string;
    pubKey: string;

}

@Injectable()
export class SkyWireService {
    api: string = Constants.API_PATH + 'manager/';
    headers: Headers = new Headers();

    constructor(private http: Http) {
        this.headers.append('Content-Type', 'application/x-www-form-urlencoded');
    }

    getNodes() {
        let self = this;
        return this.http.get(this.api + 'nodes/', {headers: self.headers})
            .map(res => res.json());
    }

    getSubscriptions(id: string) {
        let self = this;
        return this.http.get(this.api + 'nodes/' + id + '/subscriptions', {headers: self.headers})
            .map(res => res.json());
    }

    getSubscribers(id: string) {
        let self = this;
        return this.http.get(this.api + 'nodes/' + id + '/subscribers', {headers: self.headers})
            .map(res => res.json());
    }
}
