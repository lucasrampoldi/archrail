## 1. Project scaffolding

- [x] 1.1 Initialize the Go module and repository layout (`cmd/archrail`, `internal/`), CI build, and `make build`/`make test` targets
- [x] 1.2 Add the CLI framework with a root `archrail` command, `--help`, and stubs for `init`, `sync`, `check` subcommands
- [x] 1.3 Define documented exit codes (0 success, 1 violations, 2 operational error) as a shared package used by all subcommands

## 2. Architecture profiles and catalog (architecture-profiles)

- [x] 2.1 Define the profile definition model (`name`, `version`, `description`, topology + rules, optional approved stack) and its on-disk catalog layout
- [x] 2.2 Implement the catalog **source** abstraction with a local-path implementation (shared Git catalog as clone/submodule/vendored); error clearly when the path is missing/unreadable
- [x] 2.3 Implement catalog enumeration (list available profiles with name/version/description) and duplicate name+version detection
- [x] 2.4 Implement project adoption: parse adopted profile name+version from `.archrail/archrail.yaml`, resolve deterministically with explicit version pinning and no auto-upgrade
- [x] 2.5 Implement resolution errors: profile not found (list available), version not found (list versions), no profile declared (guidance, non-passing)

## 3. Standards model and loading (architecture-standards)

- [x] 3.1 Define the config/topology schema (version, components as path globs, layers, default severity, extractor selection, local overrides) with typed structs
- [x] 3.2 Implement `.archrail/` discovery by walking up to the repository root; error clearly when absent
- [x] 3.3 Load and merge a profile's `standards/*.yaml` into one active rule set, preserving originating file for reporting
- [x] 3.4 Implement config + profile validation: schema version check, unknown-key rejection with file/key location, duplicate rule-`id` detection
- [x] 3.5 Define the rule declaration model (`id`, `type`, `description`, `severity`, params) and the **rule-type registry** abstraction with an unknown-type error listing known types
- [x] 3.6 Add a command/flag to enumerate registered rule types and their required parameters

## 4. Import extraction (architecture-standards)

- [x] 4.1 Define the import **extractor** interface (`file -> imports`, no code execution) and per-extension selection from config
- [x] 4.2 Implement a generic regex-based extractor plus at least one ecosystem-specific extractor
- [x] 4.3 Implement "skipped: no extractor" reporting for files with no matching extractor (never counted as clean)

## 5. Rule engine and built-in rule types (architecture-standards, conformance-check)

- [x] 5.1 Implement the evaluation core: resolve topology, compute file facts, run each rule as a pure `(topology, subjects, facts) -> []Violation`
- [x] 5.2 Enforce determinism: sort all file/collection iteration; add a golden-output byte-stability test
- [x] 5.3 Implement `forbidden-import`
- [x] 5.4 Implement `layer-boundary` (allowed-dependency graph between layers)
- [x] 5.5 Implement `forbidden-dependency` and `required-dependency` (manifest inspection, incl. approved-stack enforcement)
- [x] 5.6 Implement `required-files` (component skeleton presence)

## 6. Semantic evaluation (semantic-evaluation)

- [x] 6.1 Define the `semantic` rule type (`assertion` + target scope) and register it in the rule-type registry with validation
- [x] 6.2 Define the **evaluator backend** abstraction (assertion + file context → verdict; no code execution)
- [x] 6.3 Build the deterministic **architecture fingerprint** (components + path mapping, summarized import graph, detected stack from manifests/`Dockerfile`), reusing the import-extraction layer; assert it is byte-stable and excludes dependency/vendored paths
- [x] 6.4 Implement the Vercel AI Gateway (Claude) backend: configurable model id with a documented default, read API key from env var only
- [x] 6.5 Enforce secret hygiene: never read/write the key to repo/catalog/`.archrail/`/`ARCHRAIL.md`/logs/reports; add a test asserting the key never appears in output
- [x] 6.6 Implement advisory-by-default severity (`warn`), configurable to `error` per rule
- [x] 6.7 Implement graceful degradation (skip-with-note on missing key/network/error/timeout) and a `--require-semantic` switch that turns unavailability into an operational error
- [x] 6.8 Curate context sent to the LLM: fingerprint + first-party targeted source + infra-relevant files (e.g., `Dockerfile`); exclude dependencies/vendored/generated paths (`node_modules`, `vendor`, `dist`, `build`, lockfiles) via configurable defaults; respect diff-scoping; label findings as LLM-derived with the model id

## 7. Checking, diff-scoping, baseline (conformance-check)

- [x] 7.1 Implement `archrail check` resolving the adopted profile, full-repo mode, orchestrating deterministic + semantic rules, with validation gating
- [x] 7.2 Implement diff-scoping via `--base <ref>` using `git diff --name-only`; error (no silent fallback) when Git/ref unavailable
- [x] 7.3 Implement severity gating: exit 1 iff a non-baselined `error` exists; `warn` (incl. advisory semantic findings) never fails on its own
- [x] 7.4 Define the violation identity (rule id, normalized path, content-derived locator) and implement the baseline file format
- [x] 7.5 Implement baseline suppression during check (report baselined separately) and a baseline-update command

## 8. Reporting (conformance-check)

- [x] 8.1 Implement the human-readable report (rule id, severity, file, location, message, source file; mark semantic findings as LLM-derived/advisory)
- [x] 8.2 Implement `--format json` with a documented, stable schema (incl. semantic/LLM-derived and skipped fields); add a schema fixture/test
- [x] 8.3 Document the exit-code contract in CLI help and README

## 9. Sync and adapters (agent-context-sync)

- [x] 9.1 Implement the sync **adapter** abstraction with name-based selection defaulting to `ARCHRAIL.md`; unknown adapter lists available ones
- [x] 9.2 Implement the `ARCHRAIL.md` adapter: deterministic rendering (stable order/format), profile name+version header, generated-by-Archrail marker, faithful rule representation (incl. semantic assertions)
- [x] 9.3 Gate sync on profile resolution + validation (refuse to render if unresolved/invalid, write nothing)
- [x] 9.4 Implement `sync --check` drift detection (exit non-zero when on-disk output is stale, write nothing)

## 10. Workflow integration (workflow-integration)

- [x] 10.1 Implement `archrail init` scaffolding a valid `.archrail/` declaring catalog source + adopted profile; refuse to clobber existing without override
- [x] 10.2 Implement `archrail profiles` to list profiles available in the resolved catalog
- [x] 10.3 Provide a documented pre-commit hook recipe running `archrail check` on staged changes
- [x] 10.4 Provide a documented CI/PR gate recipe running `archrail check --base <target>` and `archrail sync --check`; document how to provide the gateway key as a CI secret
- [x] 10.5 Produce release/distribution artifacts (single static binary; install instructions)

## 11. Testing and documentation

- [x] 11.1 Unit tests for catalog/profile resolution, loading/validation, each rule type, extractors, diff-scoping, and baseline behavior
- [x] 11.2 Tests for the semantic evaluator: verdict mapping, advisory-vs-error, graceful degradation paths, `--require-semantic`, and secret non-leakage (mock the gateway)
- [x] 11.3 End-to-end tests on a fixture catalog + repo asserting spec scenarios (profile adoption, clean, error, warn, baselined, diff-scoped, stale sync, semantic skipped/advisory) and exit codes
- [x] 11.4 Author README and a getting-started guide covering the adoption path (author catalog → init/adopt profile → sync → check → baseline → hooks/CI) and semantic-eval setup (env var, model config)
