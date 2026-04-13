import { Component, inject, signal } from '@angular/core';
import { SimulationStateService } from '@tinywideclouds.com/libs/sim/data-access';
import { FileUploaderComponent } from '@tinywideclouds.com/libs/sim/shared-ui';
import { ActorTimelineComponent, ActorMetersComponent } from '@tinywideclouds.com/libs/sim/features/actor-timeline';
import { PhysicsChartComponent } from '@tinywideclouds.com/libs/sim/features/energy';

@Component({
  selector: 'sim-dashboard',
  standalone: true,
  imports: [
    FileUploaderComponent,
    ActorTimelineComponent,
    ActorMetersComponent,
    PhysicsChartComponent
  ],
  templateUrl: './dashboard.html',
  styleUrl: './dashboard.css'
})
export class DashboardComponent {
  state = inject(SimulationStateService);

  activeSection = signal<'ingestion' | 'analysis'>('ingestion');

  onHouseholdChange(event: Event): void {
    const val = (event.target as HTMLSelectElement).value;
    this.state.selectedHousehold.set(val === '' ? null : val);

    if (val !== '') {
      this.state.selectedActors.set(this.state.availableActors());
    } else {
      this.state.selectedActors.set([]);
    }
  }

  toggleActor(actor: string, event: Event): void {
    const isChecked = (event.target as HTMLInputElement).checked;
    const currentSelected = this.state.selectedActors();

    if (isChecked) {
      this.state.selectedActors.set([...currentSelected, actor]);
    } else {
      this.state.selectedActors.set(currentSelected.filter(a => a !== actor));
    }
  }

  onResolutionChange(event: Event): void {
    const val = parseInt((event.target as HTMLSelectElement).value, 10);
    this.state.aggregationIntervalMs.set(val);
  }
}