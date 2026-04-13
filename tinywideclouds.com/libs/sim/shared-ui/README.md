# @tinywideclouds.com/libs/sim/shared-ui

## Purpose
This library houses the framework-agnostic, purely presentational Angular components for the simulation analyzer. It provides the building blocks for the application layout, ensuring the main routing application remains as thin as possible.

## Components
* `FileUploaderComponent`: A discrete, multi-zone drag-and-drop interface styled with Tailwind. It parses incoming `.csv` and `.log` files, categorizes them into Actor, Power, and Log buckets, and coordinates loading them into the central `SimulationStateService`.

## Design System
This library is styled exclusively via **Tailwind CSS**. Ensure the consuming application includes this library's paths in its `tailwind.config.js` `content` array so the classes are generated correctly.