import { Component } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { SkyWireService } from '../shared/services/skywire.service';

declare var _: any;

@Component({
    selector: 'subscription',
    templateUrl: 'subscription.component.html'
})
export class SubscriptionComponent {
    // schema: Schema = {name: '', fields: []};
    items: any[];

    item: any;

    constructor(public route: ActivatedRoute, private skyWire: SkyWireService) {
    }

    ngOnInit() {
        // let schemaName = this.route.snapshot.params['name'];
        this.skyWire.getNodes().subscribe((data: any) => {
            console.log(data);
            this.items = data;

            this.skyWire.getSubscriptions(this.items[0].pubKey).subscribe((datas: any) => {
                console.log('Subscriptions', datas);
            });

            this.skyWire.getSubscribers(this.items[0].pubKey).subscribe((datas: any) => {
                console.log('Subscribers', datas);
            });
        });
        //
        // this.skyObject.getObjectList(schemaName).subscribe((data: any) => {
        //     this.items = data;
        // });
    }
}
