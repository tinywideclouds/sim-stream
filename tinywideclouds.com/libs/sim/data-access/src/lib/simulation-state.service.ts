// libs/sim/data-access/src/lib/simulation-state.service.ts
import { Injectable, signal, computed } from '@angular/core';
import { Temporal } from '@js-temporal/polyfill';
import { 
  ActorTimelineRow, 
  PowerUsageRow, 
  ActorMeterRow,
  SimLog, 
  parseActorTimelineCsv, 
  parsePowerUsageCsv, 
  parseActorMetersCsv,
  parseSlog
} from '@tinywideclouds.com/libs/sim/util-parsers';

export interface AggregatedPowerBucket {
  timestamp: Temporal.Instant;
  minWatts: number;
  maxWatts: number;
  avgWatts: number;
  indoorTempC: number;
  outdoorTempC: number;
  tankTempC: number;
}

@Injectable({ providedIn: 'root' })
export class SimulationStateService {
  // --- 1. Raw Data State ---
  readonly actorTimeline = signal<ActorTimelineRow[]>([]);
  readonly powerUsage = signal<PowerUsageRow[]>([]);
  readonly meterData = signal<ActorMeterRow[]>([]);
  readonly logs = signal<SimLog[]>([]);

  // --- 2. UI Selection State ---
  readonly selectedHousehold = signal<string | null>(null);
  readonly selectedActors = signal<string[]>([]);
  readonly aggregationIntervalMs = signal<number>(900000); 

  // --- 3. Derived/Computed State ---
  
  readonly availableHouseholds = computed(() => {
    const households = new Set<string>();
    for (const row of this.actorTimeline()) households.add(row.householdId);
    for (const row of this.powerUsage()) households.add(row.householdId);
    for (const row of this.meterData()) households.add(row.householdId);
    return Array.from(households).sort();
  });

  readonly availableActors = computed(() => {
    const currentHouse = this.selectedHousehold();
    if (!currentHouse) return [];

    const actors = new Set<string>();
    for (const row of this.actorTimeline()) {
      if (row.householdId === currentHouse) actors.add(row.actorId);
    }
    for (const row of this.meterData()) {
      if (row.householdId === currentHouse) actors.add(row.actorId);
    }
    return Array.from(actors).sort();
  });

  readonly simTimeRange = computed(() => {
    const power = this.powerUsage();
    if (power.length > 0) {
      return {
        start: power[0].timestamp,
        end: power[power.length - 1].timestamp
      };
    }
    return null;
  });

  readonly activePowerData = computed(() => {
    const household = this.selectedHousehold();
    const rawData = this.powerUsage();
    if (!household || rawData.length === 0) return [];

    const filtered = rawData.filter(r => r.householdId === household);
    const interval = this.aggregationIntervalMs();
    
    if (interval === 0) {
      return filtered.map(r => ({
        timestamp: r.timestamp,
        minWatts: r.totalWatts, maxWatts: r.totalWatts, avgWatts: r.totalWatts,
        indoorTempC: r.indoorTempC, outdoorTempC: r.outdoorTempC, tankTempC: r.tankTempC
      }));
    }

    const buckets = new Map<number, PowerUsageRow[]>();
    for (const row of filtered) {
      const bucketMs = Math.floor(row.timestamp.epochMilliseconds / interval) * interval;
      if (!buckets.has(bucketMs)) buckets.set(bucketMs, []);
      buckets.get(bucketMs)!.push(row);
    }

    const aggregated: AggregatedPowerBucket[] = [];
    const sortedKeys = Array.from(buckets.keys()).sort((a, b) => a - b);

    for (const key of sortedKeys) {
      const rows = buckets.get(key)!;
      let min = Infinity, max = -Infinity, sum = 0, indoorTempSum = 0,  outdoorTempSum = 0, tankSum = 0;

      for (const r of rows) {
        if (r.totalWatts < min) min = r.totalWatts;
        if (r.totalWatts > max) max = r.totalWatts;
        sum += r.totalWatts;
        indoorTempSum += r.indoorTempC;
        outdoorTempSum += r.outdoorTempC;
        tankSum += r.tankTempC;
      }

      aggregated.push({
        timestamp: Temporal.Instant.fromEpochMilliseconds(key),
        minWatts: min, maxWatts: max, 
        avgWatts: sum / rows.length,
        indoorTempC: indoorTempSum / rows.length, 
        outdoorTempC: outdoorTempSum / rows.length, 
        tankTempC: tankSum / rows.length,
      });
    }

    return aggregated;
  });

  readonly activeMeterData = computed(() => {
    const household = this.selectedHousehold();
    const actors = this.selectedActors();
    const rawData = this.meterData();

    if (!household || actors.length === 0 || rawData.length === 0) return [];

    // Filter to only the selected household and currently checked actors in the UI
    return rawData.filter(r => r.householdId === household && actors.includes(r.actorId));
  });

  // --- State Mutations ---

  loadActorTimelineCsv(csvRaw: string): void {
    this.actorTimeline.set(parseActorTimelineCsv(csvRaw));
    this.autoSelectDefaults();
  }

  loadPowerUsageCsv(csvRaw: string): void {
    this.powerUsage.set(parsePowerUsageCsv(csvRaw));
    this.autoSelectDefaults();
  }

  loadActorMetersCsv(csvRaw: string): void {
    this.meterData.set(parseActorMetersCsv(csvRaw));
    this.autoSelectDefaults();
  }

  loadSlogs(logRaw: string): void {
    this.logs.set(parseSlog(logRaw));
  }

  clearData(): void {
    this.actorTimeline.set([]);
    this.powerUsage.set([]);
    this.meterData.set([]);
    this.logs.set([]);
    this.selectedHousehold.set(null);
    this.selectedActors.set([]);
  }

  private autoSelectDefaults(): void {
    const houses = this.availableHouseholds();
    if (houses.length > 0 && !this.selectedHousehold()) {
      this.selectedHousehold.set(houses[0]);
      this.selectedActors.set(this.availableActors());
    }
  }
}