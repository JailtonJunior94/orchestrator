# Standard Response Scenarios per HTTP Method

Generate these scenarios for each discovered endpoint. Adapt status codes and bodies to what the handler actually returns.

## GET (single resource)
| Scenario | Status | Description |
|----------|--------|-------------|
| Success | 200 | Returns the resource with all fields populated |
| Not Found | 404 | Resource ID does not exist |
| Unauthorized | 401 | Missing or invalid auth token |
| Forbidden | 403 | Valid token but insufficient permissions |

## GET (list/collection)
| Scenario | Status | Description |
|----------|--------|-------------|
| Success (with data) | 200 | Returns array with 2-3 realistic items |
| Success (empty) | 200 | Returns empty array |
| Filtered | 200 | Applies query params, returns filtered subset |
| Unauthorized | 401 | Missing or invalid auth token |

## POST (create)
| Scenario | Status | Description |
|----------|--------|-------------|
| Created | 201 | All required fields, returns created resource with ID |
| Validation Error | 400/422 | Missing required field or invalid format |
| Conflict | 409 | Duplicate unique constraint (if applicable) |
| Unauthorized | 401 | Missing or invalid auth token |

## PUT/PATCH (update)
| Scenario | Status | Description |
|----------|--------|-------------|
| Updated | 200 | Valid update, returns updated resource |
| Not Found | 404 | Resource ID does not exist |
| Validation Error | 400/422 | Invalid field value |
| Unauthorized | 401 | Missing or invalid auth token |

## DELETE
| Scenario | Status | Description |
|----------|--------|-------------|
| Deleted | 200/204 | Resource successfully deleted |
| Not Found | 404 | Resource ID does not exist |
| Unauthorized | 401 | Missing or invalid auth token |

## Realistic Data Generation Rules
- Use UUIDs for IDs: `"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
- Use ISO 8601 for dates: `"2024-01-15T10:30:00Z"`
- Use realistic emails: `"john.doe@example.com"`
- Use realistic names: `"John Doe"`, `"Jane Smith"`
- Use realistic phone: `"+5511999887766"`
- Use realistic amounts: `199.90`, `1250.00`
- Use realistic addresses when needed
- Booleans should reflect meaningful state: `"active": true`
- Enums should use actual values found in code
- Arrays should contain 2-3 items (not 1, not 10)
- Nested objects must reflect the actual struct/class hierarchy
