import { Component, computed, inject } from '@angular/core';
import { SimulationStateService } from '@tinywideclouds.com/libs/sim/data-access';
import { NgxEchartsDirective, provideEchartsCore } from 'ngx-echarts';
import type { EChartsOption } from 'echarts';
import * as echarts from 'echarts/core';
import { ActorMeterRow } from '@tinywideclouds.com/libs/sim/util-parsers';

@Component({
  selector: 'sim-actor-meters',
  standalone: true,
  imports: [NgxEchartsDirective],
  providers: [
    provideEchartsCore({ echarts: () => import('echarts') })
  ],
  templateUrl: './actor-meters.component.html',
  styleUrl: './actor-meters.component.scss'
})
export class ActorMetersComponent {
  private state = inject(SimulationStateService);

  onChartInit(chartInstance: any): void {
    chartInstance.group = 'sim-analyzer-group';
    echarts.connect('sim-analyzer-group');
  }

  chartOptions = computed<EChartsOption | null>(() => {
    const meterData = this.state.activeMeterData();
    const actors = this.state.selectedActors();
    const household = this.state.selectedHousehold();
    const timeRange = this.state.simTimeRange();

    if (!household || actors.length === 0 || meterData.length === 0 || !timeRange) {
      return null;
    }

    const series: any[] = [];
    const legendData: string[] = [];

    const meterColors = { Energy: '#eab308', Hunger: '#ef4444', Hygiene: '#06b6d4', Leisure: '#8b5cf6' };
    const lineTypes = ['solid', 'dashed', 'dotted'];

    for (let i = 0; i < actors.length; i++) {
      const actor = actors[i];
      const actorRows = meterData.filter(r => r.actorId === actor);
      if (actorRows.length === 0) continue;

      const lineType = lineTypes[i % lineTypes.length];

      const addSeries = (meterName: 'Energy' | 'Hunger' | 'Hygiene' | 'Leisure', key: keyof ActorMeterRow) => {
        const seriesName = actors.length > 1 ? `${actor} (${meterName})` : meterName;
        legendData.push(seriesName);
        
        series.push({
          name: seriesName,
          type: 'line',
          smooth: true,
          symbol: 'circle',
          symbolSize: 4,
          showSymbol: false, // Only show dots on hover to avoid clutter
          lineStyle: { width: 2, type: lineType as any },
          itemStyle: { color: meterColors[meterName] },
          data: actorRows.map(r => [r.timestamp.epochMilliseconds, r[key]])
        });
      };

      addSeries('Energy', 'energy');
      addSeries('Hunger', 'hunger');
      addSeries('Hygiene', 'hygiene');
      addSeries('Leisure', 'leisure');
    }

    return {
      tooltip: {
        trigger: 'item', // Fixes the "snapping" issue across long empty slopes
        axisPointer: { 
          type: 'cross', // Provides an exact Y-axis tracking line for interpolated values
          label: { backgroundColor: '#475569' }
        },
        formatter: (params: any) => {
          const date = new Date(params.data[0]).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
          return `<b>${date}</b><br/>${params.marker} ${params.seriesName}: ${Number(params.data[1]).toFixed(1)}%`;
        }
      },
      legend: { data: legendData, top: 0, type: 'scroll' },
      // LOCKED GRID: Matches the other charts exactly
      grid: { left: '60px', right: '60px', top: '15%', bottom: '15%' },
      xAxis: {
        type: 'time',
        min: timeRange.start.epochMilliseconds, // Locks bounds to prevent drift
        max: timeRange.end.epochMilliseconds,
        axisLabel: { formatter: '{MM}-{dd} {HH}:{mm}' }
      },
      yAxis: {
        type: 'value',
        name: 'Satiety (%)',
        min: 0,
        max: 100,
        axisLine: { show: true, lineStyle: { color: '#94a3b8' } },
        splitLine: { show: true, lineStyle: { type: 'dashed', color: '#f1f5f9' } }
      },
      dataZoom: [
        { type: 'slider', filterMode: 'weakFilter', showDataShadow: false, bottom: 10, height: 20 },
        { type: 'inside', filterMode: 'weakFilter' }
      ],
      series: series
    } as EChartsOption;
  });
}