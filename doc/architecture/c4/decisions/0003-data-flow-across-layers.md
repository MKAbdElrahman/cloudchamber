# 3. Data flow across layers

Date: 2026-11-07

## Status

Accepted

## Context

The same concepts appear on the wire, in domain logic, and in storage. Sharing one struct or one primitive type across those surfaces hides where validation belongs and lets invalid data reach business rules.

## Decision

Each crossing between layers uses an explicit conversion. The same entity may have a wire shape, a domain shape, and a storage shape. They are not shared types.

Primitives and serialization tags belong at the HTTP edge and at the storage row. Strong value types belong in business code after conversion.

Validation and parsing run when data moves inward from an edge. Outward moves map trusted domain values to wire or row forms without revalidating business rules.

Business logic assumes inputs are already valid for their types. Storage access goes through interfaces so SQL and cache details do not flow upward.

## Consequences

Handlers stay thin: decode, convert, call business, convert, return an encoded result.

Domain packages stay free of JSON tags and HTTP status mapping.

Tests can target the layer where a bug lives because boundaries are explicit.
