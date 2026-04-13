# dashboard

This library was generated with [Nx](https://nx.dev).

## Running unit tests

Run `nx test dashboard` to execute the unit tests.
# @tinywideclouds.com/libs/sim/features/dashboard

## Purpose
This library contains the primary structural layout of the simulation analyzer. It provides the top-level navigation (Ingestion vs. Analysis) and implements the Main/Detail (Sidebar/Content) pattern for data exploration.

It acts as the orchestrator component, importing the `shared-ui` uploader and the ECharts feature libraries, binding them together using the global `SimulationStateService`.