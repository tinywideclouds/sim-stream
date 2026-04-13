// libs/sim/util-parsers/src/lib/models.ts
import { Temporal } from '@js-temporal/polyfill';

export interface ActorTimelineRow {
  householdId: string;
  timestamp: Temporal.Instant;
  actorId: string;
  actionId: string;
  isShared: boolean;
}

export interface PowerUsageRow {
  householdId: string;
  timestamp: Temporal.Instant;
  totalWatts: number;
  indoorTempC: number;
  tankTempC: number;
  activeDevices: string[];
}

export interface ActorMeterRow {
  householdId: string;
  timestamp: Temporal.Instant;
  actorId: string;
  energy: number;
  hunger: number;
  hygiene: number;
  leisure: number;
}

export interface SimLog {
  cpuTime: Temporal.Instant;
  level: string;
  msg: string;
  simTime?: string;
  actor?: string;
  event?: string;
  attributes: Record<string, string | number>;
}