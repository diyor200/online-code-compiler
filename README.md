# Go Code Executor API

A lightweight **code execution API** written in Go. Supports multiple languages (Go, Python, JavaScript) and executes code safely inside a **Docker container pool**.

---

## Features

- Execute code in **Go, Python, and JavaScript**
- **Safe execution** using Docker containers:
  - CPU & memory limits
  - Disabled network
  - Temporary folders per request
- **Container pool** for high performance (reuses running containers instead of creating new ones for each request)
- **Timeouts** to prevent long-running code
- Easy to extend for additional languages

---

## API Endpoints

### 1. GET `/api/v1/langs`

Returns the list of supported languages.

**Request:**

```http
GET /api/v1/langs HTTP/1.1
Host: localhost:8080
```

**Response:**

```json
["go", "python", "js"]
```

---

### 2. POST `/api/v1/run`

Executes code in the requested language.

**Request Body:**

```json
{
  "lang": "go",
  "code": "package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"Hello from Go!\") }"
}
```

**Curl Example:**

```bash
curl -X POST http://localhost:8080/api/v1/run \
  -H "Content-Type: application/json" \
  -d '{
        "lang": "python",
        "code": "print(\"Hello from Python!\")"
      }'
```

**Successful Response:**

```json
{
  "result": "Hello from Go!\n"
}
```

**Error Response Examples:**

- **Execution Timeout:**

```json
{
  "error": "execution timeout"
}
```

- **Unsupported Language:**

```json
{
  "error": "unsupported language: ruby"
}
```

- **Code Runtime Error:**

```json
{
  "error": "main.go:5: syntax error: unexpected }"
}
```

---

## Setup

1. **Install Docker** and make sure it can run containers:

```bash
docker version
```

2. **Build the Go project**:

```bash
go build -o code-executor ./cmd/server
```

3. **Run the server**:

```bash
./code-executor
```

4. The server will **auto-start a container pool** for Go, Python, and JS (configured in `executor/docker_executor.go`).

---

## Adding New Languages

To add a new language:

1. Update the `prepare()` function in `executor/docker_executor.go`:

```go
case "ruby":
    fileNam
