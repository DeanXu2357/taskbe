# taskbe

A RESTful task API in Go. It uses only the standard library and keeps the data in memory.

## Quick start

Run the server with Docker:

```sh
docker build -t task-api .
docker run --rm -p 8080:8080 task-api
```

Or run the server with a local Go toolchain (Go 1.18 or later):

```sh
go run ./cmd/server
```

The server listens on port `8080`. To use a different port, set the `PORT` environment variable:

```sh
PORT=9090 go run ./cmd/server
docker run --rm -e PORT=9090 -p 9090:9090 task-api
```

If `PORT` is not a number from 1 to 65535, the server logs `invalid config` and exits with code 1.
On `SIGINT` or `SIGTERM`, the server stops new connections and waits up to 10 seconds for the requests in progress.

## API

The full contract is in [`api/openapi.yaml`](api/openapi.yaml) (OpenAPI 3.0).

A task has this shape:

```json
{
  "id": 1,
  "name": "Buy breakfast",
  "status": 0,
  "version": 1,
  "created_at": "2026-11-17T08:00:00.123456789Z",
  "updated_at": "2026-11-17T08:00:00.123456789Z"
}
```

| Field | Rules |
|---|---|
| `id` | Assigned by the server. Starts at 1 and is never reused. |
| `name` | Required. The server trims leading and trailing white space. The result must be 1 to 200 characters (Unicode code points). |
| `status` | `0` is incomplete. `1` is completed. |
| `version` | Starts at 1. Each successful `PUT` increases it by 1. |
| `created_at`, `updated_at` | UTC, RFC 3339 with nanoseconds. |

A single task is returned as `{"data": <task>}`.
A list is returned as `{"data": [<task>, ...], "next_cursor": "<string>" | null}`.
An error is returned as `{"error": {"code": "<code>", "message": "<message>"}}`.
All responses except `204` have `Content-Type: application/json`.

### `POST /tasks`

Creates a task. `status` is optional. If `status` is omitted, the server uses `0`.

```sh
curl -i -X POST localhost:8080/tasks -d '{"name":"Buy breakfast"}'
```

```
HTTP/1.1 201 Created
Content-Type: application/json
Location: /tasks/1

{"data":{"id":1,"name":"Buy breakfast","status":0,"version":1,"created_at":"...","updated_at":"..."}}
```

### `GET /tasks`

Lists tasks in ascending `id` order, one page at a time.

| Query parameter | Default | Rules |
|---|---|---|
| `limit` | `20` | An integer from 1 to 100. |
| `cursor` | none | The `next_cursor` value from the previous page. |

```sh
curl 'localhost:8080/tasks?limit=2'
```

```json
{"data":[{"id":1,...},{"id":2,...}],"next_cursor":"eyJhZnRlcl9pZCI6Mn0"}
```

To get the next page, send the cursor back. On the last page, `next_cursor` is `null`.

```sh
curl 'localhost:8080/tasks?limit=2&cursor=eyJhZnRlcl9pZCI6Mn0'
```

### `GET /tasks/{id}`

Returns one task. The exercise does not ask for this endpoint. It is added so that a client can read the current `version` of one task before a `PUT`. If the task does not exist, the response is `404`.

```sh
curl localhost:8080/tasks/1
```

### `PUT /tasks/{id}`

Replaces the `name` and the `status` of a task. `name`, `status` and `version` are all required.
`version` must be the version that the client read last.

```sh
curl -i -X PUT localhost:8080/tasks/1 -d '{"name":"Buy lunch","status":1,"version":1}'
```

```
HTTP/1.1 200 OK
Content-Type: application/json

{"data":{"id":1,"name":"Buy lunch","status":1,"version":2,"created_at":"...","updated_at":"..."}}
```

If another request changed the task first, the version does not match and the server rejects the change:

```sh
curl -i -X PUT localhost:8080/tasks/1 -d '{"name":"Buy dinner","status":0,"version":1}'
```

```
HTTP/1.1 409 Conflict
Content-Type: application/json

{"error":{"code":"conflict","message":"version does not match the current task"}}
```

The body can contain `id`. If it does, it must be equal to the `id` in the path.
The server ignores `created_at`, `updated_at` and all unknown fields.

### `DELETE /tasks/{id}`

Deletes a task. The response is `204` with no body. If the task does not exist, the response is `404`.
`DELETE` does not check `version`. It deletes the task whatever its current version is.

```sh
curl -i -X DELETE localhost:8080/tasks/1
```

### Errors

| `code` | HTTP | When |
|---|---|---|
| `invalid_json` | 400 | The body is empty, is not a JSON object, is not valid JSON, or has a field of the wrong type. |
| `validation_failed` | 400 | A field is missing or has an invalid value, `limit` or `cursor` is invalid, or the body `id` does not match the path. |
| `not_found` | 404 | The task does not exist, the path `id` is not valid, or the path is unknown. |
| `method_not_allowed` | 405 | The method is not supported on the path. The `Allow` header lists the supported methods. |
| `conflict` | 409 | The `version` in a `PUT` does not match the current version. |
| `payload_too_large` | 413 | The body is larger than 1 MiB. |
| `internal_error` | 500 | An unexpected error. The details go to the server log only. |

## Design decisions

**Go 1.18 is tested, not only claimed.** "Go 1.18+" means the code must build and pass its tests on 1.18 itself. `go.mod` declares `go 1.18`, and the code avoids newer APIs such as `log/slog`, `errors.Join` and the Go 1.22 `ServeMux` patterns. CI runs the tests on 1.18 and on the current stable release.

**Standard library only.** A few endpoints do not need a framework, and a module with no dependencies cannot break on Go 1.18 because of a dependency. Routing is a small manual dispatch on `/tasks` and `/tasks/{id}`. `internal/logx` defines a logger interface with the method set of `*slog.Logger`, so `slog` can replace it when the minimum Go version allows.

**Layers with interfaces at the consumer.**

```
httpapi  ──uses──▶  httpapi.TaskService  ◀──implements──  service.Service
service  ──uses──▶  service.Store        ◀──implements──  memory.Store
```

- `httpapi` decodes requests, checks that required fields are present, and maps errors to HTTP responses.
- `service` applies the business rules: name trimming and length, valid status, timestamps, and paging.
- `memory` stores the data.

Each interface is declared in the package that uses it. The types that cross the service boundary (the inputs, the list parameters, the page, the default page size and the errors) are in `internal/task`. So `httpapi` does not import `service`. Only `main` and the end-to-end test import it, to connect the real service. To use a database, write a new `Store` implementation. No other layer changes.

**Explicit error contracts.** `service.Store` declares the errors that a store must return: `ErrRecordNotFound` and `ErrVersionMismatch`. The service translates them into the errors of its own contract in `internal/task`. The service wraps every other store error in `task.ErrUnexpected`. Each handler matches each error with `errors.Is` and writes its own message. Two validation errors also put the text of the service error into the message: a name that is too long and a `limit` that is out of range. For these errors, the service wraps the sentinel with the limit, for example `name is too long (at most 200 characters)`. The limits are unexported constants in `service`, so no other package depends on their values. The text of all other errors, for example a store error, goes only to the server log.

**Optimistic locking on `PUT`.** The exercise only says that `PUT /tasks/{id}` updates a task. A plain overwrite of `name` and `status` silently loses one of two concurrent updates. So each task has a `version`, and a `PUT` must send the version that it read. The store compares the version and writes in one atomic step, the same as `UPDATE ... SET version = version + 1 WHERE id = ? AND version = ?`. A stale version gets `409 Conflict`. The cost is that a `PUT` with only `name` and `status` gets `400 version is required`.
The version is in the body and not in an `If-Match` header. The `id` is also allowed in the body, and each item of a list already includes its version.
`DELETE` does not check the version. A delete does not overwrite a change, and a `PUT` after the delete gets `404`, so no update is silently lost. The cost is that a client can delete a task that another client changed after the first client read it.

**`PUT` replaces the whole task.** `name` and `status` are required. The service inputs, `task.CreateInput` and `task.ReplaceInput`, have no pointers, because the HTTP layer never lets a required field reach the service missing. On `POST`, an omitted status decodes to `0`, which is the documented default.

**Keyset pagination with a default page size.** Without parameters, `GET /tasks` returns the first 20 tasks, not all tasks. A list that has no limit becomes slow and large as the data grows. The cursor is an opaque, base64url-encoded `{"after_id": N}`. A keyset cursor is stable when other requests create or delete tasks, which offset paging is not. If the cursor points to a deleted task, the next page starts after that id. The memory store keeps a sorted slice of ids, so a page is a binary search and not a sort of all tasks.

**Strict input rules.**

- Unknown fields are ignored. This keeps old servers compatible with newer clients.
- The body limit is 1 MiB.
- A path `id` must be a canonical positive decimal integer. `/tasks/01`, `/tasks/+1` and `/tasks/1/` get `404`. They do not get a `500` or a plain-text error.

**Production basics.** The server has read, write and idle timeouts. It logs each request with its status and duration. It changes a handler panic into a JSON `500`. It shuts down gracefully on `SIGTERM`. The Docker image is a static binary on `distroless/static` that runs as a non-root user. The image is smaller than 10 MB.

## Project structure

```
api/openapi.yaml               API contract
cmd/server/                    entry point: config, wiring, graceful shutdown
internal/httpapi/              routing, JSON decoding, error mapping, middleware
internal/logx/                 slog-shaped logger interface for Go 1.18
internal/task/                 Task, Status, and the inputs, outputs and errors of the service
internal/task/service/         business rules, size limits, the Store interface and its errors
internal/task/store/memory/    in-memory Store (map + RWMutex + sorted ids)
scripts/smoke.sh               builds and runs the Docker image, then calls each endpoint
.github/workflows/ci.yml       tests on Go 1.18 and stable, smoke test of the image
```

## Testing

Run the unit tests and the end-to-end test:

```sh
go test -race ./...
```

Run the same tests on Go 1.18 in Docker:

```sh
docker run --rm -e GOFLAGS=-buildvcs=false -v "$PWD":/src -w /src golang:1.18 go test -race ./...
```

On a host with SELinux, add `:Z` to the volume: `-v "$PWD":/src:Z`.

Build the Docker image, run it, and call each endpoint:

```sh
scripts/smoke.sh
GO_VERSION=1.18 scripts/smoke.sh
```

The tests are in these groups:

- **Handler tests** use a fake `TaskService`. They check routing, decoding, status codes and error codes.
- **Service tests** use a fake `Store`. They check the business rules and the translation of store errors.
- **Store tests** check id assignment, keyset paging, the version check and concurrent access.
- **The end-to-end test** (`internal/httpapi/e2e_test.go`) sends HTTP requests through the real service and the memory store.
- **The smoke script** checks the built image: the status code of each endpoint and graceful shutdown.

The Dockerfile uses Go 1.27.4 by default, which has the current security fixes. Go 1.18 no longer gets security fixes. The `GO_VERSION` build argument builds the same Dockerfile on Go 1.18.

## Limitations

- The data is in memory. A restart deletes all tasks.
- The data is in one process. You cannot run more than one instance.
- There is no authentication or authorization.
- There is no rate limit.
