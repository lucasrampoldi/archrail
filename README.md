# Archrail

**AI can generate working software. Archrail makes sure it belongs in your architecture.**

Archrail is an open-source **architecture conformance engine** for AI-assisted development and platform engineering. You keep a catalog of *approved, authorized architectures* (frontend/backend blueprints). New and existing projects **adopt** one and Archrail:

1. **`archrail sync`** — renders the adopted architecture into agent-consumable context (`ARCHRAIL.md`) so coding agents design *within* your standards.
2. **`archrail check`** — deterministically validates the repository (or a diff) against the architecture and reports violations — as a local, pre-commit, or CI/PR gate.

Archrail is **not** a security policy engine (OPA/Kyverno/Checkov). Its domain is **architecture conformance**: layers, boundaries, approved stacks, service skeletons, dependency direction.

Design principles: deterministic where possible, usable without a SaaS backend, Git-friendly, human- and agent-readable, extensible, useful for brownfield repos, and independent of any specific AI agent.

## Install

```bash
go install github.com/archrail/archrail/cmd/archrail@latest
# or build a static binary
make build   # -> bin/archrail
make release # -> dist/ (linux/darwin/windows, amd64/arm64)
```

## Concepts

- **Architecture Profile** — a named, versioned, approved architecture: topology (components + layers) + rules (+ optional approved stack).
- **Catalog** — a shared Git repo/directory holding several profiles. In v0 it is consumed as a **local path** (clone / submodule / vendored); remote catalogs are a documented extension point.
- **Adoption** — a project pins a profile by `name` + `version` in `.archrail/archrail.yaml`.
- **Baseline** — records pre-existing violations so brownfield repos adopt without an immediate red build; only *new* violations fail the gate.

## Getting started

### 1. Author a catalog

```
catalog/
  profiles/
    react-node-hexagonal/
      1.0.0/
        profile.yaml
        standards/
          rules.yaml
```

`profile.yaml`:

```yaml
name: react-node-hexagonal
version: "1.0.0"
description: "Hexagonal Node backend with layered boundaries."
components:
  - name: handlers
    paths: ["src/handlers/**"]
  - name: services
    paths: ["src/services/**"]
  - name: repositories
    paths: ["src/repositories/**"]
layers:
  - name: presentation
    components: [handlers]
  - name: domain
    components: [services]
  - name: data
    components: [repositories]
extractors:
  - name: js
    extensions: [".ts", ".js"]
```

`standards/rules.yaml`:

```yaml
rules:
  - id: no-handler-db
    type: forbidden-import
    description: "HTTP handlers must not access repositories/DB directly."
    from: handlers
    to: repositories

  - id: layers
    type: layer-boundary
    description: "Respect layered dependencies."
    allow:
      - { from: presentation, to: [domain] }
      - { from: domain,       to: [data] }

  - id: approved-obs
    type: required-dependency
    description: "Apps must use the approved observability library."
    manifest: package.json
    dependency: "@company/observability"

  # Optional, opt-in, non-deterministic semantic check (LLM):
  - id: handler-delegates
    type: semantic
    severity: warn
    description: "Handlers delegate business logic to services."
    assertion: "Each HTTP handler delegates business logic to a service and contains no business rules itself."
    scope: handlers
```

### 2. Adopt a profile in your repo

```bash
archrail init --catalog ../catalog --profile react-node-hexagonal --profile-version 1.0.0
archrail profiles          # list what the catalog offers
```

### 3. Sync context for your agent

```bash
archrail sync              # writes ARCHRAIL.md (commit it)
```

### 4. Check conformance

```bash
archrail check                 # full repository
archrail check --base main     # only files changed vs main (brownfield/PR)
archrail check --format json    # machine-readable for CI
archrail check --update-baseline # record existing violations (brownfield onboarding)
```

## Rule types

Run `archrail rule-types` for the authoritative list. Built-in deterministic types:

| Type | Purpose |
|------|---------|
| `forbidden-import` | A component/layer must not import a target component/layer (or matching `pattern`). |
| `layer-boundary` | Imports must follow the declared `allow` graph between layers. |
| `forbidden-dependency` | A dependency must NOT appear in a manifest. |
| `required-dependency` | A dependency MUST appear in a manifest (approved-stack enforcement). |
| `required-files` | A component must contain the declared skeleton files. |
| `semantic` | Natural-language assertion evaluated by an LLM (opt-in, advisory by default). |

## Semantic (LLM) evaluation

The `semantic` rule type checks constraints deterministic rules can't express. It sends a **deterministic architecture fingerprint** (components, import graph, detected stack) plus curated **first-party** files (never dependencies/vendored) and the assertion to a Claude model via the **Vercel AI Gateway**.

- **Opt-in**: only runs when `semantic` rules exist.
- **Advisory by default** (`warn`); set `severity: error` to make a rule gate-failing.
- **Key handling**: the gateway key is read **only** from `AI_GATEWAY_API_KEY` and is never written to the repo, catalog, `ARCHRAIL.md`, logs, or reports.
- **Graceful degradation**: if the key/network/gateway is unavailable, semantic rules are *skipped with a note* (never pass/fail) and deterministic checking still completes. Use `--require-semantic` to make unavailability a hard error.
- **Model**: configure `semantic.model` in `.archrail/archrail.yaml` (default `anthropic/claude-sonnet-4-6`).

```bash
export AI_GATEWAY_API_KEY=...   # never commit this
archrail check
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success — no gate-failing violations. |
| `1` | Violations found — at least one non-baselined `error`. |
| `2` | Operational error — invalid standards, missing git/base ref, bad usage. |

`warn` findings (including advisory semantic findings) are reported but never fail the gate on their own.

## Integrations

See [`examples/`](examples/):

- **Pre-commit**: [`examples/pre-commit`](examples/pre-commit) — runs `archrail check --staged` before each commit.
- **CI/PR gate**: [`examples/github-actions.yml`](examples/github-actions.yml) — runs `archrail check --base <target>` and `archrail sync --check` on pull requests.

## License

See [LICENSE](LICENSE).
