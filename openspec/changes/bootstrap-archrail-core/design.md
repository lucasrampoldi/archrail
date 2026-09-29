## Context

Archrail is a greenfield open-source CLI. The repository currently contains only a LICENSE. The v0 goal is a self-contained, deterministic architecture-conformance engine that a single team can adopt in one repository with no SaaS backend, while establishing the extension points that later enable central governance, more languages, and more agent adapters.

Two forces shape the design. First, the tool sits on both sides of the AI coding loop: it *emits* context (`sync`) that steers an agent, and it *validates* the result (`check`) as a gate. Both sides read the same architecture, so the architecture model is the center of gravity. The primary use case is an organization maintaining a **catalog of approved, authorized architectures** (frontend/backend blueprints) that new projects adopt and must conform to — so a named, versioned **Architecture Profile** is the central concept, and a project selects one from a shared catalog. Second, it must be honest about brownfield reality — legacy repos cannot conform to every rule on day one — so diff-scoping and a baseline are first-class, not afterthoughts.

Constraints: deterministic where possible; no code execution of the target repo; the deterministic core runnable offline (the opt-in semantic evaluator needs network + a key); Git-friendly (stable, diffable outputs); human-readable and agent-readable; independent of any specific AI agent; explicitly *not* a security/infra policy engine.

This version also brings **non-deterministic semantic evaluation** into scope: an opt-in LLM evaluator (via the Vercel AI Gateway, Claude) for architecture constraints that deterministic rules cannot express. It is deliberately isolated so it augments — never destabilizes — the deterministic gate.

## Goals / Non-Goals

**Goals:**
- A catalog of named, versioned **Architecture Profiles** (approved architectures) that projects adopt and are validated against.
- A machine- and human-readable standards format (topology + rules composing a profile) that is the single source of truth for both `sync` and `check`.
- Deterministic evaluation of a small, useful set of architecture rule types with no target-code execution and no network.
- Diff-scoped checking and a baseline so brownfield repos can adopt incrementally.
- Opt-in **non-deterministic semantic evaluation** via an LLM (Vercel AI Gateway, Claude) for constraints deterministic rules cannot express, advisory by default, with graceful degradation.
- Clear violation reporting (human + JSON) with severities and documented exit codes.
- Pre-commit and CI/PR integration built purely on the CLI's exit codes and JSON output.
- Extension points (rule-type registry, standards **source**, sync **adapter**, import **extractor**) so future capabilities are additive.

**Non-Goals (v0):**
- MCP servers, SaaS/central governance backend, dashboards, enterprise features.
- Full semantic/type-aware program analysis. v0 uses lightweight, deterministic import extraction, not a full compiler front-end. (Natural-language semantic checks are done by the LLM evaluator, not by static type analysis.)
- Making the LLM evaluator the primary gate. It is opt-in and advisory by default; the deterministic core remains the reliable, offline gate.
- Auto-fixing violations or generating code. Archrail reports and gates; it does not mutate source.

## Decisions

### DP1: Architecture Profile is the central unit; a catalog holds many

The unit of approval is a **profile**: a named, versioned bundle of `{topology + rules + approved stack}`. A **catalog** holds several approved profiles (`react-node-hexagonal`, `nextjs-fastapi-layered`, …). A project **adopts** one profile by pinning `name` + `version`. `sync` renders the adopted profile; `check` validates against it. **Why:** this directly matches the product's core use case ("a list of approved architectures that new projects comply with"), which a flat per-repo rule set cannot express — there's no notion of "which approved architecture is this". Making the profile the unit also gives a natural boundary for versioning and, later, central distribution. **Alternatives:** per-repo standalone rules only (rejected: no catalog, no selection, duplicated definitions across projects); one profile per repo with no versioning (rejected: can't evolve blueprints safely).

### DP2: Profiles resolved via a catalog **source**; v0 = local path to a shared Git catalog

The shared catalog is a Git repo/directory of profile definitions. In v0 it is consumed as a **local path** (clone, submodule, or vendored copy) resolved through a **catalog source** abstraction; remote/URL fetching is a documented extension point, not implemented. Version pinning is explicit; resolution is deterministic; no auto-upgrade. **Why:** delivers the "shared repo of approved architectures referenced by name+version" workflow while staying fully offline and deterministic at evaluation time, honoring the SaaS/central-governance non-goal. The source seam means a remote catalog is additive later without touching the profile format or a project's adoption declaration. **Alternatives:** fetch profiles over the network in v0 (rejected: violates offline/determinism goals and drags in the deferred central-governance concerns); copy the whole profile into each repo (rejected as the primary model: duplicates definitions, defeats a single approved catalog — still allowed via the vendored variant).

### D1: Standards (topology + rules) are authored in profiles; the project `.archrail/` adopts one, YAML

A profile definition (in the catalog) holds its topology (components as path globs, layers, defaults, extractor selection) and its rule declarations (`standards/*.yaml`). The adopting project's `.archrail/archrail.yaml` declares the catalog source, the adopted profile name+version, and any local topology mapping/overrides needed to bind the profile's abstract components to this repo's paths. **Why:** separating *topology* (what the parts of the system are) from *rules* (constraints over those parts) keeps rules small, reusable, and diff-friendly, and lets many rules reference shared component/layer definitions; keeping the reusable definition in the profile (not the repo) is what lets many projects share one approved architecture. Merging all `standards/*.yaml` lets teams split rules by concern. **Alternatives:** a single monolithic file (rejected: poor diffs, poor ownership boundaries); an embedded DSL / general-purpose policy language like Rego (rejected: non-deterministic-feeling, steep learning curve, and it pulls the product toward the security-policy framing we explicitly avoid).

### D2: Topology by path globs; a "component" is a set of files, a "layer" is a set of components

Rules operate over components/layers rather than raw paths, so the same rule set reads naturally as agent context ("handlers must not import repositories") and evaluates mechanically. **Why:** path globs are language-agnostic, deterministic, and cheap. **Alternatives:** package/module-system-native grouping (rejected for v0: language-specific and heavier); tagging via file annotations (rejected: intrusive to the target repo).

### D3: Deterministic built-in rule types only, resolved via a registry

v0 ships `forbidden-import`, `layer-boundary`, `forbidden-dependency`, `required-dependency`, `required-files`. Each is a pure function `(topology, changedSet, fileFacts) -> []Violation`. A **rule-type registry** maps `type` string → implementation. **Why:** these five cover the brief's canonical examples (handlers→DB, cross-service imports, required skeleton, forbidden/required infra deps, layer boundaries) with zero ambiguity and zero code execution. The registry makes new types additive without touching declaration syntax. **Alternatives:** user-authored custom rule plugins in v0 (deferred: registry is designed to allow it later).

### D4: Import extraction is pluggable and non-executing

Import-based rules need each file's imports. A per-language **extractor** derives imports via lightweight parsing/regex without executing the file. v0 ships extractors for a small starter set (e.g., a generic regex-based one plus one or two ecosystem-specific ones). A file with no matching extractor is *skipped with a note*, never silently passed as clean. **Why:** determinism and safety (never run untrusted repo code); honesty (skips are visible). **Alternatives:** full AST/type analysis (deferred: language-specific, heavy); executing build tools (rejected: unsafe, non-deterministic).

### D5: Diff-scoping via Git, opt-in with no silent fallback

`check --base <ref>` restricts *rule subjects* to files changed vs `<ref>` (via `git diff --name-only`). Without `--base`, check is full-repo. If `--base` is given but Git/ref is unavailable, it errors — it does **not** silently full-scan. **Why:** brownfield adoption needs "gate only new code"; silent fallback would turn a scoped gate into a repo-wide failure and erode trust. Note: topology and cross-file facts may still read unchanged files to resolve imports; only the *subjects* judged are the changed set.

### D6: Baseline file suppresses known violations, keyed by stable identity

A violation's identity is `(rule id, normalized file path, a stable content-derived locator)` — deliberately *not* raw line numbers, so unrelated edits don't churn the baseline. `check` suppresses baselined violations (reporting them as baselined) and fails only on new ones. A baseline-update command regenerates the file in sorted, diffable form. **Why:** lets legacy repos go green immediately while preventing new drift. **Trade-off:** a content-based locator can occasionally mismatch after heavy edits; acceptable for v0 and revisitable.

### D7: `sync` renders through an adapter abstraction; `ARCHRAIL.md` is the only v0 adapter

Rendering is deterministic (stable ordering/formatting) so unchanged standards produce no diff, and `sync --check` enforces freshness in CI. **Why:** `ARCHRAIL.md` is agent-agnostic and human-readable; the adapter seam means tool-specific targets are additive later without changing the standards format. **Alternatives:** shipping several tool-specific adapters now (deferred to keep v0 focused and agent-independent).

### D8: Documented exit codes as the integration contract

`0` = success/clean, `1` = violations found (non-baselined error), `2` = operational error (invalid standards, missing Git/base, bad usage). Pre-commit and CI recipes depend only on these and the JSON schema. **Why:** stable, scriptable integration with no bespoke glue.

### DS1: Semantic evaluation is a distinct, opt-in evaluator behind a backend seam

Semantic rules (`type: semantic`) carry a natural-language `assertion` and are evaluated by an **evaluator backend** — v0 implements a Vercel AI Gateway backend targeting a configurable **Claude** model. The evaluator receives the assertion + targeted file contents (never executes code) and returns a verdict. Context is deliberately curated: **only first-party source relevant to the target scope plus infra-relevant files (e.g., `Dockerfile`) are sent — never third-party dependencies or vendored/generated paths** (`node_modules`, `vendor`, `dist`, `build`, lockfiles, …), which are excluded by configurable defaults. This keeps prompts small, relevant, and cheaper, and avoids leaking irrelevant vendored code.

The evaluator does not ask the LLM to re-derive the architecture from raw text. Instead, Archrail's deterministic layer pre-computes an **architecture fingerprint** — components + their path mapping, a summarized component-to-component import graph, and the detected stack (manifests, `Dockerfile`) — and sends it alongside the assertion and targeted code. **Why:** the model reasons over already-verified structural facts (cheaper, more accurate, more stable run-to-run) rather than guessing imports/layers from source. The fingerprint is built from the same inputs as deterministic evaluation and reuses the import-extraction layer, so it costs little and stays consistent with what the deterministic rules see.

**Fingerprint computation (deterministic, no LLM, no code execution).** Inputs: the filtered repo file tree (excludes `.gitignore` + configured exclusions + deps/vendored/generated), the adopted profile's topology (component globs, layers), the import extractors, and manifests/infra files (`package.json`, `go.mod`, `Dockerfile`, …). Pipeline:

1. **Enumerate & filter** files; sort the list.
2. **Assign each file to a component** by glob match, with a fixed tie-break for overlapping globs (most-specific glob wins; ties broken by declaration order); unmatched files → `unassigned`.
3. **Extract imports** from each first-party file via its extractor.
4. **Resolve each import to a component**: repo-internal → its component; dependency → tagged `external` (grouped by package); unresolved → recorded as `unresolved` (never silently dropped).
5. **Aggregate edges**: `component → component` with counts (optionally a couple of example files), plus `component → external-package` from imports/manifests.
6. **Detect stack**: parse manifests (language, runtime, declared frameworks) and read the `Dockerfile` as text (base image, build/run commands) — never executed.
7. **Serialize canonically**: sorted collections, normalized relative paths, no timestamps/random → byte-stable output.

Output shape: `components[{name, paths, fileCount}]`, `layers[]`, `edges[{from, to, count}]`, `externalDeps[{component, package, version?}]`, `stack{languages, frameworks, runtime, dockerBaseImage}]`, and `unresolved[]` for transparency. This structure (graph + stack + layers) is what is sent to the LLM alongside the curated targeted code and the assertion. Findings are **`warn` (advisory) by default**, configurable to `error` per rule. The deterministic engine and the semantic evaluator are separate; `check` orchestrates both and merges results into one report, clearly labeling semantic findings as LLM-derived. **Why:** it delivers checks that mechanical rules cannot express ("does the handler actually delegate to a service?", "is the approved observability abstraction used correctly?") while quarantining non-determinism so it never silently destabilizes the deterministic gate. The backend seam keeps the LLM provider swappable. **Alternatives:** make every rule LLM-evaluated (rejected: non-deterministic gate, cost, offline breakage); embed a local model (rejected for v0: heavy, slower to ship). **Trade-offs:** cost, latency, network dependence, and variability — mitigated by advisory default, graceful degradation, and scoping context to targeted files.

### DS2: Secret handling and graceful degradation

The gateway API key is read only from an environment variable (`AI_GATEWAY_API_KEY`) and is never read from or written to the repo, catalog, `.archrail/`, `ARCHRAIL.md`, logs, or reports. When the LLM is unavailable (no key, no network, gateway error/timeout), affected `semantic` rules are **skipped and reported as "skipped: LLM unavailable"** — never as passing or failing — and deterministic evaluation/gating still completes. A `--require-semantic` (or config) switch flips this to a hard operational error for environments that mandate the LLM pass. **Why:** keeps the tool usable and CI green in offline/keyless contexts (e.g., forks, local runs) without ever faking a semantic verdict, and prevents credential leakage. **Alternatives:** fail the whole check when the LLM is down (rejected as default: brittle, blocks unrelated deterministic gating); cache verdicts to disk (deferred: raises staleness/consistency questions).

### D9: Implementation language — Go

A single statically-linked binary, easy cross-platform distribution, strong stdlib for filesystem/globbing/JSON and for shelling to Git, and it is the idiom for this class of devtool. **Why:** matches "single executable, offline, fast, Git-friendly." **Alternatives:** Rust (excellent but slower to iterate for v0 breadth), Python/Node (runtime dependency friction for a widely-distributed CLI gate). This decision is isolated behind the CLI/spec contract and does not leak into the standards format.

## Risks / Trade-offs

- **Regex/lightweight import extraction misses or misattributes imports** → Mitigate: never execute code; make extractor selection explicit; surface "skipped: no extractor" instead of false-clean; design the extractor seam so AST-based extractors can replace regex ones later without spec changes.
- **Path-glob topology is coarse vs real module boundaries** → Mitigate: keep components/layers explicit and reviewable in config; document limits; the topology seam allows module-native grouping later.
- **Baseline content-locator churn on heavy refactors** → Mitigate: sorted, diffable baseline + a regenerate command; accept occasional re-baseline in v0.
- **Diff-scoping gives false confidence (a change can break an unchanged file's assumptions)** → Mitigate: document that `--base` gates *changed subjects* only; recommend periodic full `check`; keep full-repo mode the default.
- **Determinism regressions** (map ordering, filesystem order) → Mitigate: sort all collections before evaluation/rendering; a golden-output test asserts byte-stability.
- **Scope creep toward a security-policy engine** → Mitigate: rule types and docs stay in the architecture-conformance domain; no security/CVE/infra-posture rule types in v0.
- **LLM non-determinism destabilizes the gate** → Mitigate: semantic findings are advisory (`warn`) by default and clearly labeled; deterministic byte-stability is claimed only for deterministic rules; `error` severity for semantic rules is explicit opt-in per rule.
- **Secret leakage of the gateway API key** → Mitigate: key only via env var; never read from/written to repo, catalog, `.archrail/`, `ARCHRAIL.md`, logs, or reports; a test asserts the key never appears in any output.
- **LLM cost / latency / rate limits** → Mitigate: only `semantic` rules call the LLM; scope context to targeted files; diff-scoping limits subjects; graceful degradation on error/timeout; (verdict caching is a documented future option).
- **False verdicts from the LLM** → Mitigate: advisory default keeps humans in the loop; findings are labeled LLM-derived with the model id; baseline can suppress known-noisy findings.
- **Offline/keyless environments** → Mitigate: deterministic core runs without network or key; semantic rules skip-with-note unless `--require-semantic` is set.

## Migration Plan

Greenfield tool; no data migration. First, an org authors a **profile catalog** (a Git repo/dir of approved architectures). Adoption path for a target repo: make the catalog available locally (clone/submodule/vendored) → `archrail init` and set the catalog source + adopted profile name+version → bind the profile's components to repo paths in `.archrail/archrail.yaml` → `archrail sync` (commit `ARCHRAIL.md`) → `archrail check --base <default-branch>` locally → generate baseline for legacy violations → wire pre-commit and CI recipes. Rollback is removal of the `.archrail/` directory, `ARCHRAIL.md`, and the hook/CI steps; nothing in Archrail is stateful outside the repo.

## Open Questions

- How is a profile's abstract component bound to a concrete repo's paths — purely via the project's local topology mapping, or can a profile ship default path conventions that the repo opts into?
- Should the "approved technology stack" (frontend/backend frameworks) be expressed only through existing `required/forbidden-dependency` rules, or does it warrant a dedicated `approved-stack` rule type in v0?
- Catalog layout: one profile per directory vs one file per profile; how versions are represented (directory per version, semver tags, or a version field with multiple files).
- Should `archrail init` support interactive profile selection from the resolved catalog?
- Exact glob-precedence rule for overlapping component definitions in the fingerprint (most-specific-wins vs declaration-order) and how monorepos are handled.
- How verbose the fingerprint's import graph should be (edge existence only vs counts vs example files) to balance signal against prompt size, and the exact serialized field set.
- Default Claude model id for the semantic evaluator, and how much file context to send per assertion (whole component vs targeted files vs chunking) to balance accuracy and cost.
- Whether semantic verdicts should be cached (keyed by content + assertion + model) to reduce cost and add run-to-run stability, and where that cache would live.
- Exact env var name and whether to also support a Vercel AI Gateway base-URL override for self-hosted/proxy setups.
- Which ecosystem-specific import extractors ship first in v0 beyond the generic regex extractor (e.g., Python, JS/TS, Go)?
- Baseline file format details (single file vs per-standard) and the exact content-locator algorithm.
- Whether `layer-boundary` should model layers as a strict order (upper→lower only) or a general allowed-dependency graph for v0 (leaning graph for expressiveness).
- Minimum `ARCHRAIL.md` structure that agents consume most reliably (headings, rule blocks, examples).
