import { Component } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { BoardsService, Board } from '../shared/services/boards.service';
import { SkyObjectService } from '../shared/services/skyobject.service';

@Component({
    selector: 'boards',
    templateUrl: 'boards.component.html'
})
export class BoardsComponent {
    // boards: Board = {};
    items: any[];
    // boardName: string;

    constructor(public route: ActivatedRoute) {
    }

    // ngOnInit() {
        // this.boardsService.getBoards().subscribe((data: Board[]) => {
        //     console.log(data)
        //     this.items = data;
        // });

    // }
    //
    // newBoard(name: string):void{
    //     this.skyObject.create('board', name);
    // }

    sync(id: string): void {
        // this.skyObject.syncObject(id).subscribe((data: any) => {
        //
        // });
    }
}
