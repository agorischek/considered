# Considered

Considered is a repository policy and decision documentation tool.

It collects metrics from one or more providers, evaluates them against
declared standards, and reports violations. A violation indicates that
something deserves attention.

Some violations are intentional. These are documented as **variances** — a
reviewed, documented departure from a standard. Variances are not
suppressions. They remain visible.

The goal of Considered is not merely to enforce limits. It is to make
architectural decisions explicit and to preserve the rationale behind them.
The configuration file is designed to double as lightweight architectural
documentation: a future engineer should be able to open it and understand
why unusual structures exist.

## Concepts

| Term | Meaning |
| --- | --- |
| **Metric** | A measurable property collected from a provider, e.g. `scc.code_lines`. Metrics are namespaced; the namespace identifies the provider. |
| **Subject** | The thing being measured. The MVP focuses on files, identified by their repo-relative path. |
| **Standard** | A policy boundary, e.g. `scc.code_lines <= 500`. Standards describe normal, repository-wide expectations. |
| **Violation** | A metric value that falls outside a standard. |
| **Warning** | A metric value that still passes, but is close to a standard or approved variance boundary. |
| **Variance** | A reviewed departure from a standard for a specific subject, with a required kind and reason. |
| **Kind** | A category for a variance. Built-in: `architectural`, `debt`, `generated`. Repositories may define more. |

### Evaluation

For each subject and metric:

1. Collect the metric value from its provider.
2. Apply the matching standard.
3. If the value is within the standard → **pass**. If it is near a configured
   boundary, also report a non-failing **warning**.
4. If it exceeds the standard, check for a variance on that subject and metric.
   - Within the variance's approved boundary → reported as a **variance** (passes).
     If it is near that approved boundary, also report a non-failing **warning**.
   - No variance, or value exceeds the approved boundary → reported as a **violation** (fails).

Variances target exact subjects — the MVP intentionally avoids globs, so every
variance corresponds to one concrete, reviewed file.

## Installation

Requires Go 1.26+. This repository vendors [`scc`](https://github.com/boyter/scc)
as a git submodule, so clone with submodules:

```sh
git clone --recurse-submodules <repo-url>
cd considered
go build -o bin/considered ./cmd/considered
go build -o bin/considered-scc ./cmd/considered-scc
```

Put both binaries on your `PATH`. The `considered` CLI invokes provider
binaries (such as `considered-scc`) by name, so they must be discoverable.

## Releases

GitHub Releases are published from version tags:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The release workflow builds `considered` and `considered-scc` together for
Linux, macOS, and Windows on amd64 and arm64. Each release includes platform
archives plus `checksums.txt`.

## Usage

### Initialize

```sh
considered init
```

Writes a starter `.considered.yaml` to the repository root. Fails if one
already exists.

### Check

```sh
considered check
```

Evaluates the repository against the configuration.

Exit codes:

- `0` — all checks pass
- `1` — one or more violations
- `2` — usage or configuration error

Machine-readable output:

```sh
considered check --json
considered check --sarif
```

In SARIF, violations are emitted as errors (fail CI), warnings as warnings,
and variances as informational findings (visible in code scanning, do not fail CI).

### Add a variance

```sh
considered variance add \
  --subject src/parser/grammar.ts \
  --metric scc.code_lines \
  --kind architectural \
  --reason "Grammar definitions are intentionally centralized."
```

Captures the subject's current metric value as the approved boundary and
writes the variance into the configuration. Flags left empty are prompted
for interactively. `--metric-reason` optionally records a rationale specific
to that single metric override. Adding a variance for a value that already
satisfies its standard is rejected.

### Version

```sh
considered version
```

### Common flags

- `--root <dir>` — repository root (default `.`)
- `--config <file>` — explicit config path (default `<root>/.considered.yaml`)

## Configuration

The configuration file is `.considered.yaml` at the repository root.

```yaml
kinds:
  - performance
  - compatibility

exclude:
  gitignored: true
  categories:
    - assets
    - generated
    - tests
    - vendored
    - dependencies
  paths:
    - src/generated/**

standards:
  scc.code_lines:
    max: 500
  scc.complexity:
    max: 50
  filesystem.bytes:
    max: 50000
  filesystem.longest_line:
    max: 140

warnings:
  percentBelowMax: 10
  percentAboveMin: 10

variances:
  src/parser/grammar.ts:
    kind: architectural
    reason: >
      Grammar definitions are intentionally centralized and easier to
      understand in a single file.
    metrics:
      scc.code_lines:
        max: 1274
      scc.complexity:
        max: 120
        reason: >
          Complexity is primarily driven by parser state generation.

  src/generated/schema.ts:
    kind: generated
    reason: >
      Generated source is committed for distribution.
    metrics:
      scc.code_lines:
        max: 4123
```

- **`kinds`** — additional variance kinds beyond the built-ins
  (`architectural`, `debt`, `generated`).
- **`exclude`** — files that no standard should measure. `gitignored`
  removes paths matched by Git ignore rules, `categories` applies semantic
  presets (`assets`, `generated`, `tests`, `vendored`, `dependencies`), and `paths`
  accepts custom doublestar globs such as `src/generated/**`. The `assets`
  category follows conventional static asset paths and file extensions,
  including `assets/**`, `public/**`, `static/**`, `branding/**`, images,
  fonts, audio/video, PDFs, design files, and SVGs. The `generated` category
  covers conventional machine-written outputs whose source of truth is
  elsewhere, including common lockfiles (`bun.lock`, `Cargo.lock`,
  `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, and peers), generated
  directories, codegen directories, `__generated__` directories, generated
  filename suffixes, and common protobuf outputs. The `tests` category follows
  default or conventional discovery patterns from common runners and build
  tools, including Jest, Vitest, Node, Bun, Deno, pytest, unittest, Go, Cargo,
  Maven Surefire, Gradle, Mocha, Cypress, Playwright, RSpec, and PHPUnit. Use
  exclusions for whole functional categories; use a `variance` to document a
  single reviewed departure. A file that is excluded is simply not measured.
- **`standards`** — boundaries keyed by metric. Each entry takes `min`,
  `max`, or both. Metric names must be namespaced.
- **`warnings`** — optional near-boundary reporting. `percentBelowMax: 10`
  warns when a passing value is at least 90% of a max boundary.
  `percentAboveMin: 10` warns when a passing value is at most 110% of a
  min boundary. Warnings do not fail checks, and apply to both standards and
  approved variance boundaries.
- **`variances`** — keyed by subject path. Each requires a `kind` (built-in
  or declared in `kinds`) and a `reason`, and may override one or more metric
  boundaries under `metrics`. A per-metric `reason` can document an
  individual override.

## Providers

Considered itself defines almost no metrics — they come from providers. The
policy engine is provider-agnostic. A provider is selected automatically from
the namespace of each metric referenced in `standards`.

All providers walk the repository the same way — honoring `.gitignore`,
`.ignore`, and submodule boundaries — so every metric is evaluated against the
same set of files.

**Built-in:**

- **`filesystem`** (in-process) — `filesystem.bytes`,
  `filesystem.longest_line` (longest line measured in Unicode characters).

**Bundled external:**

- **`scc`** — `scc.code_lines`, `scc.comment_lines`, `scc.complexity`,
  provided by the `considered-scc` binary.

**Custom providers** are external processes. For a metric namespaced `foo.*`,
Considered runs a binary named `considered-foo`. A provider is invoked as:

```sh
considered-foo --root <dir> --json
```

and must print a JSON document to stdout:

```json
{
  "metrics": [
    {
      "subject": "src/app/main.go",
      "provider": "foo",
      "values": { "foo.some_metric": 42 }
    }
  ]
}
```

`subject` is the repository-relative path; `values` maps namespaced metric
names to numbers. `provider` is optional and defaults to the namespace.
Providers may be written in any language — native Go plugins are not required.

## Project layout

```
cmd/considered/        the considered CLI (init, check, variance add)
cmd/considered-scc/    the bundled scc provider
internal/considered/   policy engine, config, providers, reporting
third_party/scc/       vendored scc (git submodule)
SPEC.md                full design specification
```

## Design principles

1. Standards should be easy to declare.
2. Violations should remain visible.
3. Variances require rationale and classification.
4. Variances remain visible.
5. Metrics are provider-agnostic.
6. Configuration should be human-readable and double as architectural documentation.
7. Exact subject matching is preferred over complex selector systems.
8. Repository knowledge should be preserved alongside policy.

See [SPEC.md](SPEC.md) for the complete specification.
