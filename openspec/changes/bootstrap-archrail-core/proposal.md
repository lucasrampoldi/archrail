## Why

AI coding agents can produce technically working software that quietly violates an organization's architecture: it crosses service boundaries, bypasses approved abstractions, ignores the required repository skeleton, or introduces forbidden dependencies. Existing policy engines (OPA, Kyverno, Checkov) govern security and infrastructure posture — nobody governs *architecture conformance* in the AI-assisted development loop. Archrail closes that gap: it maintains a catalog of **approved, authorized architectures** (frontend/backend blueprints), turns the architecture a project adopts into context the agent consumes *before* it writes code, and into a deterministic gate that validates changes *after*. The thesis: **AI can generate working software; Archrail makes sure it belongs in your architecture.**

This change establishes a minimal but useful v0: an organization can define a small catalog of approved architecture profiles, a project declares which profile it adopts, and Archrail syncs that profile to the agent and gates conformance. Profiles are resolved from a shared, Git-based catalog referenced by name and version (consumed as a local path — clone/submodule/vendored — with no SaaS backend), keeping the design extensible toward centrally managed organizational standards later.

## What Changes

- Introduce **Architecture Profiles** as the central concept: named, versioned, approved architectures (each a bundle of topology + rules + approved technology stack). A shared catalog can hold several approved profiles (e.g., `react-node-hexagonal`, `nextjs-fastapi-layered`).
- Let a project **adopt a profile** by declaring it (name + version) in its `.archrail/` config; Archrail resolves the profile from a catalog **source** (local path to a shared Git catalog in v0; remote fetch is a documented extension point).
- Introduce a machine-readable, human-readable, Git-friendly way to declare architecture standards (the topology + rules that compose a profile), stored under an `.archrail/` directory (project config) and within profile definitions (catalog).
- Add the `archrail` CLI with two primary subcommands:
  - `archrail sync` — render the active standards into agent-consumable context, canonically an `ARCHRAIL.md` file, through a pluggable **adapter** abstraction (only the `ARCHRAIL.md` adapter ships in v0).
  - `archrail check` — deterministically validate repository state (or a Git diff) against the standards and report architecture violations.
- Support **diff-scoped** checking (`--base <ref>`) so brownfield repositories can gate only changed code instead of retrofitting every rule at once.
- Support a **baseline** file that records pre-existing violations so legacy repos can adopt Archrail without an immediate red build.
- Ship a small set of **deterministic rule types** covering the most common architecture constraints (layer/boundary rules, forbidden/required imports, forbidden/required dependencies including approved-stack enforcement, required skeleton files).
- Add **non-deterministic semantic evaluation** via an LLM: a `semantic` rule type evaluated through the **Vercel AI Gateway** (Claude model), for constraints that deterministic rules cannot express (e.g., "the handler actually delegates to a service", "observability abstraction is used correctly"). Semantic evaluation is **opt-in**, requires network + an API key supplied via environment variable (never committed), and its findings are **advisory (`warn`) by default**, configurable to `error` per rule. The deterministic core continues to run fully offline when no `semantic` rules are configured.
- Provide clear, machine- and human-readable **violation reporting** (human text + JSON) with per-rule severity and meaningful exit codes.
- Provide **workflow integration**: a pre-commit hook recipe and a CI/PR gate recipe built on the same `archrail check` command.
- Establish extension points (rule-type registry, profile-catalog **source** abstraction, sync **adapter** abstraction) so future central governance, remote catalogs, more languages, and more adapters are additive, not rewrites.

**Non-goals for v0 (explicitly deferred):** MCP servers, centralized SaaS governance, dashboards, and enterprise features. Design accommodates them as extension points but does not implement them. (LLM-based semantic analysis is now **in scope** for this version, as an opt-in, advisory-by-default evaluator — see below.)

## Capabilities

### New Capabilities

- `architecture-profiles`: The catalog of named, versioned, approved architectures; the profile-catalog **source** abstraction (local shared-catalog path in v0); a project adopting a profile by name+version; profile resolution, versioning, and validation.
- `architecture-standards`: The machine-readable format, on-disk layout, loading, and validation of the topology + rules that compose a profile, plus the deterministic rule-type model and the extension points (rule-type registry, import extractors).
- `agent-context-sync`: `archrail sync` — rendering the adopted profile into agent-consumable context via the adapter abstraction, canonically the `ARCHRAIL.md` file.
- `conformance-check`: `archrail check` — the evaluation engine validating a project against its adopted profile, orchestrating both deterministic and semantic rules, diff-scoping against a Git base ref, the baseline mechanism for brownfield adoption, violation reporting (human + JSON), severities, and exit codes.
- `semantic-evaluation`: The non-deterministic, opt-in LLM evaluator: the `semantic` rule type, the **evaluator backend** abstraction, the Vercel AI Gateway (Claude) integration, API-key handling via environment variable, advisory-by-default severity, and graceful degradation when the LLM is unavailable.
- `workflow-integration`: Packaging the CLI and integrating it into local development, pre-commit hooks, and CI/PR gates.

### Modified Capabilities

<!-- None. This is the initial specification; no existing specs. -->

## Impact

- New project: `archrail` CLI (greenfield; no existing code beyond LICENSE).
- New on-disk contracts: a **profile catalog** (a shared Git repo/directory of profile definitions) and, in each adopting repository, an `.archrail/` directory (project config declaring the adopted profile + any local overrides/topology), generated `ARCHRAIL.md`, optional `.archrail/baseline.*`.
- New integration surfaces: pre-commit hook, CI job invoking `archrail check`.
- Dependencies: Git (for diff-scoping) is required at runtime for `--base`; deterministic `check`/`sync` must run without it. The profile catalog is a local path in v0 (no network required for deterministic evaluation).
- New optional runtime dependency: network access + a Vercel AI Gateway API key (via environment variable, e.g. `AI_GATEWAY_API_KEY`) — required only when `semantic` rules are evaluated. The key MUST NOT be stored in the repository, the catalog, or `.archrail/`.
