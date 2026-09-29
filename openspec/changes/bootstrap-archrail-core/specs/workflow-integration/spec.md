## ADDED Requirements

### Requirement: Single CLI entrypoint

Archrail SHALL be distributed as a single command-line executable `archrail` exposing at minimum the subcommands `init`, `sync`, `check`, and a `profiles` command to list the profiles available in the resolved catalog. Running `archrail` with no arguments or `--help` SHALL print usage. The CLI SHALL be runnable fully offline and without any SaaS backend.

#### Scenario: Help is shown

- **WHEN** `archrail --help` is run
- **THEN** it SHALL list the available subcommands and global options and exit zero

#### Scenario: Offline operation

- **WHEN** `archrail check` or `archrail sync` runs with no network available
- **THEN** the command SHALL complete normally using only local standards

### Requirement: Initialization scaffolds standards

`archrail init` SHALL scaffold a starter `.archrail/` directory (a valid `archrail.yaml`) in a repository that does not yet have one, so a team can adopt Archrail quickly. The scaffolded config SHALL declare a catalog source and an adopted profile (name + version) — using a resolvable value when a catalog is provided, or a clearly marked placeholder otherwise. It SHALL NOT overwrite an existing `.archrail/` directory without explicit confirmation.

#### Scenario: Init in a fresh repository

- **WHEN** `archrail init` runs in a repository with no `.archrail/` directory
- **THEN** it SHALL create a valid `.archrail/` directory that passes validation and declares an adopted profile and catalog source

#### Scenario: Init refuses to clobber

- **WHEN** `archrail init` runs where `.archrail/` already exists and no override is given
- **THEN** it SHALL not modify existing files and SHALL exit with a message explaining how to override

### Requirement: Pre-commit integration

Archrail SHALL provide a documented pre-commit integration that runs `archrail check` (diff-scoped to staged changes) so violations are caught before a commit is created. The integration MUST rely only on the CLI's documented exit codes.

#### Scenario: Pre-commit blocks a violating commit

- **WHEN** the pre-commit hook runs `archrail check` on staged changes that introduce a non-baselined `error` violation
- **THEN** the hook SHALL fail and prevent the commit

#### Scenario: Pre-commit allows a clean commit

- **WHEN** the pre-commit hook runs on staged changes with no non-baselined `error` violation
- **THEN** the hook SHALL pass and allow the commit

### Requirement: CI / PR architecture gate

Archrail SHALL provide a documented CI/PR gate recipe that runs `archrail check --base <target-branch>` on pull requests and fails the build on non-baselined `error` violations, consuming the JSON report for programmatic use where needed. The recipe MUST work with the standard CLI and require no Archrail-hosted service.

#### Scenario: CI gate fails on new violation

- **WHEN** the CI gate runs `archrail check --base <target>` on a PR that introduces a non-baselined `error` violation
- **THEN** the CI job SHALL fail with a non-zero exit and surface the violation report

#### Scenario: CI gate enforces sync freshness

- **WHEN** the CI gate additionally runs `archrail sync --check` and `ARCHRAIL.md` is stale relative to the standards
- **THEN** the CI job SHALL fail and report that the generated context must be regenerated
