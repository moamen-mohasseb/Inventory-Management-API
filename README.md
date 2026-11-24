# Inventory Management API

REST API written in [Go](https://go.dev/) that manages an inventory of items. It uses the
[Gin](https://github.com/gin-gonic/gin) web framework, [GORM](https://gorm.io/) and a local
PostgreSQL database. There is no front end: use Postman or cURL to talk to the server.

Each item has an **ID** (UUID), **name**, **stock** and **price**.

## Features

- Full CRUD on `/inventory` and `/inventory/{id}`
- UUID primary keys
- Pagination (`limit`, `offset`)
- Filtering (`name`, `min_stock`)
- Sorting (`sort_by` = name | stock | price, `order` = asc | desc)
- Rate limiting (token bucket per client IP: 1 request/second, bursts of 5)
- Database seeded with 20 items on first start
- JSON responses and meaningful HTTP status codes

## Requirements

- Go 1.23 or newer
- PostgreSQL running on `localhost:5432`

## Installation

1. **Create the database** (once). In the Udacity workspace PostgreSQL is already installed:

   ```bash
   psql postgres postgres -h localhost -c "CREATE DATABASE inventory;"
   ```

   On a local machine you can also use `createdb -U postgres inventory`.

2. **Check the connection string.** It lives in `database.go` (`defaultDSN`):

   ```
   host=localhost user=postgres password=postgres dbname=inventory port=5432 sslmode=disable TimeZone=UTC
   ```

   If your user/password differ, either edit `defaultDSN` or set the `DATABASE_URL` environment variable:

   ```bash
   export DATABASE_URL="host=localhost user=postgres password=YOURPASSWORD dbname=inventory port=5432 sslmode=disable"
   ```

3. **Download the dependencies** (once):

   ```bash
   go mod tidy
   ```

## Launch

```bash
go run .
```

The server listens on `http://localhost:8080` (set `PORT` to change it). On first start the
`items` table is created and filled with 20 sample items.

## Usage

| Method | Path              | Description                          | Success |
| ------ | ----------------- | ------------------------------------ | ------- |
| GET    | `/inventory`      | List items (pagination/filter/sort)  | 200     |
| POST   | `/inventory`      | Create an item                       | 201     |
| GET    | `/inventory/{id}` | Get one item                         | 200     |
| PUT    | `/inventory/{id}` | Replace name, stock and price        | 200     |
| DELETE | `/inventory/{id}` | Delete an item                       | 200     |

Errors: `400` invalid input, `404` item not found, `429` rate limit exceeded, `500` server error.
Every error body looks like `{"error": "message"}`.

### Query parameters for `GET /inventory`

| Parameter   | Default | Meaning                                              |
| ----------- | ------- | ---------------------------------------------------- |
| `limit`     | 10      | Page size (1-100)                                    |
| `offset`    | 0       | Number of items to skip                              |
| `name`      | -       | Case-insensitive "contains" filter on the name       |
| `min_stock` | -       | Only items with stock greater than or equal to this  |
| `sort_by`   | name    | `name`, `stock` or `price`                           |
| `order`     | asc     | `asc` or `desc`                                      |

The response looks like:

```json
{
  "data": [ { "id": "…", "name": "Laptop", "stock": 10, "price": 999.99 } ],
  "pagination": { "total": 20, "limit": 10, "offset": 0 }
}
```

### cURL examples

```bash
# List (first page)
curl "http://localhost:8080/inventory"

# Page 2 with 5 items per page
curl "http://localhost:8080/inventory?limit=5&offset=5"

# Filter by name and minimum stock, sorted by price (highest first)
curl "http://localhost:8080/inventory?name=phone&min_stock=5&sort_by=price&order=desc"

# Create
curl -X POST http://localhost:8080/inventory \
  -H "Content-Type: application/json" \
  -d '{"name":"Desk Lamp","stock":12,"price":24.5}'

# Get one (replace <id> with a real UUID from the list)
curl http://localhost:8080/inventory/<id>

# Update
curl -X PUT http://localhost:8080/inventory/<id> \
  -H "Content-Type: application/json" \
  -d '{"name":"Desk Lamp LED","stock":20,"price":29.99}'

# Delete
curl -X DELETE http://localhost:8080/inventory/<id>
```

### Rate limiting

Each client IP has a token bucket: capacity 5, refilled at 1 token per second. Sending more than
5 requests at once returns `429 Too Many Requests` with a `Retry-After` header. Test it with:

```bash
for i in $(seq 1 8); do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/inventory; done
```

Expected output: five `200` lines followed by `429`.

## Tests

```bash
go test ./...
```

The unit tests cover the rate limiter logic (burst, refill, separate clients) and do not need a database.

## Project structure

| File                 | Purpose                                                    |
| -------------------- | ---------------------------------------------------------- |
| `main.go`            | Starts the server, creates the router, registers routes    |
| `models.go`          | `Item` struct, request/response structs                    |
| `database.go`        | Connection string, GORM connection, migration, seeding     |
| `handlers.go`        | One handler per endpoint + validation helpers              |
| `middleware.go`      | Token-bucket rate limiter and its Gin middleware           |
| `ratelimiter_test.go`| Unit tests for the rate limiter                            |
