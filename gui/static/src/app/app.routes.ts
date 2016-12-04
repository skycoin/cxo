import { Routes } from '@angular/router';
import { DashboardComponent } from './dashboard';
import { NoContentComponent } from './no-content';
import {CollectionComponent} from "./collection";
import {SchemaComponent} from "./schema";

export const ROUTES: Routes = [
  { path: '',      component: DashboardComponent },
  { path: 'collection',  component: CollectionComponent },
  { path: 'dashboard',  component: DashboardComponent },
  { path: 'schema/:name',  component: SchemaComponent },
  {
    path: 'detail', loadChildren: () => System.import('./+detail')
      .then((comp: any) => comp.default),
  },
  { path: '**',    component: NoContentComponent }
];
