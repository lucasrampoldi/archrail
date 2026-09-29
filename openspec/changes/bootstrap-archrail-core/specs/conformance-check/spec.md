## ADDED Requirements

### Requirement: Check evaluates repository against active standards

`archrail check` SHALL resolve the project's adopted architecture profile (see architecture-profiles) and evaluate the repository against that profile's validated standards, reporting all architecture violations. Check MUST validate the resolved profile before evaluation and MUST refuse to run if the profile cannot be resolved or is invalid. Check MUST NOT execute repository code. Deterministic rule evaluation MUST be deterministic; `semantic` rules (see semantic-evaluation) are non-deterministic and advisory by default and are orchestrated by the same `check` run.

#### Scenario: Full repository check with no violations

- **WHEN** `archrail check` runs and no rule is violated
- **THEN** it SHALL report zero violations and exit zero

#### Scenario: Check orchestrates deterministic and semantic rules

- **WHEN** `archrail check` runs against a profile containing both deterministic and `semantic` rules
- **THEN** it SHALL evaluate the deterministic rules and, when the LLM evaluator is available, the `semantic` rules, and combine both into one report

#### Scenario: Check refuses invalid standards

- **WHEN** `archrail check` runs against standards that fail validation
- **THEN** it SHALL report the validation error and exit non-zero without producing a violation report

### Requirement: Diff-scoped checking

`archrail check` SHALL support scoping evaluation to files changed relative to a Git base ref via `--base <ref>`. In diff-scoped mode, only files in the changed set SHALL be evaluated as rule subjects, allowing brownfield repositories to gate new changes without conforming the entire history. When `--base` is not supplied, check SHALL evaluate the full repository.

#### Scenario: Only changed files are evaluated

- **WHEN** `archrail check --base <ref>` runs in a repository with changes relative to `<ref>`
- **THEN** only files changed relative to `<ref>` SHALL be evaluated as rule subjects and violations SHALL be limited to that set

#### Scenario: Diff mode without Git available

- **WHEN** `archrail check --base <ref>` runs but Git or the base ref is not available
- **THEN** Archrail SHALL exit non-zero with a clear message and SHALL NOT silently fall back to a full check

#### Scenario: No changed files

- **WHEN** `archrail check --base <ref>` runs and there are no changes relative to `<ref>`
- **THEN** Archrail SHALL report zero violations and exit zero

### Requirement: Baseline for brownfield adoption

`archrail check` SHALL support a baseline file that records known, pre-existing violations. Violations present in the baseline SHALL be suppressed from the failing set but reported as baselined. A command SHALL exist to generate or update the baseline from the current violation set. Newly introduced violations not present in the baseline SHALL still fail the check.

#### Scenario: Baselined violation is suppressed

- **WHEN** `archrail check` finds a violation that is recorded in the baseline
- **THEN** the violation SHALL NOT cause a non-zero exit and SHALL be reported as baselined

#### Scenario: New violation not in baseline fails

- **WHEN** `archrail check` finds a violation that is not recorded in the baseline
- **THEN** the check SHALL exit non-zero and report the new violation

#### Scenario: Baseline generation

- **WHEN** the user runs the baseline-update command
- **THEN** Archrail SHALL write the current violation set to the baseline file in a deterministic, diff-friendly format

### Requirement: Severity governs gating

Each violation SHALL carry the severity of its rule (`error` or `warn`). `archrail check` SHALL exit non-zero if and only if at least one non-baselined `error` violation exists. `warn` violations SHALL be reported but SHALL NOT by themselves cause a non-zero exit.

#### Scenario: Warn-only run passes the gate

- **WHEN** `archrail check` finds only `warn` violations (no non-baselined `error`)
- **THEN** it SHALL report the warnings and exit zero

#### Scenario: Any error fails the gate

- **WHEN** `archrail check` finds at least one non-baselined `error` violation
- **THEN** it SHALL exit non-zero

### Requirement: Violation reporting in human and machine formats

`archrail check` SHALL produce a human-readable report by default and SHALL support a machine-readable JSON report via a format flag. Every reported violation MUST include: rule `id`, severity, the offending file path (and location where available), a human-readable message, and the source standards file. The JSON output MUST have a stable, documented schema suitable for CI consumption.

#### Scenario: Human-readable default output

- **WHEN** `archrail check` runs without a format flag
- **THEN** it SHALL print a human-readable summary listing each violation's rule id, severity, file, and message

#### Scenario: JSON output for CI

- **WHEN** `archrail check --format json` runs
- **THEN** it SHALL emit valid JSON conforming to the documented schema, listing every violation with rule id, severity, file, location, message, and source file

#### Scenario: Distinct exit codes are documented

- **WHEN** `archrail check` completes
- **THEN** it SHALL use documented, distinct exit codes for success, violations-found, and operational error (e.g., invalid standards or missing Git)
