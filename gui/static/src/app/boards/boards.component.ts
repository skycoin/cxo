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

    constructor(public route: ActivatedRoute, private skyObject: SkyObjectService,
                private boardsService: BoardsService) {
    }

    ngOnInit() {
        this.boardsService.getBoards().subscribe((data: Board[]) => {
            this.items = data;
        });

    }

    //
    // newBoard(name: string):void{
    //     this.skyObject.create('board', name);
    // }

    sync(id: string): void {
        console.log(this.items[0].id);

        this.skyObject.syncObject(this.items[0].id).subscribe((data: any) => {
            console.log(data);
        });
    }
}
