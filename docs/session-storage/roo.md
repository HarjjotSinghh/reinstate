# Roo Code

**Confidence: Unverified** — no Reinstate reader exists, and no vendor source
has been verified in this research pass.
**Current tier:** T0 (`layout_unverified`) · **Phase 5 target:** T1

Roo Code is an editor extension with the same architecture class as
[Cline](cline.md). Implement it **after** Cline, and reuse the F3 scanner,
multi-host root resolution, and host-attribution decisions that task
establishes. If the two layouts turn out to be identical in shape, the two
descriptors still stay separate files with separate keys, because they are
separate products a user can have both of.

## Identity

| Aspect | Value |
| ------ | ----- |
| Vendor | Roo |
| Product | Roo Code (editor extension) |
| Host | VS Code and compatible editors |
| Distribution | Official, extension marketplace |
| Storage family | F3 (editor extension storage) expected |

## Working hypothesis, not evidence

Per-extension global storage under the host editor, one directory per task.
Unconfirmed, and it must be confirmed independently of Cline: a shared origin
does not guarantee a shared current layout, and assuming it is exactly the
mistake that produces a scanner which silently finds nothing.

## What the probe must settle

The same six questions as [Cline](cline.md), answered independently:

1. Storage root per installed host, on macOS and native Windows.
2. Per-task directory shape and task ID stability.
3. Which file carries user-visible turns versus UI-render state.
4. Whether the workspace path is recorded.
5. Whether an on-disk version marker exists.
6. Whether the extension exposes its own export or history surface.

Additionally: whether Roo and Cline can collide in the same storage tree. If a
user has both installed, records must be attributed to the correct product.

## Constraints inherited from F3

No `PATH` executable, so no version probe and no resume argv. T3 is not
reachable through the current launch mechanism. State that as a property, not
a pending task.

## Sources

None verified. Establish and record vendor sources before promoting any row.
