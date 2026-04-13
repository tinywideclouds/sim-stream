// libs/analyzer/feature-log-explorer/src/lib/log-explorer.component.ts
import { Component, computed, inject } from '@angular/core';
import { SimulationStateService } from '@tinywideclouds.com/libs/sim/data-access';
import { JsonPipe } from '@angular/common';

@Component({
  selector: 'sim-log-explorer',
  standalone: true,
  imports: [JsonPipe],
  templateUrl: './log-explorer.html',
  styleUrl: './log-explorer.css'
})
export class LogExplorerComponent {
  private state = inject(SimulationStateService);

  // Show newest logs first
  reversedLogs = computed(() => {
    return [...this.state.logs()].reverse();
  });
}