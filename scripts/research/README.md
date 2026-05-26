# Research Benchmark Scripts

This directory contains a reproducible benchmark package for the RPZ research
section.

It generates:

- PostgreSQL `EXPLAIN ANALYZE` measurements with and without secondary indexes;
- HTTP load measurements for product lookup with Redis disabled and enabled;
- CSV summaries, Markdown tables, SVG/PNG charts, and an HTML report screenshot.

Run from the repository root:

```bash
./scripts/research/run.sh
```

Default output directory:

```text
DBCourseWork/RPZ/research_results/latest/
```

The runner creates an isolated PostgreSQL database named
`marketplace_research_<timestamp>` by default. Override it only for a disposable
database:

```bash
RESEARCH_DB=marketplace_research ./scripts/research/run.sh
```

