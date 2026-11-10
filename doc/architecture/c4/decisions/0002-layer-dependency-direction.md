# 2. Layer dependency direction

Date: 2026-11-07

## Status

Accepted

## Context

The codebase is organized as a monorepo with several binaries and shared libraries. Without a fixed direction of imports, HTTP concerns leak into domain code and domain code imports entrypoints, which makes refactors costly and tests brittle.

## Decision

Package dependencies flow in one direction only, from entrypoints toward foundations.

```
api → app → business → foundation
```

Foundation code imports nothing else from this repository. Business imports only foundation. App imports business and foundation. Api may import all lower layers to wire and run processes.

Deployment and packaging assets are not application layers. They may reference binaries and images but must not become import targets for domain or handler code.

## Consequences

New code belongs in the lowest layer that can express it without violating the rule.

Circular needs between domains are resolved inside business through explicit coordination patterns, not by importing app or api.

Reviews can reject any change that introduces an upward import.
