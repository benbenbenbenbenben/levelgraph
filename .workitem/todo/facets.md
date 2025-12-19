---
created: 2025-12-19T18:29:19.343Z
---

# Phase 7: Facets/Properties Feature

Implement optional facets system for attaching properties:
- Facet storage schema with dedicated key prefix
- SetFacet(component, key, value) - attach property to S/P/O
- GetFacet(component, key) - retrieve property
- GetFacets(component) - get all properties for component
- Triple-level facets for entire relationships
- Enable via WithFacets() option