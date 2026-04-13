# @tinywideclouds.com/libs/sim/feature-actor-timeline

## Purpose
This library provides the `ActorTimelineComponent`, an ECharts-based Gantt chart that visualizes the psychological and behavioral transitions of simulation actors.

It automatically reacts to the `SimulationStateService`, filtering its data based on the currently selected household and actors. Because it ingests the transition-only `_actor_timeline.csv` data, rendering is extremely fast.

## Design
The component is styled using Tailwind CSS and relies on `ngx-echarts` with tree-shakable dynamic imports.