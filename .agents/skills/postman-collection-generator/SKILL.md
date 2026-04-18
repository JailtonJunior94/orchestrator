---
name: postman-collection-generator
description: Analyzes project handlers across Go, C#, and TypeScript codebases to generate a complete Postman collection with realistic request/response JSON bodies. Scans route definitions, DTOs, models, and middleware to produce accurate endpoint entries with all possible scenarios (success, validation errors, not found, unauthorized). Use when generating API documentation, onboarding developers to an existing API, or creating integration test collections. Do not use for generating OpenAPI/Swagger specs, unit tests, or non-HTTP-based services like gRPC or message queues.
---

# Postman Collection Generator

## Procedures

**Step 1: Detect Project Language**
1. Execute `bash scripts/detect-language.sh [project_root]` to identify supported languages (go, csharp, typescript).
2. If the script exits with code 1, inform the user that no supported language was detected and halt.
3. Store the detected language(s) for use in subsequent steps.

**Step 2: Discover Routes and Handlers**
1. Read `references/handler-patterns.md` to load the regex patterns for the detected language(s).
2. For each detected language, perform the following searches across the project:

   **Go:**
   - Search for route registration patterns in `main.go`, `router.go`, `routes.go`, or files under `cmd/`, `internal/`, `api/`, `handler/`, `handlers/`, `infrastructure/`, `presentation/`.
   - Search for handler function signatures to find all HTTP handler implementations.
   - Trace each route to its handler function. Map: `{method, path, handler_function, source_file}`.

   **C#:**
   - Search for controller classes inheriting `ControllerBase` or `Controller`.
   - Extract `[Route]` prefix from controller class and `[Http*]` attributes from each method.
   - Also search for Minimal API registrations (`app.Map*`) in `Program.cs` or startup files.
   - Map: `{method, path, controller_method, source_file}`.

   **TypeScript:**
   - Search for Express/Fastify route registrations (`app.get`, `router.post`, etc.).
   - Search for NestJS decorators (`@Controller`, `@Get`, `@Post`, etc.).
   - Trace each route to its handler. Map: `{method, path, handler_function, source_file}`.

3. Build a complete route inventory as a structured list. Include the full resolved path (combining prefixes, group paths, and route-level paths).

**Step 3: Extract DTOs, Models, and Validation Rules**
1. For each handler discovered in Step 2, read the handler source file.
2. Identify the request body type:
   - **Go:** Look for `json.Decode`, `c.Bind`, `c.ShouldBindJSON`, or similar. Trace to the struct definition. Extract all fields with their `json` tags and `validate` tags.
   - **C#:** Look for `[FromBody]` parameters. Read the DTO/Command class. Extract all properties with `[JsonPropertyName]` and validation attributes (`[Required]`, `[MaxLength]`, etc.).
   - **TypeScript:** Look for `req.body` typing, NestJS `@Body()` decorator, or Zod schema. Extract all fields with types and validation decorators.
3. Identify the response body type:
   - **Go:** Look for `json.Encode`, `c.JSON`, `JSON()` calls. Trace to the response struct.
   - **C#:** Look for `Ok()`, `Created()`, `BadRequest()` return types. Trace the returned object type.
   - **TypeScript:** Look for `res.json()`, `res.status().json()`, or NestJS return types.
4. For each DTO/model, recursively resolve nested types until all fields are primitive or well-known types (UUID, DateTime, etc.).
5. Record validation constraints (required, min/max length, regex patterns, enums) to generate both valid and invalid scenario payloads.

**Step 4: Identify Middleware and Auth Requirements**
1. Search for authentication/authorization middleware applied globally or per-route:
   - **Go:** `Use(authMiddleware)`, JWT middleware, custom middleware functions.
   - **C#:** `[Authorize]`, `[AllowAnonymous]`, policy-based authorization.
   - **TypeScript:** `@UseGuards(AuthGuard)`, Express `passport.authenticate`, middleware arrays.
2. For each route, determine if authentication is required. Tag routes accordingly.
3. Identify any additional headers required (API keys, custom headers).

**Step 5: Determine Base URL and Port**
1. Search for server startup configuration:
   - **Go:** `http.ListenAndServe(":PORT", ...)`, environment variable references.
   - **C#:** `launchSettings.json`, `appsettings.json`, `Program.cs` Kestrel config.
   - **TypeScript:** `app.listen(PORT)`, `.env` files, config modules.
2. Set the `base_url` variable to the discovered host and port (default to `http://localhost:{port}`).

**Step 6: Generate Scenario Payloads**
1. Read `references/response-scenarios.md` to load the standard scenario matrix.
2. For each route from Step 2, generate scenarios based on its HTTP method:
   - **Success scenario:** Populate all request fields with realistic data matching the DTO types and constraints. Generate a response body matching the response DTO with all fields populated.
   - **Validation error scenario (POST/PUT/PATCH):** Omit a required field or send an invalid value. Generate the error response body matching the project's error format (inspect existing error handling patterns).
   - **Not Found scenario (GET/PUT/PATCH/DELETE with path params):** Use a non-existent ID. Generate 404 response.
   - **Unauthorized scenario (if auth required):** Remove the Authorization header. Generate 401 response.
3. All JSON bodies must:
   - Use the exact field names from `json` tags / `JsonPropertyName` / interface definitions.
   - Use correct types (string, number, boolean, array, object) matching the code.
   - Use realistic values following the rules in `references/response-scenarios.md`.
   - Include ALL fields from the DTO, not a subset.

**Step 7: Assemble the Postman Collection**
1. Read `assets/postman-collection-schema.json` to load the output structure template.
2. Build the collection JSON following Postman Collection v2.1.0 format:
   - Set `info.name` to the project name (from `go.mod`, `.csproj`, or `package.json`).
   - Create one **folder** per resource/controller group (e.g., "Users", "Orders", "Products").
   - Inside each folder, create one **request item per scenario** with:
     - Descriptive name: `"{Method} {Path} - {Scenario}"` (e.g., `"POST /users - Created"`, `"POST /users - Validation Error"`).
     - Complete request: method, URL (using `{{base_url}}` variable), headers, and body.
     - One **saved response example** with the expected status code and response body.
   - Set collection-level auth to Bearer Token using `{{auth_token}}` variable.
   - Define collection variables: `base_url`, `auth_token`.
3. For endpoints with path parameters, use Postman's `:param` syntax in the URL and add the parameter to `url.variable`.
4. For endpoints with query parameters, add them to `url.query` with example values.

**Step 8: Write and Validate Output**
1. Write the assembled JSON to `postman-collection.json` in the project root directory.
2. Execute `python3 scripts/validate-collection.py postman-collection.json` to validate the output.
3. If validation fails, read the error output, fix the identified issues in the JSON, and re-validate.
4. Report the final summary: number of folders, total requests, and scenarios covered.

## Error Handling
* If `scripts/detect-language.sh` exits with code 1, halt and inform the user that the project must contain Go, C#, or TypeScript source files.
* If no routes are discovered in Step 2, search alternative directory patterns (`src/`, `app/`, `server/`, `web/`). If still empty, inform the user and halt.
* If a DTO/model cannot be fully resolved (e.g., references an external package type), use the type name as a placeholder and add a comment in the scenario noting the unresolved type.
* If `scripts/validate-collection.py` reports invalid JSON in request/response bodies, regenerate only the affected bodies and re-validate.
* If the project uses a non-standard error response format, inspect the error handling middleware or utility and replicate that exact format in error scenarios.
