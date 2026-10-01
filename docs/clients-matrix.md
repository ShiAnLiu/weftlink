# Clients Feature Matrix

> Tracks feature parity across all client implementations.
> HarmonyOS may lag but must not "silently miss" features.

| Feature            | Go CLI | Flutter Desktop | Flutter Mobile | ArkTS |
|--------------------|--------|-----------------|----------------|-------|
| discover (LAN)     | ✅     | ✅              | ✅             | ✅    |
| pair (knot)        | ✅     | ✅              | ✅             | ✅    |
| unpair (untie)     | ✅     | ✅              | ✅             | ✅    |
| status             | ✅     | ✅              | ✅             | ✅    |
| send file          | ✅     | ✅              | ✅             | ✅    |
| clipboard sync     | ✅     | ✅              | ✅             | ✅    |
| remote run         | ✅     | ✅              | ✅             | ✅    |
| screen view        | ❌ M4  | ❌ M4           | ❌ M4          | 🔍    |
| keyboard/mouse     | ❌ M4  | ❌ M4           | ❌ M4          | 🔍    |

✅ planned  ❌ not yet  🔍 exploring  ⚪ TBD