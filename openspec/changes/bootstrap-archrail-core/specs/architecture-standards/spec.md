## ADDED Requirements

### Requirement: On-disk standards layout

Architecture standards (topology + rules) SHALL be authored as YAML within an **architecture profile** definition (see the architecture-profiles capability). A profile definition MUST contain a topology/config file and MAY contain one or more standards files (`.yaml`/`.yml`) whose rules SHALL be loaded and merged into one active rule set for that profile. The adopting project SHALL hold a `.archrail/` directory at its repository root containing `archrail.yaml`, which declares the adopted profile and MAY declare local topology mappings and overrides. Archrail SHALL discover the project `.archrail/` directory by walking up to the repository root.

#### Scenario: Project directory is discovered from a subdirectory

- **WHEN** `archrail` is invoked from any subdirectory of a repository that contains `.archrail/archrail.yaml`
- **THEN** Archrail SHALL locate the `.archrail/` directory by walking up to the repository root and load it

#### Scenario: Missing project directory

- **WHEN** `archrail` is invoked in a repository that has no `.archrail/` directory
- **THEN** Archrail SHALL exit with a non-zero code and a message instructing the user to run initialization

#### Scenario: Multiple standards files in a profile are merged

- **WHEN** a profile definition contains more than one standards `.yaml` file each declaring rules
- **THEN** Archrail SHALL load rules from every file into a single active rule set for that profile, preserving the originating file for reporting

### Requirement: Config file schema

The `archrail.yaml` config file SHALL declare a schema `version` and MAY declare: named **components** (logical modules identified by path globs), named **layers** (ordered or graph-based groupings of components), default rule **severity**, and language-specific **import-extractor** selections. Archrail SHALL validate the config against a known schema and reject unknown top-level keys with a clear error identifying the offending key and file.

#### Scenario: Valid config loads

- **WHEN** `archrail.yaml` declares a supported `version`, components, and layers using valid syntax
- **THEN** Archrail SHALL load the config without error and expose the components and layers to the rule engine

#### Scenario: Invalid config is rejected with location

- **WHEN** `archrail.yaml` contains an unknown top-level key or a malformed value
- **THEN** Archrail SHALL exit non-zero and report the file path, the offending key, and the expected schema

#### Scenario: Unsupported schema version

- **WHEN** `archrail.yaml` declares a `version` newer than the running CLI supports
- **THEN** Archrail SHALL refuse to proceed and instruct the user to upgrade the CLI

### Requirement: Machine- and human-readable rule declaration

Each rule SHALL be declared in YAML with at least: a stable `id` (unique across the active rule set), a `type` naming a registered rule type, a human-readable `description`, an optional `severity` (`error` or `warn`, defaulting to the config default), and a `type`-specific parameter block. Rule declarations MUST be diff-friendly (one rule is a self-contained block) and MUST be usable both as agent-readable context and as machine-evaluable input.

#### Scenario: Well-formed rule is accepted

- **WHEN** a standards file declares a rule with a unique `id`, a registered `type`, a `description`, and valid parameters
- **THEN** Archrail SHALL register the rule as active and evaluable

#### Scenario: Duplicate rule id is rejected

- **WHEN** two rules across the active rule set share the same `id`
- **THEN** Archrail SHALL exit non-zero and report both source files and the conflicting `id`

#### Scenario: Unknown rule type is rejected

- **WHEN** a rule declares a `type` that is not present in the rule-type registry
- **THEN** Archrail SHALL exit non-zero and report the rule `id`, the unknown `type`, and the list of known types

### Requirement: Deterministic built-in rule types

Archrail SHALL ship a set of deterministic, built-in rule types that require no network access and no language model. v0 MUST include at least: `forbidden-import` (a component/layer MUST NOT import from a named target), `layer-boundary` (imports MUST follow the declared allowed direction between layers), `forbidden-dependency` and `required-dependency` (a declared dependency MUST NOT / MUST appear in a named manifest), and `required-files` (paths matching a component skeleton MUST contain the declared files). Given identical inputs, every built-in rule SHALL produce identical results.

#### Scenario: Deterministic evaluation

- **WHEN** the same repository state is evaluated twice with the same standards
- **THEN** Archrail SHALL produce byte-identical violation sets

#### Scenario: Forbidden import is detected

- **WHEN** a file belonging to a component with a `forbidden-import` rule imports from the forbidden target
- **THEN** the rule SHALL produce a violation identifying the file, the import, and the rule `id`

#### Scenario: Required skeleton file is missing

- **WHEN** a path matches a component governed by a `required-files` rule but a declared required file is absent
- **THEN** the rule SHALL produce a violation identifying the component and the missing file

### Requirement: Import extraction is pluggable per language

Import/dependency detection for import-based rule types SHALL be performed by named **import extractors** selected per file by extension or config. Archrail MUST ship at least one built-in extractor and MUST allow the active extractor set to be selected in config. Extraction MUST be deterministic and MUST NOT execute repository code.

#### Scenario: Extractor selected by extension

- **WHEN** a file matching a configured extractor's extension is evaluated by an import-based rule
- **THEN** Archrail SHALL use that extractor to derive the file's imports without executing the file

#### Scenario: File with no available extractor

- **WHEN** an import-based rule targets a file whose extension has no configured extractor
- **THEN** Archrail SHALL skip that file for that rule and record a reported "skipped: no extractor" note rather than a violation

### Requirement: Extensible rule-type registry

The loading pipeline SHALL resolve rule `type`s through a **registry** abstraction. The registry MUST be defined such that additional rule types can be added without changing rule-declaration syntax already in use. (Resolution of the profile catalog itself is governed by the catalog **source** abstraction in the architecture-profiles capability.)

#### Scenario: Registry lists available rule types

- **WHEN** a user requests the list of available rule types
- **THEN** Archrail SHALL enumerate every type registered in the rule-type registry with its required parameters
