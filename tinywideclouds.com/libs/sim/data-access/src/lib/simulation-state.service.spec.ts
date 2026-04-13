// libs/sim/data-access/src/lib/simulation-state.service.spec.ts
import { TestBed } from '@angular/core/testing';
import { SimulationStateService } from './simulation-state.service';

describe('SimulationStateService', () => {
  let service: SimulationStateService;

  const mockActorCsv = `HouseholdID,Timestamp,ActorID,ActionID,IsShared
house_A,2026-01-01T12:00:00Z,actor_1,wfh_session,false
house_A,2026-01-01T12:15:00Z,actor_2,family_dinner,true
house_B,2026-01-01T12:00:00Z,actor_3,cook_dinner,false`;

  const mockPowerCsv = `HouseholdID,Timestamp,TotalWatts,IndoorTempC,TankTempC,ActiveDevices
house_A,2026-01-01T12:00:00Z,150.0,21.0,55.0,tv_1
house_A,2026-01-01T12:15:00Z,2000.0,21.0,55.0,kettle_1`;

  const mockMeterCsv = `HouseholdID,Timestamp,ActorID,Energy,Hunger,Hygiene,Leisure
house_A,2026-01-01T12:00:00Z,actor_1,80,50,90,40
house_A,2026-01-01T12:15:00Z,actor_2,70,20,80,60
house_B,2026-01-01T12:00:00Z,actor_3,100,100,100,100`;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(SimulationStateService);
  });

  it('should be created with empty initial state', () => {
    expect(service.actorTimeline().length).toBe(0);
    expect(service.meterData().length).toBe(0);
    expect(service.availableHouseholds().length).toBe(0);
    expect(service.selectedHousehold()).toBeNull();
  });

  it('should extract unique households from all datasets', () => {
    service.loadActorMetersCsv(mockMeterCsv); // Testing extraction from the new meter data
    const households = service.availableHouseholds();
    
    expect(households).toEqual(['house_A', 'house_B']);
  });

  it('should auto-select the first household and its actors upon load', () => {
    service.loadActorTimelineCsv(mockActorCsv);
    
    expect(service.selectedHousehold()).toBe('house_A');
    expect(service.selectedActors()).toEqual(['actor_1', 'actor_2']);
  });

  it('should provide activeMeterData filtered by current UI selections', () => {
    service.loadActorMetersCsv(mockMeterCsv);
    // Auto-selects house_A and [actor_1, actor_2] by default
    
    let activeMeters = service.activeMeterData();
    expect(activeMeters).toHaveLength(2);
    expect(activeMeters[0].actorId).toBe('actor_1');
    expect(activeMeters[1].actorId).toBe('actor_2');

    // Deselect actor_2
    service.selectedActors.set(['actor_1']);
    activeMeters = service.activeMeterData();
    
    expect(activeMeters).toHaveLength(1);
    expect(activeMeters[0].actorId).toBe('actor_1');
  });

  it('should completely reset state on clearData()', () => {
    service.loadActorTimelineCsv(mockActorCsv);
    service.loadPowerUsageCsv(mockPowerCsv);
    service.loadActorMetersCsv(mockMeterCsv);
    
    service.clearData();
    
    expect(service.actorTimeline().length).toBe(0);
    expect(service.powerUsage().length).toBe(0);
    expect(service.meterData().length).toBe(0);
    expect(service.selectedHousehold()).toBeNull();
    expect(service.selectedActors().length).toBe(0);
  });
});