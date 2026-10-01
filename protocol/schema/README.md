# Protocol Schema

Single source of truth for Weft Protocol type definitions.

- `weftlink.yaml` — canonical schema
- `conformance/` — conformance test vectors (JSON)

## Discipline

1. Edit only `weftlink.yaml` to add/change types.
2. Run `tools/gen` to regenerate all language types into `gen/`.
3. CI verifies that `gen/` matches `schema/`.
4. Never hand-edit files in `gen/`.