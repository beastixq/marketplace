# README and Diagram Artifact Drift

Status: fixed
Fixed: 2026-04-26
Commits: [`dc48710`](https://github.com/beastixq/marketplace/commit/dc48710), [`ce667cb`](https://github.com/beastixq/marketplace/commit/ce667cb)
Area: docs / diagrams

## Bug

Documentation referenced diagram artifacts that were missing or had stale
notation. The repo had a missing C4 L4 service diagram artifact, and README
still mentioned deprecated interface notation.

## Cause

Generated diagram outputs and README text drifted from the current diagram
source/state.

## Fix

- Added missing `diagrams/old/C4_L4_Service.png`.
- README C4 L4 wording was updated after deprecated interfaces were
  removed.

## Verification

- Fix commits changed only documentation/diagram artifacts.
