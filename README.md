<img src="assets/logo.png" alt="Considered" height="72">

# Considered

Considered is a repository policy and decision documentation tool.

It collects metrics from one or more providers, evaluates them against declared
standards, and reports violations. Intentional departures are documented as
**variances**: reviewed, visible exceptions with a reason.

The goal is not merely to enforce limits. It is to make architectural decisions
explicit and preserve the rationale behind unusual structures.

## Quick Start

Requires Go 1.26+. This repository vendors [`scc`](https://github.com/boyter/scc)
as a git submodule, so clone with submodules:

```sh
git clone --recurse-submodules https://github.com/agorischek/considered.git
cd considered
go build -o bin/considered ./cmd/considered
go build -o bin/considered-scc ./cmd/considered-scc
```

Put both binaries on your `PATH`, then initialize and check a repository:

```sh
considered init
considered check
```

## Documentation

The full guide lives in the Astro docs site under [`docs/`](docs/):

```sh
cd docs
npm install
npm run dev
```

The docs cover configuration, variances, exclusions, providers, CI output, and
the CLI reference.

## Releases

GitHub Releases are published from version tags:

```sh
git tag v0.1.1
git push origin v0.1.1
```

The release workflow builds `considered` and `considered-scc` together for
Linux, macOS, and Windows on amd64 and arm64. Each release includes platform
archives plus `checksums.txt`.

## Design Principles

1. Standards should be easy to declare.
2. Violations should remain visible.
3. Variances require rationale and classification.
4. Variances remain visible.
5. Metrics are provider-agnostic.
6. Configuration should be human-readable and double as architectural documentation.
7. Exact subject matching is preferred over complex selector systems.
8. Repository knowledge should be preserved alongside policy.
