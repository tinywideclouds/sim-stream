import { Component, computed, inject } from '@angular/core';
import { SimulationStateService } from '@tinywideclouds.com/libs/sim/data-access';
import { NgxEchartsDirective, provideEchartsCore } from 'ngx-echarts';
import type { EChartsOption } from 'echarts';
import * as echarts from 'echarts/core';

interface TimelineSpan {
  actorIndex: number;
  action: string;
  startTime: number;
  endTime: number;
  isShared: boolean;
}

function getActionColor(action: string): string {
  const lowerAction = action.toLowerCase();
  if (lowerAction.startsWith('cook_') || lowerAction.includes('meal') || lowerAction.includes('eat') || lowerAction.includes('dinner')) return '#f97316';
  if (lowerAction.startsWith('wfh_') || lowerAction.includes('work')) return '#3b82f6';
  if (lowerAction.startsWith('sleep_') || lowerAction.includes('night')) return '#8b5cf6';
  if (lowerAction.startsWith('hygiene_') || lowerAction.includes('shower') || lowerAction.includes('wash')) return '#0ea5e9';
  if (lowerAction.startsWith('leisure_') || lowerAction.includes('play') || lowerAction.includes('tv') || lowerAction.includes('game')) return '#10b981';
  if (lowerAction === 'idle/away' || lowerAction.includes('idle')) return '#cbd5e1';
  if (lowerAction.startsWith('away_')) return '#64748b';

  const hash = action.split('').reduce((acc: number, char: string) => char.charCodeAt(0) + ((acc << 5) - acc), 0);
  const hue = Math.abs(hash) % 360;
  return `hsl(${hue}, 65%, 55%)`;
}

@Component({
  selector: 'sim-actor-timeline',
  standalone: true,
  imports: [NgxEchartsDirective],
  providers: [
    provideEchartsCore({ echarts: () => import('echarts') })
  ],
  templateUrl: './actor-timeline.html',
  styleUrl: './actor-timeline.scss'
})
export class ActorTimelineComponent {
  private state = inject(SimulationStateService);

  onChartInit(chartInstance: any): void {
    chartInstance.group = 'sim-analyzer-group';
    echarts.connect('sim-analyzer-group');
  }

  chartOptions = computed<EChartsOption | null>(() => {
    const rawTimeline = this.state.actorTimeline();
    const household = this.state.selectedHousehold();
    const actors = this.state.selectedActors();
    const timeRange = this.state.simTimeRange();

    if (!household || actors.length === 0 || rawTimeline.length === 0 || !timeRange) {
      return null;
    }

    const filteredRows = rawTimeline.filter(row => row.householdId === household && actors.includes(row.actorId));
    const actorTransitions = new Map<string, { timeMs: number; action: string; isShared: boolean }[]>();
    for (const actor of actors) actorTransitions.set(actor, []);

    for (const row of filteredRows) {
      actorTransitions.get(row.actorId)!.push({ timeMs: row.timestamp.epochMilliseconds, action: row.actionId, isShared: row.isShared });
    }

    const spans: TimelineSpan[] = [];
    const endOfSimMs = timeRange.end.epochMilliseconds;

    for (let i = 0; i < actors.length; i++) {
      const actor = actors[i];
      const transitions = actorTransitions.get(actor)!;
      transitions.sort((a, b) => a.timeMs - b.timeMs);

      for (let j = 0; j < transitions.length; j++) {
        const current = transitions[j];
        const next = transitions[j + 1];
        const startTime = current.timeMs;
        const endTime = next ? next.timeMs : endOfSimMs;

        if (endTime > startTime) {
          spans.push({ actorIndex: i, action: current.action, startTime, endTime, isShared: current.isShared });
        }
      }
    }

    return {
      tooltip: {
        formatter: (params: any) => {
          const data = params.data;
          const start = new Date(data[1]).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
          const end = new Date(data[2]).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
          const sharedTag = data[4] ? `<br/><span style="color:#f59e0b; font-weight:bold;">[ Shared Event ]</span>` : '';
          return `<b>${data[3]}</b>${sharedTag}<br/>${start} - ${end}`;
        }
      },
      // LOCKED GRID: Exactly 60px padding, no label containment needed anymore
      grid: { left: '60px', right: '60px', top: '15%', bottom: '15%' },
      xAxis: {
        type: 'time',
        min: timeRange.start.epochMilliseconds,
        max: timeRange.end.epochMilliseconds,
        axisLabel: { formatter: '{MM}-{dd} {HH}:{mm}' }
      },
      yAxis: {
        type: 'category',
        data: actors,
        inverse: true,
        axisLabel: {
          show: true,
          inside: true, // Floats the label inside the grid, locked to the left edge
          verticalAlign: 'bottom',
          padding: [0, 0, 16, 5], // Pushes the text up so it hovers above the Gantt bar
          fontWeight: 'bold',
          color: '#1e293b',
          fontSize: 13,
          z: 10 // Ensures it renders above the bars when zooming
        },
        axisTick: { show: false },
        axisLine: { show: false }
      },
      dataZoom: [
        { type: 'slider', filterMode: 'weakFilter', showDataShadow: false, bottom: 10, height: 20 },
        { type: 'inside', filterMode: 'weakFilter' }
      ],
      series: [
        {
          type: 'custom',
          renderItem: (params: any, api: any) => {
            const categoryIndex = api.value(0);
            const start = api.coord([api.value(1), categoryIndex]);
            const end = api.coord([api.value(2), categoryIndex]);
            // Slightly thinner bar to ensure it doesn't overlap the floating label above it
            const height = api.size([0, 1])[1] * 0.45; 
            
            const actionName = api.value(3);
            const isShared = api.value(4);
            const color = getActionColor(actionName);

            return {
              type: 'rect',
              shape: { x: start[0], y: start[1] - height / 2, width: end[0] - start[0], height: height, r: 4 },
              style: api.style({
                fill: color,
                stroke: isShared ? '#fbbf24' : '#ffffff',
                lineWidth: isShared ? 3 : 1,
                opacity: 0.95
              })
            };
          },
          encode: { x: [1, 2], y: 0 },
          data: spans.map(s => [s.actorIndex, s.startTime, s.endTime, s.action, s.isShared])
        }
      ]
    } as EChartsOption;
  });
}