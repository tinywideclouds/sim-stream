// libs/shared/util-parsers/src/lib/util-parsers.spec.ts
import { describe, it, expect } from 'vitest';
import { parseActorTimelineCsv, parsePowerUsageCsv, parseActorMetersCsv, parseSlog } from './util-parsers';

describe('Simulation Parsers', () => {
  describe('parseActorTimelineCsv', () => {
    it('should correctly parse an actor timeline row with isShared flag', () => {
      const mockCsv = `HouseholdID,Timestamp,ActorID,ActionID,IsShared\nfamily_061,2026-01-19T22:33:45Z,wfh_workaholic_1,family_dinner,true`;

      const result = parseActorTimelineCsv(mockCsv);

      expect(result).toHaveLength(1);
      expect(result[0].householdId).toBe('family_061');
      expect(result[0].actorId).toBe('wfh_workaholic_1');
      expect(result[0].actionId).toBe('family_dinner');
      expect(result[0].isShared).toBe(true);
      expect(result[0].timestamp.toString()).toBe('2026-01-19T22:33:45Z');
    });
  });

  describe('parsePowerUsageCsv', () => {
    it('should correctly parse a power usage row with active devices', () => {
      const mockCsv = `HouseholdID,Timestamp,TotalWatts,IndoorTempC,TankTempC,ActiveDevices\nfamily_061,2026-01-19T22:33:45Z,2450.50,21.5,55.0,kettle_3|tv_9`;

      const result = parsePowerUsageCsv(mockCsv);

      expect(result).toHaveLength(1);
      expect(result[0].householdId).toBe('family_061');
      expect(result[0].totalWatts).toBe(2450.5);
      expect(result[0].indoorTempC).toBe(21.5);
      expect(result[0].tankTempC).toBe(55.0);
      expect(result[0].activeDevices).toEqual(['kettle_3', 'tv_9']);
    });
  });

  describe('parseActorMetersCsv', () => {
    it('should correctly parse a meter usage row', () => {
      const mockCsv = `HouseholdID,Timestamp,ActorID,Energy,Hunger,Hygiene,Leisure\nfamily_061,2026-01-19T22:33:45Z,actor_1,85.5,20.0,95.0,10.0`;

      const result = parseActorMetersCsv(mockCsv);

      expect(result).toHaveLength(1);
      expect(result[0].householdId).toBe('family_061');
      expect(result[0].actorId).toBe('actor_1');
      expect(result[0].energy).toBe(85.5);
      expect(result[0].hunger).toBe(20.0);
      expect(result[0].hygiene).toBe(95.0);
      expect(result[0].leisure).toBe(10.0);
    });
  });

  describe('parseSlog', () => {
    it('should dynamically parse structured logs and infer types', () => {
      const mockLog = `time=2026-04-10T12:16:19.546+01:00 level=INFO msg="WFH SLACKING VETOED" sim_time="Mon 23:03" actor=wfh_workaholic_1 attempted=play_board_game resumed=wfh_session inertia=14.5`;

      const result = parseSlog(mockLog);

      expect(result).toHaveLength(1);
      expect(result[0].msg).toBe('WFH SLACKING VETOED');
      expect(result[0].simTime).toBe('Mon 23:03');
      expect(result[0].actor).toBe('wfh_workaholic_1');
      expect(result[0].attributes['attempted']).toBe('play_board_game');
      expect(result[0].attributes['resumed']).toBe('wfh_session');
      expect(result[0].attributes['inertia']).toBe(14.5);
    });
  });
});