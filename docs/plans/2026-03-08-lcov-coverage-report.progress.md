# LCOV Coverage Report — Execution Progress

**Plan file:** `docs/plans/2026-03-08-lcov-coverage-report.md`
**Started:** 2026-03-08

## How to resume in a new session

Open a new Claude Code session in `/home/drodriguez/workspace/falco` and say:

> "Resume the LCOV coverage report plan from `docs/plans/2026-03-08-lcov-coverage-report.progress.md`. Execute remaining tasks unattended."

## Tasks

| # | Description | Status | Commit |
|---|-------------|--------|--------|
| 1 | Fix `"brancn"` typo in `cmd/falco/table.go` | pending | — |
| 2 | Write failing tests in `tester/shared/lcov_test.go` | pending | — |
| 3 | Implement `WriteLCOV` in `tester/shared/lcov.go` | pending | — |
| 4 | Wire `--coverage-out` in `cmd/falco/lcov.go` + `main.go` | pending | — |

## Completion criteria

All tasks complete when `make test` passes and:
```
./falco test --coverage --coverage-out lcov.info -I . <vcl-file>
```
produces a valid `lcov.info` file.
