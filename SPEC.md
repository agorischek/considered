Considered
==========

Purpose
-------

Considered is a repository policy and decision documentation tool.

It collects metrics from one or more providers, evaluates them against
declared standards, and reports violations.

A violation indicates that something deserves attention.

Some violations are intentional. These are documented as variances.

A variance is a reviewed and documented departure from a standard.

Variances are not suppressions.

Variances remain visible.

The goal of Considered is not merely to enforce limits.

The goal is to make architectural decisions explicit and preserve the
rationale behind them.


Core Philosophy
---------------

Standards describe normal expectations.

Violations indicate departures from those expectations.

Departures are not automatically bad.

After review, a departure may be accepted as a variance.

Every variance must explain why it exists.

A future engineer should be able to open the configuration and
understand why unusual structures exist.

The configuration is intended to function as lightweight architectural
documentation.


Examples:

  A grammar file may intentionally exceed line-count standards.

  A generated file may intentionally exceed size standards.

  A legacy subsystem may temporarily exceed complexity standards.

All of these are variances.

The rationale explains why.


Core Concepts
-------------

Metric

A measurable property collected from a provider.

Examples:

  scc.code_lines
  scc.comment_lines
  scc.complexity

  filesystem.bytes
  filesystem.longest_line

Metrics are globally unique and namespaced.

The namespace identifies the provider.


Subject

The thing being measured.

Examples:

  file
  directory
  project
  repository

The MVP focuses primarily on files.


Standard

A policy boundary.

Examples:

  scc.code_lines <= 500

  scc.complexity <= 50

  coverage.percent >= 80

Standards describe normal expectations.


Violation

A metric value that falls outside a standard.


Variance

A reviewed departure from a standard.

Variances apply to specific subjects.

For the MVP, variances target exact subjects rather than globs.

A variance requires:

  kind
  reason

A variance may override one or more metric boundaries.


Rationale

Human-readable explanation for a variance.

Required.

Example:

  Grammar definitions are intentionally centralized and are easier to
  understand in a single file.


Kinds
-----

Variances are categorized.

Built-in kinds:

  architectural
  debt
  generated

Repositories may define additional kinds.

Examples:

  performance
  compatibility
  vendor
  protocol

Kinds help repositories develop a shared vocabulary around why
variances exist.


Metric Providers
----------------

Considered itself does not define most metrics.

Metrics come from providers.

Examples:

  SCC provider

    scc.code_lines
    scc.comment_lines
    scc.complexity

  Filesystem provider

    filesystem.bytes
    filesystem.longest_line

Future providers may include:

  ESLint
  TypeScript
  Roslyn
  Git
  Bundlers
  Security scanners

The policy engine is provider-agnostic.


Plugin Model
------------

Providers are external processes.

Examples:

  considered-scc

  considered-eslint

  considered-typescript

Providers communicate with Considered using a stable protocol.

Providers may be implemented in any language.

Native Go plugins are not required.


Configuration
-------------

Primary configuration file:

  .considered.yaml


Example:

  kinds:
    - performance
    - compatibility

  standards:
    scc.code_lines:
      max: 500

    scc.complexity:
      max: 50

    filesystem.bytes:
      max: 50000

  variances:
    src/parser/grammar.ts:
      kind: architectural

      reason: >
        Grammar definitions are intentionally centralized.

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


Standards
---------

Standards are organized by metric.

Examples:

  standards:
    scc.code_lines:
      max: 500

    coverage.percent:
      min: 80

Standards represent repository-wide expectations.


Variances
---------

Variances are organized by subject.

Examples:

  variances:
    src/parser/grammar.ts:
      ...

This structure intentionally makes unusual files easy to discover.

The variances section acts as a registry of architectural decisions.


Variance Evaluation
-------------------

Evaluation:

  collect metrics

  ->
  apply standard

  ->
  check for variance

  ->
  evaluate against variance boundary if present

Examples:

  Standard:

    scc.code_lines <= 500

  Variance:

    grammar.ts
    scc.code_lines <= 1274

Results:

  1274

    PASS

  1273

    PASS

  900

    PASS

  1275

    FAIL


Exact Subject Matching
----------------------

The MVP intentionally avoids globs.

Variances target exact subjects.

Example:

  variances:
    src/parser/grammar.ts:
      ...

Rationale:

  Every variance should correspond to a concrete reviewed subject.

  Reviewers should immediately understand which variance applies.

  Configuration should not require selector debugging.

  Configuration should not require selector tests.

Future versions may introduce broader selectors if needed.


CLI
---

Initialize repository:

  considered init

Creates:

  .considered.yaml


Evaluate repository:

  considered check

Exit codes:

  0 = all checks pass

  1 = one or more violations


Machine-readable output:

  considered check --json

  considered check --sarif


Create variance:

  considered variance add

Interactive workflow:

  select subject

  select metric

  capture current metric value

  choose kind

  enter rationale

Generated output:

  variance entry with approved metric boundary


Reporting
---------

Reports distinguish between:

  Violations

and

  Variances


Example:

  Violations

    src/compiler/compiler.ts

      scc.code_lines

      actual: 842

      standard: <= 500


  Variances

    src/parser/grammar.ts

      kind: architectural

      scc.code_lines

      actual: 1274

      approved: <= 1274

      reason:

        Grammar definitions are intentionally centralized.


SARIF
-----

Violations:

  emitted as errors

  fail CI


Variances:

  emitted as informational findings

  remain visible in code scanning

  do not fail CI


Design Principles
-----------------

1. Standards should be easy to declare.

2. Violations should remain visible.

3. Variances require rationale.

4. Variances require classification.

5. Variances remain visible.

6. Metrics are provider-agnostic.

7. Configuration should be human-readable.

8. Configuration should function as lightweight architectural
   documentation.

9. Exact subject matching is preferred over complex selector systems.

10. Repository knowledge should be preserved alongside policy.