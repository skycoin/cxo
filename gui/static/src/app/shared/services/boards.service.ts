import { Injectable } from '@angular/core';
import { SkyObjectService } from './skyobject.service';
import { Observable } from 'rxjs';

export class Board {
}

@Injectable()
export class BoardsService {
    constructor(private skyobjects: SkyObjectService) {
    }

    getBoards(): Observable<Board[]> {
        let self = this;
        return this.skyobjects.getObjectList('board');
    }

    // createBoard(name: string) {
        // return this.skyobjects.create('board');
    // }
}
