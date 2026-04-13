# @sim-workspace/util-parsers

## Purpose
`util-parsers` is a pure, framework-agnostic TypeScript library responsible for translating the raw text output of the Go simulation engine into strongly typed data models. 

It acts as the data-ingestion boundary for the Angular application. By isolating this logic outside of Angular components and services, we ensure the parsing functions remain lightweight, highly testable (via Vitest), and completely decoupled from framework lifecycles.

## Ingestion Formats

This library provides dedicated parsers for three specific engine outputs:

1. **Actor Timeline (`_actor_timeline.csv`)**: Captures psychological/behavioral transitions.
   - *Columns*: `HouseholdID, Timestamp, ActorID, ActionID`
2. **Power Usage (`_power_usage.csv`)**: Captures dense, tick-by-tick physics and grid telemetry.
   - *Columns*: `HouseholdID, Timestamp, TotalWatts, IndoorTempC, TankTempC, ActiveDevices`
3. **Structured Logs (`app.log`)**: Captures narrative application behavior (slogs).
   - *Format*: `key=value` pairs with dynamic attributes.

## Usage

```typescript
import { 
  parseActorTimelineCsv, 
  parsePowerUsageCsv, 
  parseSlog 
} from '@sim-workspace/util-parsers';

const timeline = parseActorTimelineCsv(rawCsvString);
const powerData = parsePowerUsageCsv(rawCsvString);
const logs = parseSlog(rawLogString);
```