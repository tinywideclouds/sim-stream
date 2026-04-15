import { Component, computed, inject } from '@angular/core';
import { SimulationStateService } from '@tinywideclouds.com/libs/sim/data-access';
import { NgxEchartsDirective, provideEchartsCore } from 'ngx-echarts';
import type { EChartsOption } from 'echarts';
import * as echarts from 'echarts/core';

@Component({
  selector: 'sim-physics-chart',
  standalone: true,
  imports: [NgxEchartsDirective],
  providers: [
    provideEchartsCore({ echarts: () => import('echarts') })
  ],
  templateUrl: './physics-chart.component.html',
  styleUrl: './physics-chart.component.scss'
})
export class PhysicsChartComponent {
  private state = inject(SimulationStateService);

  onChartInit(chartInstance: any): void {
    chartInstance.group = 'sim-analyzer-group';
    echarts.connect('sim-analyzer-group');
  }

  chartOptions = computed<EChartsOption | null>(() => {
    const powerData = this.state.activePowerData();
    const household = this.state.selectedHousehold();
    const timeRange = this.state.simTimeRange();

    if (!household || powerData.length === 0 || !timeRange) return null;

    const timeData: number[] = [];
    const avgWattsData: number[] = [];
    const indoorTempData: number[] = [];
    const outdoorTempData: number[] = [];
    const tankTempData: number[] = [];

    for (const row of powerData) {
      timeData.push(row.timestamp.epochMilliseconds);
      avgWattsData.push(row.avgWatts);
      indoorTempData.push(row.indoorTempC);
      outdoorTempData.push(row.outdoorTempC);
      tankTempData.push(row.tankTempC);
    }

    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'cross' },
        formatter: (params: any) => {
          const date = new Date(params[0].axisValue).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
          let res = `<b>${date}</b><br/>`;
          for (const p of params) {
            const unit = p.seriesName.includes('Temp') ? '°C' : 'W';
            res += `${p.marker} ${p.seriesName}: ${p.data[1].toFixed(1)} ${unit}<br/>`;
          }
          return res;
        }
      },
      legend: { data: ['Avg Power', 'Indoor Temp', 'Outdoor Temp', 'Tank Temp'], top: 0 },
      // LOCKED GRID: Matches the other charts exactly
      grid: { left: '60px', right: '60px', top: '15%', bottom: '15%' },
      xAxis: {
        type: 'time',
        min: timeRange.start.epochMilliseconds, // Locks bounds to prevent drift
        max: timeRange.end.epochMilliseconds,
        axisLabel: { formatter: '{MM}-{dd} {HH}:{mm}' }
      },
      yAxis: [
        { type: 'value', name: 'Power (W)', position: 'left', axisLine: { show: true, lineStyle: { color: '#e6a23c' } }, splitLine: { show: false } },
        { type: 'value', name: 'Temp (°C)', position: 'right', axisLine: { show: true, lineStyle: { color: '#f56c6c' } }, splitLine: { show: true, lineStyle: { type: 'dashed', color: '#eee' } }, min: 'dataMin', max: 'dataMax' }
      ],
      dataZoom: [
        { type: 'slider', filterMode: 'weakFilter', showDataShadow: false, bottom: 10, height: 20 },
        { type: 'inside', filterMode: 'weakFilter' }
      ],
      series: [
        { name: 'Avg Power', type: 'line', step: 'end', yAxisIndex: 0, areaStyle: { opacity: 0.2 }, lineStyle: { color: '#e6a23c', width: 2 }, itemStyle: { color: '#e6a23c' }, data: timeData.map((time, idx) => [time, avgWattsData[idx]]) },
        { name: 'Indoor Temp', type: 'line', smooth: true, yAxisIndex: 1, lineStyle: { color: '#67c23a', width: 2 }, itemStyle: { color: '#67c23a' }, data: timeData.map((time, idx) => [time, indoorTempData[idx]]) },
        { name: 'Outdoor Temp', type: 'line', smooth: true, yAxisIndex: 1, lineStyle: { color: '#398ba4', width: 2 }, itemStyle: { color: '#398ba4' }, data: timeData.map((time, idx) => [time, outdoorTempData[idx]]) },
        { name: 'Tank Temp', type: 'line', smooth: true, yAxisIndex: 1, lineStyle: { color: '#f56c6c', width: 2 }, itemStyle: { color: '#f56c6c' }, data: timeData.map((time, idx) => [time, tankTempData[idx]]) }
      ]
    } as EChartsOption;
  });
}