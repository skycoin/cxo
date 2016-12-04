import {Component} from '@angular/core';
import {ActivatedRoute} from '@angular/router';
import {SkyObjectService, Statistic} from '../shared/services/skyobject.service';


@Component({
    selector: 'dashboard',
    styles: [`
  `],
    templateUrl: 'dashboard.component.html'
})
export class DashboardComponent {
    stat: Statistic = {
        total: 0,
        memory: 0
    };

    constructor(public route: ActivatedRoute, private skyObject: SkyObjectService) {
    }

    ngOnInit() {
        this.skyObject.getStatistic().subscribe((data: Statistic) => {
            this.stat = data;
        });

    }

    formatSizeUnits(bytes) {
        if (bytes >= 1073741824) {
            bytes = (bytes / 1073741824).toFixed(2) + ' GB';
        }
        else if (bytes >= 1048576) {
            bytes = (bytes / 1048576).toFixed(2) + ' MB';
        }
        else if (bytes >= 1024) {
            bytes = (bytes / 1024).toFixed(2) + ' KB';
        }
        else if (bytes > 1) {
            bytes = bytes + ' bytes';
        }
        else if (bytes === 1) {
            bytes = bytes + ' byte';
        }
        else {
            bytes = '0 byte';
        }
        return bytes;
    }
}
