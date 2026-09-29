## ADDED Requirements

### Requirement: Architecture profile definition

An **architecture profile** SHALL be a named, versioned, self-contained definition of an approved architecture. A profile MUST declare at least: a stable `name`, a `version`, a human-readable `description`, and the topology + rules (per the architecture-standards capability) that compose it. A profile MAY declare an approved technology stack (e.g., required/forbidden frameworks or dependencies for frontend and backend components). Profiles MUST be human-readable and Git-friendly.

#### Scenario: Well-formed profile is loaded

- **WHEN** a profile declares a `name`, `version`, `description`, and valid topology + rules
- **THEN** Archrail SHALL load it as an available approved architecture

#### Scenario: Profile missing required identity is rejected

- **WHEN** a profile omits its `name` or `version`
- **THEN** Archrail SHALL exit non-zero and report which field is missing and in which profile source

### Requirement: Profile catalog with multiple approved architectures

Archrail SHALL support a **catalog** that contains more than one approved profile so an organization can maintain a list of authorized architectures. Profiles in the catalog MUST be uniquely identified by `name` (and distinguishable by `version`). Archrail SHALL be able to enumerate the profiles available in the catalog.

#### Scenario: Catalog exposes multiple profiles

- **WHEN** a catalog contains several profiles (e.g., `react-node-hexagonal` and `nextjs-fastapi-layered`)
- **THEN** Archrail SHALL list every available profile with its name, version, and description

#### Scenario: Duplicate profile name+version in catalog is rejected

- **WHEN** two profile definitions in the catalog share the same `name` and `version`
- **THEN** Archrail SHALL exit non-zero and report the conflict and both sources

### Requirement: Catalog source abstraction

Profiles SHALL be resolved through a **catalog source** abstraction. v0 MUST implement a local-path source that reads the catalog from a directory on disk (a shared Git catalog consumed as a clone, submodule, or vendored copy). The source abstraction MUST be defined so that additional sources (e.g., remote/URL-based catalogs) can be added later without changing the profile format or the project's adoption declaration. Evaluation MUST NOT require network access in v0.

#### Scenario: Local-path catalog resolves

- **WHEN** the project config points at a local catalog path and requests a profile by name+version
- **THEN** Archrail SHALL resolve the profile from that path without network access

#### Scenario: Catalog path is missing or unreadable

- **WHEN** the configured catalog path does not exist or cannot be read
- **THEN** Archrail SHALL exit non-zero with a clear message identifying the configured path

### Requirement: Project adopts a profile

A project SHALL declare, in its `.archrail/` config, which profile it adopts by `name` and `version`, together with the catalog source to resolve it from. `archrail sync` and `archrail check` SHALL operate against the adopted profile. If no profile is declared, Archrail SHALL exit with guidance rather than silently checking nothing.

#### Scenario: Project resolves its adopted profile

- **WHEN** a project config declares an adopted profile name+version and a resolvable catalog source
- **THEN** `archrail check` and `archrail sync` SHALL load that profile as the active architecture

#### Scenario: Adopted profile not found in catalog

- **WHEN** the declared profile name+version is not present in the resolved catalog
- **THEN** Archrail SHALL exit non-zero and list the profiles that are available

#### Scenario: No profile declared

- **WHEN** a project config declares no adopted profile
- **THEN** Archrail SHALL exit non-zero with guidance on how to adopt one, and SHALL NOT report a passing check

### Requirement: Profile version resolution is explicit and deterministic

Profile adoption SHALL pin an explicit `version`. Given the same catalog contents and the same pinned name+version, resolution MUST be deterministic and reproducible. Archrail MUST NOT auto-upgrade a project to a newer profile version without an explicit change to the project's declaration.

#### Scenario: Pinned version is honored

- **WHEN** a project pins a specific profile version and the catalog contains that version
- **THEN** Archrail SHALL resolve exactly that version regardless of newer versions present in the catalog

#### Scenario: Pinned version absent

- **WHEN** a project pins a profile version that is not present in the catalog
- **THEN** Archrail SHALL exit non-zero and list the available versions for that profile name
