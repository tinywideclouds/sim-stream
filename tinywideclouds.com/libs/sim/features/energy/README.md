# @tinywideclouds.com/energy

## Purpose
This library provides the `PhysicsChartComponent`, an ECharts-based dual-axis line chart that visualizes the electrical and thermal thermodynamics of a household.

It automatically reacts to the `SimulationStateService`, rendering either raw tick data or aggregated bucket data (Min/Max/Avg) based on the user's resolution selection in the UI.

## Features
* **Dual Axis Mapping**: Maps Total Watts to the left axis, and Thermal data (Indoor Temp, Tank Temp) to the right axis.
* **Temporal Integration**: Maps `Temporal.Instant` timestamps into ECharts epoch coordinates seamlessly.
* **Tailwind Integration**: Completely styled using Tailwind CSS classes.