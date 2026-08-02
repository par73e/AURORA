---
name: AURORA Orbital Observatory
description: A scrollable spatial observatory for exploring near-Earth activity.
colors:
  space-black: "#03070c"
  deep-navy: "#07131e"
  surface-navy: "#0b1b28"
  frost-white: "#ecf5f9"
  telemetry-muted: "#7f98a7"
  orbital-blue: "#72d7ff"
  launch-amber: "#ffb866"
  structural-line: "rgba(139, 180, 202, 0.18)"
typography:
  display:
    fontFamily: "Manrope, Noto Sans SC, sans-serif"
    fontSize: "clamp(2.5rem, 5vw, 5rem)"
    fontWeight: 500
    lineHeight: 1.02
    letterSpacing: "-0.03em"
  body:
    fontFamily: "Manrope, Noto Sans SC, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.7
  data:
    fontFamily: "IBM Plex Mono, monospace"
    fontSize: "0.75rem"
    fontWeight: 500
    lineHeight: 1.4
rounded:
  control: "4px"
  surface: "12px"
spacing:
  unit: "8px"
  section: "clamp(88px, 10vw, 152px)"
---

# Design System: AURORA Orbital Observatory

## Overview

**Creative North Star: "The Orbital Observatory"**

AURORA should feel like a quiet public observatory connected to live orbital data, not a cramped mission-control dashboard. The three-dimensional Earth leads the first viewport. Dense datasets receive their own generous workspaces farther down the page.

The interface uses cold, nearly black spatial surfaces with one orbital-blue interaction color and launch amber only where launch context is real. Technical typography labels measurements and state; ordinary reading text remains comfortable and uncompressed.

## Colors

Deep navy establishes physical depth. Orbital blue marks selectable spatial objects and focus state. Launch amber is reserved for launch events and facilities.

## Typography

Display and reading copy use the existing sans-serif stack. IBM Plex Mono is restricted to time, identifiers, orbital values and source metadata.

## Layout

All page regions share one centered container and a 12-column grid. The header, first viewport, object catalog, launch schedule and site directory use the same left and right edges. Vertical dividers are avoided; horizontal rules separate real reading groups.

The page scrolls normally. Only the global header may remain sticky. The first viewport prioritizes the Earth; object search, complete event schedules and directories live in dedicated sections below.

## Elevation & Depth

Depth comes from WebGL, tonal layering and restrained backdrop blur. Content sections are flat. Shadows are reserved for a contextual detail panel that appears over the Earth after selection.

## Shapes

Large content surfaces use a 12px radius only when they behave as contained workspaces. Inputs and compact controls use 4px corners. Data rows remain mostly rectangular and are separated by spacing or a single horizontal rule.

## Components

- The contextual inspector shows only the currently selected entity and can be closed.
- The object catalog owns search, regular-expression queries, filters, sorting and future pagination.
- The launch schedule shows every loaded event in the promised period. Selecting an event focuses the globe and replaces the inspector with task and launch-site context.
- Status dots indicate genuine live or data health state only.

## Do's and Don'ts

### Do:

- Do let the Earth own the first viewport.
- Do provide complete lists in sections designed for reading and querying.
- Do align every section to the same container and grid.
- Do reveal detailed information in response to user selection.

### Don't:

- Don't force the entire product into one fixed-height viewport.
- Don't leave unrelated object controls visible after a launch event is selected.
- Don't use decorative vertical rules that do not continue a real grid boundary.
- Don't require users to manually scan hundreds of spacecraft names.
