# Security & Code Quality Review — Google Workspace MCP Server

**Date:** 2026-02-06
**Reviewer:** Claude Opus 4.6
**Status:** All fixes applied — 12 initial + 27 additional security fixes (Issue 9 deferred as refactoring)

---

## Already Fixed (Commit 3fdc4b8)

These 12 fixes are already committed:

1. CSRF state validation in OAuth flows (oauth.go + oauth_server.go)
2. Localhost binding for OAuth callback server
3. Nil params guards in 4 JSON-RPC handlers (server/mcp.go)
4. 10MB stdin message size limit
5. Race condition: GetAccount/GetAccountForContext use Lock() not RLock()
6. http.DefaultClient replaced with timeout-configured client
7. Query injection prevention via escapeDriveQuery
8. Removed goldmark WithUnsafe() to prevent XSS
9. Download (100MB) and upload (50MB) size limits
10. Delete confirmation required (confirm=true)
11. Shared link permission type/role validation + public access warnings
12. New tests: TestGenerateOAuthState, TestEscapeDriveQuery

---

## Remaining Issues to Fix

### CRITICAL (Fix Immediately)

#### Issue 1: Race condition — goroutines spawned inside mutex locks
- **File:** `auth/account_manager.go:244-248, 276-280`
- `GetAccount()` and `GetAccountForContext()` spawn `go func()` that captures account pointer, then releases lock. The goroutine writes (`saveAccount`) while other goroutines may read concurrently.
- **Fix:** Move goroutine spawn after `mu.Unlock()`, or save synchronously.

#### Issue 2: Refresh token not revoked in Revoke()
- **File:** `auth/oauth.go:383`
- Only access token revoked. Refresh token remains valid indefinitely.
- **Fix:** Revoke both `AccessToken` and `RefreshToken` in separate API calls.

#### Issue 3: Token sent in URL query parameter
- **File:** `auth/oauth.go:383`
- `https://oauth2.googleapis.com/revoke?token=...` leaks via proxy/server logs.
- **Fix:** Send as POST body with `Content-Type: application/x-www-form-urlencoded`.

#### Issue 4: Missing file ID validation across all services
- **Files:** `drive/tools.go`, `tasks/tools.go`
- File IDs, task list IDs, permission IDs passed directly to Google APIs without validation.
- **Fix:** Add `validateID()` helper rejecting empty, too-long, or malformed IDs.

#### Issue 5: Internal error messages exposed to clients
- **File:** `server/mcp.go:320, 431`
- `err.Error()` returned directly in JSON-RPC error responses.
- **Fix:** Return generic message to clients, log full error to stderr.

---

### HIGH (Fix Soon)

#### Issue 6: HTTP server leak on error/timeout paths
- **File:** `auth/oauth.go:133-224`
- In `authenticate()`, error/timeout select branches don't call `server.Shutdown()`.
- **Fix:** `defer server.Shutdown(...)` immediately after `go server.ListenAndServe()`.

#### Issue 7: Token file permissions not checked at load time
- **File:** `auth/oauth.go:227-249`
- `loadToken()` opens file without verifying it's `0600`.
- **Fix:** `os.Stat()` and reject if `mode.Perm() != 0600`.

#### Issue 8: RefreshToken() writes without holding lock
- **File:** `auth/account_manager.go:360-361`
- After `GetAccount()` releases lock, writes to `account.Token` without synchronization.
- **Fix:** Re-acquire `am.mu.Lock()` before writing token fields.

#### Issue 9: Code duplication — tools.go vs multi_account.go
- **Files:** `drive/`, `tasks/`
- Tool definitions and handler logic copy-pasted between files.
- **Fix:** Single source of truth; multi-account handlers delegate to base handlers.

#### Issue 10: DownloadFile has no context timeout or size-bounded copy
- **File:** `drive/client.go:135-147`
- `io.Copy` runs unbounded, no `context.Context` accepted.
- **Fix:** Accept ctx, use `io.LimitReader` on the body.

#### Issue 11: Message size checked AFTER full allocation
- **File:** `server/mcp.go:131-138`
- `ReadBytes('\n')` reads entire line into memory first, then checks size.
- **Fix:** Use `io.LimitReader` upstream of the bufio.Reader.

---

### MEDIUM (Fix Within Sprint)

| # | File | Issue | Fix |
|---|------|-------|-----|
| 12 | `auth/oauth_server.go:57-60` | Missing `ReadTimeout`, `WriteTimeout`, `IdleTimeout` | Add all three timeout fields |
| 13 | `auth/oauth.go:344-354` | Recursive `refreshToken()` with `time.Sleep(100ms)` cascades | Use `time.AfterFunc` instead |
| 14 | `auth/account_manager.go:289` | `strings.Split(email, "@")[1]` panics if no `@` | Check `len(parts) == 2` first |
| 15 | `drive/client.go:96-99` | `escapeDriveQuery` only escapes `'` — boolean operators injectable | Also escape `\` and `"` |
| 16 | `drive/tools.go:611-627` | Upload size checked on base64, not decoded output (33% larger) | Check decoded size too |
| 17 | `drive/tools.go:780-793` | `formatFile`/`formatFiles` ignore json errors | Return errors |
| 18 | `drive/client.go:44-94` | `ListFiles` creates `context.Background()` not caller's | Accept `ctx` param |
| 19 | `drive/client.go:281-315` | `CreateShareLink` defaults to `"anyone"` | Require explicit type |
| 20 | `server/mcp.go:78-90` | No duplicate service registration check | Check and warn/error |
| 21 | `server/mcp.go:291-304` | O(n^2) tool lookup on every call | Build tool-to-service map |
| 22 | `server/mcp.go:94-111` | No graceful shutdown — no `Stop()` method | Add `Stop()` method |
| 23 | `drive/client_test.go:367-413` | Tests log failures with `t.Logf` instead of failing | Use `t.Errorf` |

---

### LOW (Fix When Convenient)

| # | File | Issue | Fix |
|---|------|-------|-----|
| 24 | `auth/oauth.go:303` | Magic number `5*time.Minute` for refresh buffer | Extract to named constant |
| 25 | Multiple files | Inconsistent error wrapping (some `%w`, some raw) | Standardize `%w` wrapping |
| 26 | Multiple files | `defer func() { _ = file.Close() }()` swallows errors | Log close errors |
| 27 | `auth/scope_checker.go:65-89` | Unknown service names silently accepted | Return error for unknown |
| 28 | `server/mcp.go:188+` | All `conn.Reply()` errors discarded with `_ =` | Log reply failures |

---

## Implementation Notes

### Signature Changes That Affect Multiple Files

These fixes require updating callers across the codebase:

- **Issue 10/18** (`DownloadFile`/`ListFiles` accept `context.Context`): Update callers in `drive/tools.go`, `drive/multi_account.go`, `drive/resources.go`
- **Issue 17** (`formatFile`/`formatFiles` return error): Update all callers in `drive/tools.go`, `tasks/tools.go`
- **Issue 19** (`CreateShareLink` no default): Already partially done — verify `drive/multi_account.go` callers

### Files to Modify

| File | Issues |
|------|--------|
| `auth/oauth.go` | 2, 3, 6, 7, 13, 24, 25, 26 |
| `auth/oauth_server.go` | 12 |
| `auth/account_manager.go` | 1, 8, 14 |
| `auth/scope_checker.go` | 27 |
| `server/mcp.go` | 5, 11, 20, 21, 22, 28 |
| `drive/client.go` | 10, 15, 18, 19, 26 |
| `drive/tools.go` | 4, 16, 17 |
| `drive/client_test.go` | 23 |
| `tasks/tools.go` | 4 |
| `tasks/multi_handler.go` | (Issue 9 — deferred refactor) |

### Verification Steps

After all fixes:
```bash
cd ~/MCP/google-mcp-server
go fmt ./...
go vet ./...
go build -o google-mcp-server .
go test ./...
```

Then functional test:
```bash
echo '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | timeout 5 ./google-mcp-server 2>/dev/null
```
