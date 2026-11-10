# Structurizr DSL notes (CloudChamber)

Canonical examples live in `references/structurizr-java/structurizr-dsl/src/test/resources/dsl/`. Read the Java view classes when `include *` behaviour is unclear.

## Model

- **C1 / C2 / C3 relationships are separate edges** in the model (see `big-bank-plc/internet-banking-system.dsl`). Do not tag relationships per view; each diagram uses the edges whose endpoints appear in that view.
- **Do not repeat the same story on every level.** C2: people and containers only. C3: component chains inside each process plus the few cross-container component links you need (e.g. auth client to gRPC auth). With `!impliedRelationships false`, declare C2 container links explicitly; do not rely on implied lift from C3.
- **Component views need caller edges.** Structurizr only shows people when a relationship connects them to something in the view (`addNearestNeighbours` is not enough without a model edge). Add at least `endUser -> entryComponent` and `adminUI -> entryComponent` (container to component is valid; see big-bank-plc).
- **`!impliedRelationships false`** keeps system context to the two person→system edges only. With implied relationships enabled, every person→container and person→component link also creates person→software system summaries (parent walk in `CreateImpliedRelationshipsUnlessSameRelationshipExistsStrategy`), which clutters C1.
- At **C2**, declare container→container links explicitly (e.g. product API→Auth, Auth→PostgreSQL) when implied relationships are off.
- **Components:** `id = component "Name" "Description" "Technology" "Tag"` (`ComponentParserTests`).
- **Containers with shape:** fourth token is a tag (e.g. `Database`, `Web Browser`), not a separate technology field when using four string slots after name.

## Views

| View | Reference pattern | CloudChamber |
|------|------------------|-------|
| System context | People + system; only explicit C1 edges | `include operator endUser cloudchamber` (no `include *?` when implied rels are off) |
| Container | `include *` | `big-bank-plc` container view |
| Component | `include *` + optional `animation { }` | Scoped with `component cloudchamber.cloudchamberAPI "…"` |

### `include *` vs `include *?` (structurizr-core)

- **`include *`:** `addDefaultElements(true)` — greedy default neighbours and relationships.
- **`include *?`:** `addDefaultElements(false)` — keeps relationships only to/from the scoped system (context), containers (container view), or components (component view).

Do not combine greedy `include *` with `include *?` on the same view unless you intend both passes (`StaticViewContentParser`).

### Theme and styles

- **Theme:** `theme themes/default/theme.json` in `views { }` — vendored copy of [structurizr/themes `default`](https://github.com/structurizr/themes/tree/master/default) plus `Component`, `Database`, and `Web Browser` shapes. Use `theme default` instead when online (same palette via `static.structurizr.com`; `make c4-validate` needs the file when DNS is unavailable).
- Keep **workspace `styles`** minimal so they do not override the theme: CloudChamber only styles **ADR** decision tags (`Decision:Accepted`, etc.).

### Validation

`make c4-validate` against the pinned Structurizr image in the makefile.
