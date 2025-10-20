## pctl Build + Deploy with Compose `build:` Support

### Goal
- Allow `pctl deploy` to work seamlessly with Docker Compose files that include `build:` directives.
- No dependency on public registries; images are made available on the remote Docker engine managed by Portainer.

### Operating Modes
- **build.mode = remote-build (default)**: Tar each service build context and call Portainer’s Docker proxy Build API to build on the target engine. Minimal bandwidth from local to remote.
- **build.mode = load**: Build locally, export an image tar, and stream it to the remote engine via Portainer’s Docker proxy Images Load API. Useful when the remote has poor internet egress to pull base layers. Requires local Buildx for multi-arch.

### User Experience
- `pctl deploy` detects `build:` directives and performs builds automatically according to `build.mode`.
- Logs stream during builds with clear per-service prefixes and status.
- Deterministic image tags are generated; `build:` is replaced with `image:` in a generated compose that is sent to Portainer for stack create/update.
- Optional flags override config (e.g., `--build-mode`, `--no-cache`, `--build-arg KEY=VAL`, `--parallel N`).
- `--force-build` forces rebuild even if content hash indicates no change.

### High-Level Flow (applies to both modes)
1. Load `pctl.yml` and the compose file specified by `compose_file`.
2. Parse compose to find services with `build:` (string or object). Resolve:
   - context directory
   - dockerfile path (relative to context; default `Dockerfile`)
   - build args (map), target (stage), cache-from (if provided)
3. Determine remote engine platform and CPU count (via Docker proxy `/api/endpoints/{envId}/docker/info` or `/docker/version`). Use CPU count to derive default parallelism.
4. For each service with `build:`:
   - Create a streaming tar of the context (honor `.dockerignore`).
   - Generate a deterministic image tag: `pctl-<stack>-<service>:<content-hash|shortsha|timestamp>`.
   - Perform build according to mode (see below) and stream logs to the user.
5. When all builds succeed, transform compose in-memory: remove `build:` and set `image:` to the generated tag for each service.
6. Create or update the stack in Portainer using the transformed compose content (existing `CreateStack`/`UpdateStack` behavior). Do not pull images during update (they already exist on the engine).

### Remote Build Mode (default)
- API: `POST /api/endpoints/{envId}/docker/build`
- Query params: `t=<tag>`, `dockerfile=<rel-path>`, `buildargs=<json>`, `target=<stage>` (as applicable).
- Headers: `Content-Type: application/x-tar`.
- Private base image auth is out of scope for v1 (ignore for now).
- Benefits: minimal bytes from local to remote; native arch build; remote cache reuse across runs.

### Load Mode (local build + remote load)
- Build locally (using Buildx) targeting the remote platform.
- Export an image tar stream and upload to the remote engine:
  - API: `POST /api/endpoints/{envId}/docker/images/load`
  - Header: `Content-Type: application/x-tar`
- Benefits: avoids remote base image pulls; suitable for air‑gapped/locked‑down remotes.

### Configuration Additions (`pctl.yml`)
```yaml
build:
  mode: remote-build        # remote-build | load (default: remote-build)
  no_cache: false           # pass through to builds
  parallel: auto            # concurrent builds; auto derives from remote CPU
  tag_format: "pctl-{{stack}}-{{service}}:{{hash}}"  # supports {{stack}}, {{service}}, {{timestamp}}, {{hash}}
  platforms: ["linux/amd64"]  # used for load mode local builds
  extra_build_args: {}      # optional global overrides merged on top of compose build.args
  force_build: false        # force rebuild even if unchanged
  warn_threshold_mb: 50     # WARN if tar/image stream exceeds this size
```

### Compose Parsing & Transformation
- Parse YAML; for each service with `build:` support both forms:
  - `build: ./dir`
  - `build: { context: ./dir, dockerfile: Dockerfile, args: {...}, target: stage }`
- After successful build, produce a transformed compose (in-memory) where each service has `image: <generated-tag>` and no `build:`.
- Keep all other service attributes unchanged.

### Context Tarring
- Streamed tar to avoid high memory usage.
- Honor `.dockerignore` patterns to minimize transfer.
- Preserve file permissions; normalize path separators.
- Emit a WARN if the generated tar exceeds `warn_threshold_mb` (default 50 MB).

### Caching & Idempotency
- Compute a content hash for each context (e.g., hash of included files) to produce deterministic tags and enable fast skip logic:
  - Optional optimization: probe remote `GET /api/endpoints/{envId}/docker/images/<tag>/json`; if present and `--no-cache` not set, skip rebuild.
- For remote builds, the engine’s own build cache will be reused automatically when tags and layers match.

### Tagging Strategy (decision)
- Two common strategies:
  - Content-hash: tag includes a hash derived from the build context contents. Pros: deterministic, enables cache hits and build skipping. Cons: tag changes only when content changes.
  - Timestamp: tag includes a time-based suffix. Pros: always unique, easy to force redeploy. Cons: defeats cache-based skipping; more bandwidth over time.
- Recommendation: default to content-hash via `{{hash}}` in `tag_format`. Provide `--force-build` (or `build.force_build`) to rebuild even when unchanged, and allow a `--tag-format` override if timestamp-based tags are desired.

### Authentication for Private Base Images (remote-build)
- Out of scope for v1; ignore private base image auth.

### CLI Additions
- `pctl deploy --build-mode [remote-build|load]`
- `pctl deploy --no-cache`
- `pctl deploy --build-arg KEY=VAL` (merged into compose `build.args`, wins on conflict)
- `pctl deploy --parallel N`
- `pctl deploy --dry-run` (parses and reports planned builds and resulting image tags without executing)
- `pctl deploy --force-build`

### Error Handling & UX
- Surface build output lines with per-service prefixes and concise status summaries.
- Detect and explain common failures: missing Dockerfile, context path invalid, base image pull/auth failures, large context timeouts.
- Ensure partial failures leave a clear next action (e.g., `pctl deploy --retry serviceA`).

### Backward Compatibility
- If no `build:` directives are present, behavior is unchanged.
- Existing `redeploy` updates continue to function; images are not pulled during update unless explicitly requested.

### Security Considerations
- Ensure `.dockerignore` prevents accidental inclusion of secrets and large artifacts.
- Do not echo build args that look secret; redact values based on key patterns (e.g., `*KEY*`, `*SECRET*`, `*TOKEN*`).
- TLS handling remains governed by existing `skip_tls_verify`.

### Performance Considerations
- Remote-build reduces local→remote bandwidth; encourage small contexts and multi‑stage builds.
- Parallelize builds up to configured `parallel` with fairness; guard against overwhelming the engine.
- Stream I/O everywhere; avoid buffering entire tars in memory.
- Default parallelism policy when `parallel=auto`: derive from remote CPU count, e.g., `max(1, min(NCPU-1, 4))`.

### Testing Strategy
- Unit tests: compose parsing, context hashing, tag generation, transformation logic.
- Integration tests: mock Portainer client (build, images/load, info); validate end‑to‑end flow.
- E2E (optional): against a dev Portainer environment with simple sample services.

### Documentation Updates
- README: describe `build:` support, modes, and config additions; add examples and best practices for `.dockerignore`.
- `pctl.yml.example`: include the new `build` section with commented defaults.

### Acceptance Criteria
- `pctl deploy` with a compose containing one or more `build:` services completes successfully in both modes.
- Deterministic tags applied; transformed compose sent to Portainer; containers start with built images present on the remote engine.
- Clear, concise build logs per service; helpful errors on failure.

### Out of Scope (for this iteration)
- Running a managed registry in the target environment.
- Build secrets and advanced BuildKit frontends (can be considered later).
- Full per‑service overrides beyond what compose `build.*` already supports.

### Decisions captured
- Default mode: remote-build.
- Tagging: default content-hash; `--force-build` available.
- Parallelism: derive from remote CPU; allow override.
- Private base images: ignored for v1.
- Force build option: supported via flag and config.
- Warnings: emit WARN at >50 MB for context tar or image tar streams.
- Buildx: acceptable requirement for load mode.

### Implementation Tasks (sequenced)
1. Config schema: add `build` section; load/validate.
2. Compose parser: detect `build:` and resolve context/dockerfile/args/target.
3. Context tar streamer honoring `.dockerignore`.
4. Client methods:
   - Remote build: call Docker Build API via Portainer proxy; stream logs.
   - Load mode: upload image tars to Images Load API; stream progress.
   - Helpers: remote platform probe; image existence probe.
5. Tagging & hashing utilities.
6. Build orchestrator with parallelism, logging, and error collation.
7. Compose transformer: replace `build:` with `image:`.
8. Wire into `deploy` (and `redeploy`) flow with flags and config overrides.
9. Tests (unit/integration) and docs updates.


