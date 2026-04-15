// libs/shared/util-parsers/src/lib/util-parsers.ts
import { Temporal } from '@js-temporal/polyfill';
import { ActorMeterRow, ActorTimelineRow, PowerUsageRow, SimLog } from './models';

export function parseActorTimelineCsv(csvRaw: string): ActorTimelineRow[] {
  const lines = csvRaw.trim().split('\n');
  if (lines.length < 2) return [];

  return lines.slice(1).map((line) => {
    const [householdId, timestamp, actorId, actionId, isSharedStr] = line.split(',');
    return {
      householdId: householdId.trim(),
      timestamp: Temporal.Instant.from(timestamp.trim()),
      actorId: actorId.trim(),
      actionId: actionId.trim(),
      isShared: isSharedStr ? isSharedStr.trim().toLowerCase() === 'true' : false,
    };
  }).filter((row) => row.householdId);
}

export function parsePowerUsageCsv(csvRaw: string): PowerUsageRow[] {
  const lines = csvRaw.trim().split('\n');
  if (lines.length < 2) return [];

  return lines.slice(1).map((line) => {
    const [householdId, timestamp, totalWatts, indoorTempC, outdoorTempC, tankTempC, activeDevices] = line.split(',');
    
    const rawDevices = activeDevices ? activeDevices.trim() : '';
    const devices = rawDevices && rawDevices !== '-' ? rawDevices.split('|') : [];

    return {
      householdId: householdId.trim(),
      timestamp: Temporal.Instant.from(timestamp.trim()),
      totalWatts: parseFloat(totalWatts),
      indoorTempC: parseFloat(indoorTempC),
      outdoorTempC: parseFloat(outdoorTempC),
      tankTempC: parseFloat(tankTempC),
      activeDevices: devices,
    };
  }).filter((row) => !isNaN(row.totalWatts));
}

export function parseActorMetersCsv(csvRaw: string): ActorMeterRow[] {
  const lines = csvRaw.trim().split('\n');
  if (lines.length < 2) return [];

  return lines.slice(1).map((line) => {
    const [householdId, timestamp, actorId, energy, hunger, hygiene, leisure] = line.split(',');
    return {
      householdId: householdId.trim(),
      timestamp: Temporal.Instant.from(timestamp.trim()),
      actorId: actorId.trim(),
      energy: parseFloat(energy),
      hunger: parseFloat(hunger),
      hygiene: parseFloat(hygiene),
      leisure: parseFloat(leisure),
    };
  }).filter((row) => row.householdId && !isNaN(row.energy));
}

export function parseSlog(logRaw: string): SimLog[] {
  const lines = logRaw.trim().split('\n');
  const kvRegex = /([a-zA-Z0-9_]+)=(".*?"|\S+)/g;

  return lines.map((line) => {
    const entry: SimLog = {
      cpuTime: Temporal.Instant.from('1970-01-01T00:00:00Z'),
      level: 'INFO',
      msg: '',
      attributes: {},
    };

    let match;
    while ((match = kvRegex.exec(line)) !== null) {
      const key = match[1];
      const rawValue = match[2].replace(/^"(.*)"$/, '$1');
      const numValue = Number(rawValue);
      const value = isNaN(numValue) ? rawValue : numValue;

      switch (key) {
        case 'time':
          entry.cpuTime = Temporal.Instant.from(rawValue);
          break;
        case 'level':
          entry.level = rawValue as string;
          break;
        case 'msg':
          entry.msg = rawValue as string;
          break;
        case 'sim_time':
          entry.simTime = rawValue as string;
          break;
        case 'actor':
          entry.actor = rawValue as string;
          break;
        case 'event':
          entry.event = rawValue as string;
          break;
        default:
          entry.attributes[key] = value;
      }
    }
    return entry;
  });
}