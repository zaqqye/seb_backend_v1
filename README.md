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
  
  Rooms (admin-only):
- `GET  /api/v1/admin/rooms`          — list rooms (pagination/sort supported)
- `POST /api/v1/admin/rooms`          — create room (body: `name`, optional `active`)
- `GET  /api/v1/admin/rooms/:id`      — get room by numeric id
- `PUT  /api/v1/admin/rooms/:id`      — update room (partial: `name`, `active`)
- `DELETE /api/v1/admin/rooms/:id`    — delete room

  Majors/Jurusan (admin-only):
- `GET  /api/v1/admin/majors`         — list majors (pagination/sort supported)
- `POST /api/v1/admin/majors`         — create major (body: `code`, `name`)
- `GET  /api/v1/admin/majors/:id`     — get major by numeric id
- `PUT  /api/v1/admin/majors/:id`     — update major (partial: `code`, `name`)
- `DELETE /api/v1/admin/majors/:id`   — delete major

  Assignments (admin-only):
- `POST   /api/v1/admin/rooms/:room_id/supervisors`          — assign pengawas to room (body: `user_id`)
- `DELETE /api/v1/admin/rooms/:room_id/supervisors/:user_id` — unassign pengawas from room
- `POST   /api/v1/admin/rooms/:room_id/students`             — assign siswa to room (body: `user_id`)
- `DELETE /api/v1/admin/rooms/:room_id/students/:user_id`    — unassign siswa from room

  SDUI & Remote Config:
- `GET /api/v1/sdui/screens/:name`       — public; returns JSON screen (login works without auth)
- `GET /api/v1/sdui/auth/screens/:name`  — requires auth; role-aware screens
- `GET /api/v1/config/public`            — public remote config
- `GET /api/v1/config`                   — requires auth; role-aware flags

  Exit Codes (admin + pengawas):
- `POST /api/v1/exit-codes/generate` — generate single-use exit code. Body:
  - `room_id` (required for pengawas; optional for admin)
  - `length` (optional, default 6)
- `GET  /api/v1/exit-codes` — list exit codes with query params:
  - `limit`, `page`, `all`, `sort_by` (id, created_at, used_at, code), `sort_dir`
  - `room_id` — filter by room
  - `used` — `true|false|all` (default `false`, only unused codes)
  - pengawas hanya melihat data untuk ruangan yang diawasi
- `POST /api/v1/exit-codes/:id/revoke` — revoke (mark as used now)
- `POST /api/v1/exit-codes/consume`    — consume code (single-use)

  Rooms List Pagination/Sort/Filter
- `GET /api/v1/admin/rooms` supports query params:
  - `limit` (int, default 20), `page` (int, default 1)
  - `all` (bool: `true`/`1`) to return all without pagination
  - `sort_by` in: `id, created_at, name, active`
  - `sort_dir` in: `ASC` or `DESC` (default `DESC`)
  - `q` (search) — ILIKE on `name`
  - `active` — `true|false|1|0`

  Majors List Pagination/Sort/Filter
- `GET /api/v1/admin/majors` supports query params:
  - `limit` (int, default 20), `page` (int, default 1)
  - `all` (bool: `true`/`1`) to return all without pagination
  - `sort_by` in: `id, created_at, code, name`
  - `sort_dir` in: `ASC` or `DESC` (default `DESC`)
  - `q` (search) — ILIKE on `code` or `name`

**Constraints**
- Rooms: `name` is unique. Creating/updating with duplicate `name` returns 409 Conflict.
- Majors: `code` is unique. Creating/updating with duplicate `code` returns 409 Conflict.
- Exit codes: `code` unique. Single-use; revoke/consume sets `used_at` to now. Listing hides used codes by default.

**Environment**
- `PORT` — server port, default 8080
- `DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE` — PostgreSQL
- `JWT_SECRET` — secret used to sign tokens
- `JWT_EXPIRES_IN` — minutes until token expires
- `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `ADMIN_FULL_NAME` — seed first admin if none exists

**Notes**
- On first run, the server auto-migrates the `users` table.
- Admin manages registration via `POST /api/v1/admin/users`. If `role` is omitted, defaults to `siswa`. `active` defaults to `true`.
- If you see a Postgres error like `simple protocol queries must be run with client_encoding=UTF8`, ensure your Postgres instance supports UTF8 client encoding. The DSN in code sets `client_encoding=UTF8`; alternatively set env `PGCLIENTENCODING=UTF8`.

**Admin List Users Pagination/Sort/Filter**
- `GET /api/v1/admin/users` supports query params:
  - `limit` (int, default 20), `page` (int, default 1)
  - `all` (bool: `true`/`1`) to return all without pagination
  - `sort_by` in: `id, created_at, full_name, email, role, kelas, jurusan, active`
  - `sort_dir` in: `ASC` or `DESC` (default `DESC`)
  - `q` (search) — ILIKE on `full_name` or `email`
  - `role` — filter by role (`admin|pengawas|siswa`)
  - `active` — `true|false|1|0`
  Exit Codes (admin + pengawas):
- `POST /api/v1/exit-codes/generate` — generate exit code. Body:
  - `room_id` (required for pengawas; optional for admin)
  - `expires_in_minutes` (int, default 10)
  - `length` (optional, default 6)
- `GET  /api/v1/exit-codes` — list exit codes with query params:
  - `limit`, `page`, `all`, `sort_by` (id, created_at, expired_at, code), `sort_dir`
  - `room_id` — filter by room
  - `active` — `true|false|1|0` (by `expired_at` vs now)
  - pengawas hanya melihat data untuk ruangan yang diawasi
- `POST /api/v1/exit-codes/:id/revoke` — revoke (expire now)

- Exit codes: `code` unique. Revoke sets `expired_at` to now.

**Notes (Exit Codes)**
- Pengawas hanya boleh generate/list/revoke untuk `room_id` yang menjadi pengawasnya.
- Admin boleh untuk semua ruangan dan juga tanpa `room_id` (global code).
- Mapping pengawas ke ruangan menggunakan tabel `room_supervisors` (dikelola admin atau via DB).
\n