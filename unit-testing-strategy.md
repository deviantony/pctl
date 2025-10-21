# Unit Testing Strategy for pctl

## Overview

This plan introduces comprehensive unit test coverage for the pctl codebase, starting with pure functions and gradually moving toward more complex integration points. We'll use Go's standard testing framework with table-driven tests where appropriate.

## Testing Priorities (Ordered)

### Phase 1: Core Configuration & Utilities (High Value, Low Complexity)

#### 1. `internal/config/config.go`

**Rationale**: Foundation of the application, pure data structures and validation logic.

**Test Cases**:

- `TestLoad()` - Load configuration from valid YAML file
- `TestLoad_FileNotFound()` - Error when config file doesn't exist
- `TestLoad_InvalidYAML()` - Error on malformed YAML
- `TestConfig_Save()` - Successfully write configuration to file
- `TestConfig_Validate()` - Validate required fields (PortainerURL, APIToken, EnvironmentID, StackName, ComposeFile)
- `TestConfig_Validate_MissingFields()` - Error for each missing required field
- `TestBuildConfig_Validate()` - Valid build modes (remote-build, load)
- `TestBuildConfig_Validate_InvalidMode()` - Error on invalid build mode
- `TestBuildConfig_Validate_InvalidParallel()` - Error on invalid parallel value
- `TestConfig_GetBuildConfig()` - Returns build config with defaults applied
- `TestConfig_GetBuildConfig_NilBuild()` - Returns default build config when nil
- `TestGetDefaultStackName()` - Generates stack name from directory
- `TestGetDefaultComposeFile()` - Returns default compose file name

#### 2. `internal/errors/formatter.go`

**Rationale**: Pure string formatting functions, easy to test, high code coverage value.

**Test Cases**:

- `TestFormatError_Timeout()` - Formats timeout errors with friendly message
- `TestFormatError_ConnectionRefused()` - Formats connection refused errors
- `TestFormatError_Certificate()` - Formats certificate/TLS errors
- `TestFormatError_Generic()` - Formats generic errors
- `TestContainsAny()` - Tests substring matching with various inputs
- `TestContainsAny_EmptyString()` - Edge case: empty string
- `TestContainsAny_NoMatch()` - No matching substrings

### Phase 2: Build System (Core Business Logic)

#### 3. `internal/build/tagging.go`

**Rationale**: Pure functions for tag generation, deterministic hashing, critical for build system.

**Test Cases**:

- `TestTagGenerator_GenerateTag()` - Generate tag with all template variables
- `TestTagGenerator_GenerateTag_StackVariable()` - Replaces {{stack}} correctly
- `TestTagGenerator_GenerateTag_ServiceVariable()` - Replaces {{service}} correctly
- `TestTagGenerator_GenerateTag_HashVariable()` - Replaces {{hash}} correctly
- `TestTagGenerator_GenerateTag_TimestampVariable()` - Replaces {{timestamp}} correctly
- `TestContentHasher_HashBuildContext()` - Generates consistent hash for same context
- `TestContentHasher_HashBuildContext_DifferentContent()` - Different hash for different content
- `TestContentHasher_HashBuildContext_WithBuildArgs()` - Includes build args in hash
- `TestContentHasher_HashBuildContext_WithDockerignore()` - Respects .dockerignore patterns
- `TestContentHasher_HashBuildContext_Deterministic()` - Same input produces same hash
- `TestTagValidator_ValidateTag()` - Validates valid Docker tags
- `TestTagValidator_ValidateTag_Empty()` - Error on empty tag
- `TestTagValidator_ValidateTag_TooLong()` - Error on tag > 128 chars
- `TestTagValidator_ValidateTag_InvalidChars()` - Error on whitespace/invalid chars
- `TestTagTemplateValidator_ValidateTagFormat()` - Validates valid tag templates
- `TestTagTemplateValidator_ValidateTagFormat_InvalidVariable()` - Error on invalid template variable
- `TestTagTemplateValidator_ValidateTagFormat_UnclosedVariable()` - Error on unclosed template
- `TestSanitizeServiceName()` - Sanitizes service names for tags
- `TestSanitizeStackName()` - Sanitizes stack names for tags

#### 4. `internal/build/context.go`

**Rationale**: File system operations with .dockerignore handling, important for build integrity.

**Test Cases**:

- `TestContextTarStreamer_CreateTarStream()` - Creates tar stream from directory
- `TestContextTarStreamer_loadDockerignore()` - Loads .dockerignore patterns
- `TestContextTarStreamer_loadDockerignore_NotFound()` - Returns empty when .dockerignore missing
- `TestContextTarStreamer_shouldIgnore()` - Correctly ignores patterns
- `TestContextTarStreamer_shouldIgnore_WildcardPattern()` - Handles wildcard patterns
- `TestContextTarStreamer_shouldIgnore_DirectoryPattern()` - Handles directory patterns (trailing /)
- `TestContextTarStreamer_matchesPattern()` - Pattern matching logic
- `TestContextTarStreamer_GetContextSize()` - Calculates context size
- `TestContextTarStreamer_GetContextSize_WithIgnore()` - Excludes ignored files from size
- `TestContextTarStreamer_ValidateContext()` - Validates build context directory
- `TestContextTarStreamer_ValidateContext_NotDirectory()` - Error when not a directory

### Phase 3: Compose File Parsing & Transformation

#### 5. `internal/compose/parser.go`

**Rationale**: Core compose file parsing logic, handles different YAML formats.

**Test Cases**:

- `TestParseComposeFile()` - Parses valid compose file
- `TestParseComposeFile_InvalidYAML()` - Error on invalid YAML
- `TestComposeFile_FindServicesWithBuild()` - Finds services with build directives
- `TestComposeFile_FindServicesWithBuild_None()` - Returns empty when no build directives
- `TestExtractBuildInfo_StringFormat()` - Handles simple build: "./path" format
- `TestExtractBuildInfo_MapFormat()` - Handles complex build object format
- `TestExtractBuildInfo_WithArgs()` - Extracts build args correctly
- `TestExtractBuildInfo_WithTarget()` - Extracts target correctly
- `TestExtractBuildInfo_WithCacheFrom()` - Extracts cache_from correctly
- `TestExtractBuildInfo_DefaultDockerfile()` - Sets Dockerfile default
- `TestComposeFile_HasBuildDirectives()` - Returns true when build directives exist
- `TestComposeFile_GetServiceNames()` - Returns all service names
- `TestComposeFile_GetBuildContextSummary()` - Generates summary string

#### 6. `internal/compose/transformer.go`

**Rationale**: Transforms compose files by replacing build directives with image tags.

**Test Cases**:

- `TestTransformComposeFile()` - Transforms compose file with build directives
- `TestTransformComposeFile_RemovesBuild()` - Removes build directive from service
- `TestTransformComposeFile_AddsImage()` - Adds image field with correct tag
- `TestTransformComposeFile_MultipleServices()` - Handles multiple services
- `TestTransformComposeFile_ServiceNotFound()` - Error when service not in compose file
- `TestTransformResult_ValidateTransformation()` - Validates transformation result
- `TestTransformResult_ValidateTransformation_BuildRemaining()` - Error if build still present
- `TestTransformResult_ValidateTransformation_ImageMissing()` - Error if image not added
- `TestTransformResult_ValidateTransformation_WrongImage()` - Error if wrong image tag
- `TestTransformResult_GetTransformationSummary()` - Generates summary string
- `TestDiffTransformation()` - Shows differences between original and transformed

#### 7. `internal/compose/compose.go`

**Rationale**: Simple file reading operations, easy to test.

**Test Cases**:

- `TestReadComposeFile()` - Reads existing compose file
- `TestReadComposeFile_NotFound()` - Error when file doesn't exist
- `TestReadComposeFile_Empty()` - Error when file is empty
- `TestValidateComposeFile()` - Validates compose file exists and is readable

### Phase 4: Portainer API Client

#### 8. `internal/portainer/types.go`

**Rationale**: Pure data structures, JSON marshaling/unmarshaling.

**Test Cases**:

- `TestEnvironment_JSONMarshal()` - Marshals Environment to JSON
- `TestEnvironment_JSONUnmarshal()` - Unmarshals JSON to Environment
- `TestStack_JSONMarshal()` - Marshals Stack to JSON
- `TestStack_JSONUnmarshal()` - Unmarshals JSON to Stack
- `TestContainer_JSONMarshal()` - Marshals Container to JSON
- `TestContainer_JSONUnmarshal()` - Unmarshals JSON to Container

#### 9. `internal/portainer/client.go`

**Rationale**: HTTP client with API interactions. Use httptest for mocking.

**Test Cases**:

- `TestNewClient()` - Creates client with default TLS skip
- `TestNewClientWithTLS()` - Creates client with TLS verification control
- `TestClient_newRequest()` - Creates properly formatted HTTP requests
- `TestClient_newRequest_URLFormatting()` - Handles URL formatting (trailing slashes)
- `TestClient_handleErrorResponse()` - Parses API error responses
- `TestClient_handleErrorResponse_EmptyBody()` - Handles empty error response
- `TestValidateURL()` - Validates URL format
- `TestValidateURL_NoScheme()` - Error when URL missing scheme
- `TestValidateURL_NoHost()` - Error when URL missing host
- `TestClient_GetEnvironments()` - Mocked API call to get environments (httptest)
- `TestClient_GetStack()` - Mocked API call to get stack (httptest)
- `TestClient_CreateStack()` - Mocked API call to create stack (httptest)
- `TestClient_UpdateStack()` - Mocked API call to update stack (httptest)

### Phase 5: Build Orchestration (More Complex)

#### 10. `internal/build/orchestrator.go`

**Rationale**: Complex orchestration logic with parallelism. Use mock interfaces.

**Test Cases**:

- `TestNewBuildOrchestrator()` - Creates build orchestrator
- `TestBuildOrchestrator_getParallelism_Auto()` - Auto parallelism calculation
- `TestBuildOrchestrator_getParallelism_Explicit()` - Explicit parallelism value
- `TestBuildOrchestrator_getParallelism_Invalid()` - Defaults to 1 on invalid value
- `TestBuildOrchestrator_BuildServices_Empty()` - Returns empty map for no services
- `TestSimpleBuildLogger_LogService()` - Logs service messages
- `TestSimpleBuildLogger_LogInfo()` - Logs info messages
- `TestSimpleBuildLogger_LogWarn()` - Logs warning messages
- `TestSimpleBuildLogger_LogError()` - Logs error messages

#### 11. `internal/build/logger.go`

**Rationale**: Logging with JSON parsing and styling.

**Test Cases**:

- `TestStyledBuildLogger_cleanDockerLine_JSON()` - Parses Docker JSON output
- `TestStyledBuildLogger_cleanDockerLine_StreamField()` - Extracts stream field
- `TestStyledBuildLogger_cleanDockerLine_ErrorDetail()` - Extracts error details
- `TestStyledBuildLogger_cleanDockerLine_Aux()` - Extracts aux field with image ID
- `TestStyledBuildLogger_cleanDockerLine_PlainText()` - Handles non-JSON lines
- `TestStyledBuildLogger_cleanDockerLine_EmptyLine()` - Handles empty lines

## Testing Infrastructure Setup

### Required Test Dependencies

```go
// go.mod additions
github.com/stretchr/testify v1.11.1  // For assert/require helpers and mocks
```

We'll use testify's `assert` package for fluent test assertions, `require` package for assertions that should stop test execution on failure, and `mock` package for creating test doubles where needed.

### Test File Structure

```
internal/
  config/
    config.go
    config_test.go          # NEW
  errors/
    formatter.go
    formatter_test.go       # NEW
  build/
    tagging.go
    tagging_test.go         # NEW
    context.go
    context_test.go         # NEW
    orchestrator.go
    orchestrator_test.go    # NEW
    logger.go
    logger_test.go          # NEW
  compose/
    compose.go
    compose_test.go         # NEW
    parser.go
    parser_test.go          # NEW
    transformer.go
    transformer_test.go     # NEW
  portainer/
    types.go
    types_test.go           # NEW
    client.go
    client_test.go          # NEW
```

### Test Helpers & Fixtures

Create `internal/testutil/` package with:

- `fixtures.go` - Sample YAML configs, compose files
- `tempdir.go` - Temporary directory helpers for file operations
- `mocks.go` - Mock implementations of interfaces (BuildLogger, etc.)

## Execution Order

1. Phase 1: Config & Errors (foundational)
2. Phase 2: Build tagging & context (core logic)
3. Phase 3: Compose parsing & transformation
4. Phase 4: Portainer types & client
5. Phase 5: Build orchestration

## Success Metrics

- Target: 70-80% code coverage for internal packages
- All pure functions should have >90% coverage
- Critical paths (config, parser, transformer, tagging) should have >85% coverage
- Integration points (client, orchestrator) should have >60% coverage

## To-dos

- [x] Phase 1: Test internal/config/config.go - Configuration loading, validation, defaults (14 test cases) ✅ 90.6% coverage
- [x] Phase 1: Test internal/errors/formatter.go - Error formatting functions (7 test cases) ✅ 100% coverage
- [x] Phase 2: Test internal/build/tagging.go - Tag generation, hashing, validation (24 test cases) ✅ 82-100% coverage
- [x] Phase 2: Test internal/build/context.go - Tar streaming, dockerignore handling (11 test cases) ✅ 73-100% coverage
- [x] Phase 3: Test internal/compose/compose.go - Compose file reading and validation (4 test cases) ✅ 87.5-100% coverage
- [x] Phase 3: Test internal/compose/parser.go - Compose file parsing and build directive extraction (13 test cases) ✅ 75-100% coverage
- [x] Phase 3: Test internal/compose/transformer.go - Compose file transformation (10 test cases) ✅ 79-100% coverage
- [x] Phase 4: Test internal/portainer/types.go - JSON marshaling for API types (6 test cases) ✅ Complete coverage
- [x] Phase 4: Test internal/portainer/client.go - API client with httptest mocks (12 test cases) ✅ 76.8% coverage
- [ ] Phase 5: Test internal/build/logger.go - Docker log parsing and formatting (6 test cases)
- [ ] Phase 5: Test internal/build/orchestrator.go - Build orchestration logic (9 test cases)
- [ ] Create test infrastructure: internal/testutil package with fixtures, tempdir helpers, and mocks
