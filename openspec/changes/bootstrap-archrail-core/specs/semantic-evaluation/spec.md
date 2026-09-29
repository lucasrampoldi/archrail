## ADDED Requirements

### Requirement: Semantic rule type

Archrail SHALL provide a `semantic` rule type that expresses an architecture constraint in natural language for evaluation by an LLM. A `semantic` rule MUST declare, in addition to the common rule fields (`id`, `type`, `description`, `severity`), a natural-language `assertion` describing the expected property and the target scope (component/layer/paths) it applies to. Semantic rules SHALL be used for constraints that deterministic rule types cannot express.

#### Scenario: Well-formed semantic rule is registered

- **WHEN** a profile declares a rule with `type: semantic`, a target scope, and a natural-language `assertion`
- **THEN** Archrail SHALL register it as an evaluable semantic rule

#### Scenario: Semantic rule missing assertion is rejected

- **WHEN** a `semantic` rule omits its `assertion` or its target scope
- **THEN** Archrail SHALL exit non-zero and report the rule `id` and the missing field

### Requirement: Evaluation via Vercel AI Gateway (Claude)

Semantic rules SHALL be evaluated by sending the rule `assertion` together with the relevant code context to an LLM through the Vercel AI Gateway, using a Claude model. The model identifier SHALL be configurable in `.archrail/archrail.yaml` (or the profile), defaulting to a Claude model. Archrail MUST NOT execute repository code to gather context; it MUST only read file contents.

#### Scenario: Semantic rule is evaluated against code context

- **WHEN** a `semantic` rule targets a component and the LLM evaluator is configured and reachable
- **THEN** Archrail SHALL send the assertion and the targeted files' contents to the configured Claude model via the Vercel AI Gateway and obtain a verdict

#### Scenario: Model is configurable

- **WHEN** the configuration specifies a Claude model identifier
- **THEN** Archrail SHALL use that model, and SHALL fall back to a documented default Claude model when none is specified

### Requirement: Structured architecture context (fingerprint)

Semantic evaluation SHALL provide the LLM with a deterministically pre-computed **architecture fingerprint** in addition to the targeted code, so the model reasons over verified structural facts rather than re-deriving them from raw text. The fingerprint MUST be produced by Archrail's deterministic layer and MUST include at least: the components and their path mapping (from the adopted profile/topology), a summarized component-to-component import/dependency graph, and the detected technology stack (from manifests and infra files such as `Dockerfile`). The fingerprint MUST be built from the same inputs as deterministic evaluation and MUST NOT include excluded dependency/vendored/generated content.

#### Scenario: Fingerprint accompanies the assertion

- **WHEN** a `semantic` rule is evaluated
- **THEN** the context sent to the LLM SHALL include the architecture fingerprint (components + path mapping, summarized import graph, detected stack) alongside the rule `assertion` and the targeted code

#### Scenario: Fingerprint is deterministic

- **WHEN** the fingerprint is generated twice for the same repository state and profile
- **THEN** it SHALL be byte-identical across runs

#### Scenario: Fingerprint respects exclusions

- **WHEN** the fingerprint is generated for a repository containing vendored dependencies
- **THEN** the fingerprint SHALL NOT incorporate content from excluded dependency/vendored/generated paths

### Requirement: Context selection excludes dependencies

The code context sent to the LLM SHALL be limited to first-party source relevant to the rule's target scope, plus infrastructure-relevant files where pertinent (e.g., `Dockerfile`, compose/manifest files declared as relevant). Archrail MUST NOT send third-party dependency or vendored/generated content (e.g., `node_modules`, `vendor`, `dist`, `build`, lockfile contents, and other configured-excluded paths) to the LLM. The set of excluded paths SHALL be configurable, with sensible defaults for common ecosystems. When diff-scoping is active, context SHALL be further limited to the changed subjects and their relevant first-party neighbors.

#### Scenario: Dependencies are not sent to the LLM

- **WHEN** a `semantic` rule is evaluated in a repository containing vendored dependencies (e.g., `node_modules`)
- **THEN** the content of those dependency/vendored/generated paths SHALL NOT be included in the context sent to the LLM

#### Scenario: Dockerfile and relevant code are included

- **WHEN** a `semantic` rule's scope covers a component and the repository has a `Dockerfile` declared as relevant
- **THEN** Archrail SHALL include the targeted first-party source and the `Dockerfile` in the context sent to the LLM

#### Scenario: Exclusions are configurable

- **WHEN** the configuration declares additional excluded paths
- **THEN** those paths SHALL be omitted from any context sent to the LLM, in addition to the built-in defaults

### Requirement: API key supplied via environment, never persisted

The Vercel AI Gateway API key SHALL be read exclusively from an environment variable (e.g., `AI_GATEWAY_API_KEY`). Archrail MUST NOT read the key from, or write the key to, the repository, the profile catalog, `.archrail/`, generated `ARCHRAIL.md`, logs, or reports. When semantic evaluation is requested but the key is absent, Archrail SHALL treat it as the LLM being unavailable (see graceful degradation).

#### Scenario: Key read from environment

- **WHEN** the environment variable holding the API key is set and a `semantic` rule is evaluated
- **THEN** Archrail SHALL authenticate to the gateway using that value

#### Scenario: Key never leaks to outputs

- **WHEN** Archrail produces any report, log line, or generated file
- **THEN** the API key value SHALL NOT appear in that output

### Requirement: Advisory by default, configurable severity

Findings from `semantic` rules SHALL default to severity `warn` (advisory: reported but not gate-failing). A `semantic` rule MAY explicitly set `severity: error` to make its findings gate-failing. Severity gating SHALL follow the same contract as deterministic rules (a non-baselined `error` fails the gate).

#### Scenario: Advisory semantic finding does not fail the gate

- **WHEN** a `semantic` rule with default severity produces a violation and no deterministic `error` exists
- **THEN** `archrail check` SHALL report the finding and exit zero

#### Scenario: Semantic rule marked error fails the gate

- **WHEN** a `semantic` rule explicitly set to `error` produces a non-baselined violation
- **THEN** `archrail check` SHALL exit non-zero

### Requirement: Non-determinism is disclosed

Because semantic evaluation is non-deterministic, its output MUST be clearly marked as LLM-derived and advisory in both human and JSON reports, and MUST be distinguishable from deterministic findings. The deterministic byte-stability guarantee SHALL apply only to deterministic rules; it SHALL NOT be claimed for `semantic` findings.

#### Scenario: Semantic findings are labeled

- **WHEN** `archrail check` reports a `semantic` finding
- **THEN** the report SHALL mark it as LLM-derived (and advisory unless configured as error) and identify the evaluating model

#### Scenario: Deterministic findings remain stable

- **WHEN** the same repository state is evaluated twice with the same profile
- **THEN** the deterministic findings SHALL be byte-identical regardless of any `semantic` findings' variation

### Requirement: Graceful degradation when the LLM is unavailable

When semantic evaluation cannot run (missing API key, no network, gateway error, or timeout), Archrail SHALL NOT crash the whole check. It SHALL skip the affected `semantic` rules, report them as "skipped: LLM unavailable" (never as passing and never as failing), and still complete deterministic evaluation and gating. A configuration or flag MAY require semantic evaluation to run (failing the run if it cannot), for environments that mandate it.

#### Scenario: Missing key degrades gracefully

- **WHEN** a `semantic` rule is present but no API key is available
- **THEN** Archrail SHALL skip that rule, report it as skipped (LLM unavailable), and still evaluate deterministic rules and produce the normal gate result

#### Scenario: Gateway error during evaluation

- **WHEN** the Vercel AI Gateway returns an error or times out for a `semantic` rule
- **THEN** Archrail SHALL report that rule as skipped (LLM unavailable) rather than failing the entire check, unless semantic evaluation was explicitly required

#### Scenario: Semantic evaluation explicitly required

- **WHEN** the configuration/flag requires semantic evaluation and the LLM is unavailable
- **THEN** Archrail SHALL exit with an operational error rather than silently skipping
