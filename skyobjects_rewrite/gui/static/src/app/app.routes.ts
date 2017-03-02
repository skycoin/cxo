import { Routes } from '@angular/router';
import { DashboardComponent } from './dashboard';
import { NoContentComponent } from './no-content';
import { CollectionComponent } from './collection';
import { SchemaComponent } from './schema/schema.component';
import { SchemaDetailsComponent } from './schema/details.component';
import { SubscriptionComponent } from './subscription/subscription.component';
import { BoardsComponent } from './boards/boards.component';

export const ROUTES: Routes = [
    {path: '', component: DashboardComponent},
    {path: 'collection', component: CollectionComponent},
    {path: 'dashboard', component: DashboardComponent},
    {path: 'boards', component: BoardsComponent},
    {path: 'subscription', component: SubscriptionComponent},
    {path: 'schema/:name', component: SchemaComponent},
    {path: 'schema/:name/:id', component: SchemaDetailsComponent},
    {
        path: 'detail', loadChildren: () => System.import('./+detail')
        .then((comp: any) => comp.default),
    },
    {path: '**', component: NoContentComponent}
];
