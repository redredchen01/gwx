# CHANGELOG

All notable changes to GWX are documented in this file.

## [0.25.2] - 2026-03-31

### Added

- **Skill Scaffolding** (`gwx skill new` command)
  - Auto-generate valid YAML templates without manual writing
  - Supports `-d` (description), `-t` (tools), `-o` (output path), `--dry-run` flags
  - Built-in validation ensures generated YAML parses correctly

- **Shared Test Infrastructure** (`internal/testutil` package)
  - `MockCaller` — fluent API for mocking tool calls in tests
  - `SkillBuilder` — programmatic skill construction with support for sequential, parallel, and DAG workflows
  - Filesystem isolation helpers (`SetSkillConfigHome`, `TempSkillFile`)
  - 12 comprehensive tests covering all patterns

- **Complete Documentation**
  - `docs/skill-dsl.md` — Full YAML schema reference with tools, execution models, variables, error handling, and examples (300+ lines)
  - `docs/dag-workflows.md` — DAG patterns, fan-out/fan-in, filter chains, multi-stage pipelines, testing, and debugging (400+ lines)
  - `docs/custom-tools.md` — Shell and HTTP tool usage, security best practices, troubleshooting (400+ lines)
  - Updated `skills/README.md` with quick-start guide, Skill Marketplace v2 reference, and documentation index

### Changed

- Enhanced skill execution engine to support test infrastructure
- Improved API rate limiting and request handling
- Updated internal skill types to support test builders

### Tests

- Added 18 new tests for scaffold command, test utilities, and filesystem isolation (100% pass rate)
- All tests run in CI/CD pipeline before merge

---

## [0.25.1] - Previous Release

(See git history for details on earlier versions)
