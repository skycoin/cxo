import { Injectable } from '@angular/core';

@Injectable()
export class Constants {
    public static get API_PATH() {
        if ('development' === ENV) {
            return 'http://localhost:6481/';
        } else {
            return '/';
        }
    };
}
