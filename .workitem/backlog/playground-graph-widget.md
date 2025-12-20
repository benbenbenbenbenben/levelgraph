---
created: 2025-12-20T10:38:49.918Z
---

# Add visual graph widget to playground

Add a visual graph rendering widget to the playground that displays:

- **Graph visualization**: Render triples as an interactive node-link diagram
  - Nodes for subjects/objects
  - Labeled edges for predicates
  - Support panning, zooming, and node dragging

- **Query result highlighting**: When a query runs, highlight matching nodes/edges
  - Show which parts of the graph matched the pattern
  - Display variable bindings visually

- **Implementation options**:
  - D3.js force-directed graph
  - Cytoscape.js
  - vis.js Network

This would make the playground more educational and help users understand how their queries traverse the graph.