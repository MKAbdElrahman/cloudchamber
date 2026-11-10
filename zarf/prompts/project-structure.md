# Project structure

Canonical architecture principles live in Structurizr under the C4 workspace: supplementary pages for context and containers, Decisions for layer dependencies and data flow.

## Dependency rule (quick reference)

```
api → app → business → foundation
```

No package may import from a layer above it. Cross layer data uses explicit conversion at app and store boundaries. See the layered architecture types agent skill for concrete patterns.
