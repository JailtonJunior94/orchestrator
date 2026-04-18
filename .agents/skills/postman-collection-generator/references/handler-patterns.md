# Handler Discovery Patterns by Language

## Go

### Route Registration Patterns
```
r.HandleFunc("[path]", handler)              // net/http
r.GET("[path]", handler)                     // gin
e.GET("[path]", handler)                     // echo
r.Route("[path]", func(r chi.Router){...})   // chi
router.Handle("[method]", "[path]", handler) // httprouter
group.Add("[method]", "[path]", handler)     // fiber
```

### Search Patterns (regex)
- Routes: `\.(GET|POST|PUT|PATCH|DELETE|Handle|HandleFunc)\s*\(`
- Handlers: `func\s+\w+\s*\(\s*(w\s+http\.ResponseWriter|c\s+\*?(gin\.Context|echo\.Context|fiber\.Ctx))`
- DTOs/Models: `type\s+\w+(Request|Response|Input|Output|DTO|Payload|Body)\s+struct`
- JSON tags: `` `json:"(\w+)(,omitempty)?"` ``
- Status codes: `(WriteHeader|Status|JSON|c\.JSON)\s*\(\s*(http\.Status\w+|\d{3})`
- Middleware: `(Use|Middleware)\s*\(` and `func\s+\w+Middleware`
- Path params: `{(\w+)}` or `:(\w+)`
- Query params: `(Query|FormValue|URL\.Query)\s*\(\s*"(\w+)"`
- Validation: `validate:"([^"]+)"`

## C# (.NET)

### Route Registration Patterns
```
[HttpGet("[path]")]           // Controller attribute
[HttpPost("[path]")]
[Route("[path]")]
app.MapGet("[path]", handler) // Minimal API
app.MapPost("[path]", handler)
endpoints.MapControllerRoute(...)
```

### Search Patterns (regex)
- Routes: `\[Http(Get|Post|Put|Patch|Delete)(\("([^"]*)")?\]`
- Route prefix: `\[Route\("([^"]+)"\)\]`
- Minimal API: `app\.Map(Get|Post|Put|Patch|Delete)\s*\(`
- Controllers: `class\s+(\w+)Controller\s*:\s*Controller(Base)?`
- DTOs/Models: `class\s+(\w+(Request|Response|Dto|ViewModel|Command|Query|Input|Output))`
- Properties: `public\s+(\w+[\?\[\]<>,\s]*)\s+(\w+)\s*\{\s*get;\s*set;\s*\}`
- JSON properties: `\[JsonPropertyName\("(\w+)"\)\]`
- Status codes: `(Ok|Created|BadRequest|NotFound|Unauthorized|NoContent|StatusCode)\s*\(`
- Validation: `\[Required\]|\[MaxLength|\[MinLength|\[Range|\[RegularExpression`
- Authorize: `\[Authorize(\("([^"]+)")?\]`
- Path params: `\{(\w+)\}`

## TypeScript (Node.js / NestJS / Express / Fastify)

### Route Registration Patterns
```
app.get("[path]", handler)              // Express
router.get("[path]", handler)           // Express Router
@Get("[path]")                          // NestJS
fastify.get("[path]", opts, handler)    // Fastify
```

### Search Patterns (regex)
- Express routes: `(app|router)\.(get|post|put|patch|delete)\s*\(\s*['"]([^'"]+)`
- NestJS decorators: `@(Get|Post|Put|Patch|Delete)\s*\(\s*['"]?([^'")\s]*)`
- NestJS controllers: `@Controller\s*\(\s*['"]([^'"]+)`
- Fastify routes: `fastify\.(get|post|put|patch|delete)\s*\(`
- DTOs/Interfaces: `(interface|class|type)\s+(\w+(Request|Response|Dto|Input|Output|Body|Params|Query))`
- Properties: `(\w+)(\?)?\s*:\s*(\w+[\[\]<>,\s|]*)`
- Validation (class-validator): `@Is(String|Number|Email|NotEmpty|Optional|Int|Boolean|Array|Enum|Date)`
- Status codes: `(status|httpCode|HttpStatus)\s*\(\s*(\d{3}|HttpStatus\.\w+)`
- Path params: `:(\w+)`
- Guards/Middleware: `@UseGuards|@UseInterceptors|app\.use`
- Zod schemas: `z\.(object|string|number|boolean|array|enum)\(`
