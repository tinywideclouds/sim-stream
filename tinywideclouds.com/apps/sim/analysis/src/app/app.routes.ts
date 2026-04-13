import { Route } from '@angular/router';
import { DashboardComponent } from '@tinywideclouds.com/libs/sim/features/dashboard';

export const appRoutes: Route[] = [
  { path: '', component: DashboardComponent },
  { path: '**', redirectTo: '' }
];