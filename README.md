**Overview**
- REST API in Go using `gin`, `gorm` (PostgreSQL), JWT auth, and role-based access (`admin`, `pengawas`, `siswa`).
- Config via `.env`. See `.env.example` and create your own `.env`.

**Run**
- Copy `.env.example` to `.env` and adjust values.
- Install Go 1.21+.
- From repo root:
  - `go mod tidy`
  - `go run ./cmd/server`

**Endpoints (v1)**
- `POST /api/v1/auth/login`         — public, returns JWT token.
- `GET  /api/v1/auth/me`            — requires auth, returns current user.
- `POST /api/v1/auth/logout`        — requires auth, stateless logout (client should discard token).
- `GET  /api/v1/admin/users`        — admin only, list users.
- `POST /api/v1/admin/users`        — admin only, create user (register). Body supports `role` and `active`.
- `GET  /api/v1/admin/users/:user_id`   — admin only, get one user.
- `PUT  /api/v1/admin/users/:user_id`   — admin only, update user (partial supported).
- `DELETE /api/v1/admin/users/:user_id` — admin only, delete user.
- `GET  /api/v1/pengawas/panel`     — `pengawas` or `admin`.
- `GET  /api/v1/siswa/panel`        — `siswa` or `admin`.

**Environment**
- `PORT` — server port, default 8080
- `DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE` — PostgreSQL
- `JWT_SECRET` — secret used to sign tokens
- `JWT_EXPIRES_IN` — minutes until token expires
- `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `ADMIN_FULL_NAME` — seed first admin if none exists

**Notes**
- On first run, the server auto-migrates the `users` table.
- Admin manages registration via `POST /api/admin/users`. If `role` is omitted, defaults to `siswa`. `active` defaults to `true`.

**Admin List Users Pagination/Sort**
- `GET /api/v1/admin/users` supports query params:
  - `limit` (int, default 20), `page` (int, default 1)
  - `all` (bool: `true`/`1`) to return all without pagination
  - `sort_by` in: `id, created_at, full_name, email, role, kelas, jurusan, active`
  - `sort_dir` in: `ASC` or `DESC` (default `DESC`)
