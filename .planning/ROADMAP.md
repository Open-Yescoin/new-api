# Roadmap: Token123 new-api Fork

**Created:** 2026-07-01

## Phase 0: Project Intelligence Baseline

Goal: Make the codebase navigable before more BytePlus or upstream work.

Deliverables:

- CodeGraph initialized in `/Users/lyz/new-api/.codegraph`.
- GSD `.planning/codebase/` documents generated for stack, integrations, architecture, structure, conventions, testing, and concerns.
- GSD project context, requirements, roadmap, and state created.

Requirements: `INTEL-01`, `INTEL-02`, `INTEL-03`

Status: In Progress

## Phase 1: Upstream Video Patch Plan

Goal: Isolate the minimum upstream changes needed for Seedance/Doubao video billing.

Deliverables:

- Confirm production/base/upstream commits before patching.
- Identify and review upstream commits `c8491b41`, `e514db20`, `d10fc762`, and optional `d2dcbc31`.
- Decide cherry-pick vs manual transplant.
- Produce a test plan for video task submit, poll, usage, billing, and refund.
- Record code rollback baseline and deployment artifact strategy.

Requirements: `UPSYNC-01`, `UPSYNC-02`, `UPSYNC-03`

Status: Pending

## Phase 2: BytePlus Seedance Billing Implementation

Goal: Make Seedance video billing match BytePlus overseas official pricing.

Deliverables:

- Apply or transplant dynamic video billing patch.
- Replace China-oriented `46/28/51/31/26/16` assumptions with BytePlus overseas USD/M ratios.
- Verify `usage.total_tokens` reaches async task settlement.
- Ensure video billing uses `ModelRatio` and `OtherRatios`, not `CompletionRatio` or `CacheRatio`.
- Diff all existing group/model pricing keys before and after patching; only new Seedance model keys may differ.

Requirements: `BYTE-01`, `BYTE-02`, `VIDEO-01`, `VIDEO-02`, `VIDEO-03`, `PROD-04`

Status: Pending

## Phase 3: Production Rollout Runbook

Goal: Safely expose Seedance video models without pricing leakage or rollback ambiguity.

Deliverables:

- Generate rollout, verify, and rollback SQL.
- Confirm BytePlus endpoint base models, region, status, model_mapping, vendor metadata, abilities, and descriptions.
- Obtain explicit production authorization for SQL and restart.
- Run smoke test, verify model marketplace display, verify video task billing, verify failed task refund.
- Keep low-price resolution aliases out of scope until parameter enforcement exists.

Requirements: `BYTE-03`, `VIDEO-04`, `PROD-01`, `PROD-02`, `PROD-03`

Status: Pending

## Phase 4: Full Upstream Upgrade Project

Goal: Treat the 254-commit upstream gap as a separate upgrade project, not part of Seedance rollout.

Deliverables:

- Create an isolated upgrade branch or workspace.
- Map upstream changes by domain.
- Run backend, frontend, DB, admin UI, channel, billing, and task regression tests.
- Produce a separate rollout and rollback plan.

Requirements: `UPGRADE-01`, `UPGRADE-02`

Status: Deferred

---
*Last updated: 2026-07-01 after initial GSD setup*
