/*
 * Angular 2 decorators and services
 */
import { Component, ViewEncapsulation } from '@angular/core';

import { AppState } from './app.service';
import { SkyWireService } from './shared/services/skywire.service';
import { SkyObjectService } from './shared/services/skyobject.service';

/*
 * App Component
 * Top Level Component
 */
@Component({
    selector: 'app',
    encapsulation: ViewEncapsulation.None,
    styleUrls: [
        './app.component.css'
    ],
    template: `
    <div class="container-fluid">
        <div class="row">
            <div class="col-sm-12">
                <a [routerLink]=" ['./'] ">
                <h2>Skyhash - Objects</h2>
                </a>
            </div>
        </div>
         <nav>
          <span>
            <a [routerLink]=" ['./dashboard'] ">
              Dashboard
            </a>
          </span>
          |
          <span>
            <a [routerLink]=" ['./boards'] ">
              Boards
            </a>
          </span>
          |
          <span>
            <a [routerLink]=" ['./subscription'] ">
              Subscriptions
            </a>
          </span>
          |
          <span>
            <a [routerLink]=" ['./collection'] ">
              Collection
            </a>
          </span>
        </nav>
    </div>
    <main>
      <router-outlet></router-outlet>
    </main>
    <!--<pre class="app-state">this.appState.state = {{ appState.state | json }}</pre>-->
    <footer>
    </footer>
  `
})
export class AppComponent {

    constructor(public appState: AppState, public skyObjects: SkyObjectService) {

    }

    ngOnInit() {
        console.log('Initial App State', this.appState.state);
    }

}
