# AGENTS.md — arcana-cloud-go

Instructions for any AI coding agent (Claude Code, Codex, Cursor, Copilot, Gemini…) working
in this repository **or in any product created from it**. `CLAUDE.md`, `GEMINI.md` and
`.github/copilot-instructions.md` all point here; this file is the single source.

> **You MUST NOT report a task as done until `tools/arch-check.sh` exits 0 and the CI test
> lane (§4) is green.** Breaking the layering costs a red build: the pre-commit hook
> (`tools/githooks/pre-commit`) rejects new Go files outside the layer directories and impl
> imports outside `internal/di/`, and Jenkins stage "Architecture Qube" fails the merge below
> score 90.

## 0. First command in a fresh clone

Run `tools/install-hooks.sh` once. Git does not install hooks on clone; until you run it the
pre-commit check does not exist and only CI catches a flat layout — after the work is
written. `tools/new-app.sh` runs it for you.

## 1. Starting something new

| Situation | Do this | Never do this |
|---|---|---|
| New service product | `tools/new-app.sh <name> <dest-dir>` (optional `MODULE=<go module path>`) — copies the whole repo, renames the module path and every import, binary, Docker/compose/k8s names, Jenkins `APP_NAME`, Sonar key and per-host CI names, installs the hook | `go mod init` and copy a few packages or files into it |
| New business feature (entity + CRUD API) | `tools/new-feature.sh <Name>` — one vertical slice: entity, DAO interface + GORM impl, repository interface + impl, request/response DTOs, service interface + impl, Gin controller, tests for each, fx providers, route and AutoMigrate wiring | Write a handler that calls GORM, or a new `foo.go` in `internal/` |
| Feature over gRPC / layered deployment / MongoDB | After `new-feature.sh`, mirror the User feature by hand: `api/proto/user_service.proto`, `internal/controller/grpc/service/user_service.go`, `internal/domain/repository/grpc/`, `internal/domain/dao/impl/mongo/` (+ `document/`, `mapper/`) | Call a DAO from a gRPC service |

Copying parts instead of `new-app.sh` loses this file, the hook and the CI gate, so nothing
tells the next agent the rules exist. `new-app.sh` has no `--no-demo`: Auth/User, Plugin, SSR
and Jobs are platform features wired through `internal/di/`, not demo modules.

## 2. Layers — where a file goes

| Layer | Path | Contains | May depend on |
|---|---|---|---|
| Entry + DI | `cmd/server/main.go`, `cmd/worker/main.go`, `internal/di/*.go` | startup, Uber fx modules (`DAOModule`, `RepositoryModule`, `ServiceModule`, `ControllerModule`, …), routes in `server.go`, AutoMigrate in `database.go` | everything; the only place that imports an `impl` package |
| HTTP boundary | `internal/controller/http/<x>_controller.go` | Gin handlers: bind → call service interface → wrap `response.ApiResponse` | service interfaces, `internal/dto`, `internal/middleware`, `internal/security` |
| gRPC / GraphQL boundary | `internal/controller/grpc/{service,repository}/`, `internal/controller/graphql/`; messages in `api/proto/` → `api/proto/pb` (generated) | gRPC servers per layer, GraphQL resolver | service / repository interfaces, `pb`, dto |
| DTO | `internal/dto/request/<x>.go`, `internal/dto/response/<x>.go` | API input/output shapes, `ApiResponse`, `PagedResponse` | nothing in `internal/domain` |
| Service | interface `internal/domain/service/<x>_service.go`; impl `internal/domain/service/impl/<x>_service_impl.go` | business rules, validation, entity → response DTO mapping (private `toXResponse`) | repository interfaces, entity, dto, security, config |
| Repository | interface `internal/domain/repository/<x>_repository.go`; impl `.../repository/impl/<x>_repository_impl.go`; gRPC-backed `.../repository/grpc/` | domain-level data access, delegates to DAOs | DAO interfaces, entity |
| DAO | interface `internal/domain/dao/<x>_dao.go` (embeds `BaseDAO[T, uint]`); impls `.../dao/impl/gorm/<x>_dao_impl.go` (MySQL/PostgreSQL) and `.../dao/impl/mongo/` | one data source per impl; the only code that runs queries | entity, `gorm` / mongo driver |
| Entity | `internal/domain/entity/<x>.go` | GORM-tagged domain structs | nothing in `internal/` |
| Cross-cutting | `internal/{config,configserver,middleware,security,resilience,observability,jobs,plugin,websocket,grpc/client}`, `pkg/{errors,logger}` | config, JWT/RBAC, circuit breaker, metrics/tracing, job queue, plugins | config, entity, dto |

There is no separate use-case/interactor layer and no mapper package for DTOs: services map
entities to response DTOs themselves. Do not invent either. Direction is one way:
controller → service → repository → DAO → entity. A lower layer never imports a higher one.
Worked example of every layer: the Plugin feature — `plugin_controller.go` →
`PluginService` / `impl/plugin_service_impl.go` → `PluginRepository` /
`impl/plugin_repository_impl.go` → `PluginDAO` / `impl/gorm/plugin_dao_impl.go`, wired in
`internal/di/{controller,service,repository,dao}.go`.

## 3. Rules

| # | Rule | Enforced by |
|---|---|---|
| R1 | Every service, repository and DAO is an exported interface (`XService`, `XRepository`, `XDAO`) with an unexported struct in an `impl` package, built by `NewX(...) Interface`. | arch-qube "Interface/Impl Colocation", "Implementation Naming Convention" |
| R2 | Only `internal/di/` imports an `impl` package (`internal/domain/*/impl`, `impl/gorm`, `impl/mongo`). Everyone else holds the interface. | arch-qube "Only DI Container Imports Impl"; pre-commit hook |
| R3 | Controllers call services only; services call repositories only; only DAOs touch `*gorm.DB` / mongo collections; transactions belong in the service layer. | arch-qube "Backend Layer Chain", "Controller Cannot Use DAO", "Service Cannot Access DB Directly", "Transaction at Service Layer Only", "Repository Cannot Call Service", "Layer Direction Enforcement" |
| R4 | A new `.go` file goes in a layer directory of §2 — never `internal/<x>.go`, `internal/domain/<x>.go`, the repo root, or a new top-level `internal/` package. | pre-commit hook |
| R5 | A handler only binds, delegates and wraps: ≤ 20 lines, no business logic. | arch-qube "No Business Logic in Boundary" |
| R6 | APIs take `internal/dto/request` and return `internal/dto/response` types, never `entity` structs. | arch-qube "DTO/Entity Separation" |
| R7 | Every new DAO, repository, service and controller has a `_test.go` next to it that runs without an external database (SQLite in-memory for GORM DAOs, testify mocks or in-memory fakes above). | arch-qube "Test Coverage Threshold"; SonarQube quality gate (Jenkins stage "SonarQube Analysis") |
| R8 | A new entity is added to `runMigrations` in `internal/di/database.go`; a MongoDB DAO also gets indexes in `createMongoIndexes`. | `new-feature.sh`; MongoDB part on you |
| R9 | Generated code in `api/proto/pb` is never edited or committed; change the `.proto` and regenerate. | `.gitignore`; CI regenerates it (`Dockerfile.ci`) |

## 4. Before you say "done"

Run these, in this order, and paste the last line of each into your summary:

```bash
tools/arch-check.sh            # same image, --framework go, --threshold 90 as CI; must print "arch-check: PASS"
mkdir -p api/proto/pb && protoc --go_out=api/proto/pb --go_opt=paths=source_relative \
  --go-grpc_out=api/proto/pb --go-grpc_opt=paths=source_relative -I api/proto api/proto/*.proto
go vet ./...                   # catches what the CI build would
go test -tags '!integration' ./... -coverprofile=coverage.out -covermode=atomic   # what Dockerfile.ci runs
```

The CI lane is Jenkins stage "Docker Compose Build" (`docker compose -f docker-compose.ci.yml
build`, which runs the test command above in `Dockerfile.ci`). Without a local toolchain run
that same build. It pipes `go test` through `tee`, so a red test does not fail the image
build — read the `FAIL` lines yourself; a passing build is not proof the tests passed.

## 5. Architecture check (arch-qube)

arch-qube (https://github.com/jrjohn/arcana-arch-qube; usage skill
https://github.com/jrjohn/arcana-skills/tree/main/arch-qube-skill) scores the layering
rules of §3 statically. The version is pinned in `tools/arch-qube.version`; both
`tools/arch-check.sh` and the Jenkinsfile read it.

`tools/arch-qube.version` is bumped automatically by Renovate when arch-qube releases; do not edit the version anywhere else.

- Run: `tools/arch-check.sh` — Docker image `ghcr.io/jrjohn/arch-qube:<version>` (public,
  multi-arch), then `arcana.boo/arcana/arch-qube:<version>`; no `:latest` fallback. Same
  `--framework go`, `--threshold 90` and exclusions as the Jenkins stage.
- No Docker: `pipx run --spec "git+https://github.com/jrjohn/arcana-arch-qube.git@v$(cat tools/arch-qube.version)"
  arch-qube scan . --framework go --no-ai --ci --format json,markdown -o arch-qube-reports --threshold 90`.
- Exit 0 = pass, 1 = fail, 2 = could not check (no Docker / no image). Exit 2 is NOT a pass.
- Report: `arch-qube-reports/arch-qube.md` and `arch-qube.json`.
- PASS = score ≥ threshold **and** zero violations in every CRITICAL rule — a 99 can still FAIL.
- `--ci` prints nothing on failure. Rerun without `--ci` (or open `arch-qube.md`) to see why.

Reading the report: the score alone is not enough. Read every FAIL row and its violations.
A rule under "Not evaluated" checked nothing — NOT EVALUATED is not a pass for a layer you touched.
Check `files_checked` in the JSON: a number far below the Go file count means the scan saw
an empty or wrong tree. An unlayered project scores 0 on "Layer Direction Enforcement".

False positive: open an issue on https://github.com/jrjohn/arcana-arch-qube/issues with the
report attached. NEVER lower `--threshold`, add exclusions, or move code to dodge a rule.

### What enforces the check locally (there is no CI server)

| Trigger | Mechanism | Blocks when |
|---|---|---|
| Claude Code finishes a turn | `.claude/settings.json` Stop hook → `tools/claude-stop-arch-check.sh` | source under `cmd/`, `internal/`, `pkg/`, `api/`, `tests/` changed and `tools/arch-check.sh` fails (gives up after 3 blocks, saying the work is NOT done) |
| `git push` | `tools/githooks/pre-push` | pushed commits touch `cmd/`, `internal/`, `pkg/`, `api/`, `tests/` and the check fails or cannot run |
| `git commit` | `tools/githooks/pre-commit` | fast layering checks (no Docker) |

Claude Code asks once whether to trust this project's settings: answer yes, or the Stop hook never runs.
Git hooks run only after `tools/install-hooks.sh`.

## 6. Reference

- Optional detail: `arcana-go-developer-skill`
  (https://github.com/jrjohn/arcana-skills/tree/main/arcana-go-developer-skill).
  This file is the source of truth: if the skill and this file disagree, this file wins.
