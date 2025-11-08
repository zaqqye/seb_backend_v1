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
- `POST /api/v1/admin/users/import` — admin only, import user CSV (multipart `file` field).
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
- `POST   /api/v1/admin/rooms/:id/supervisors`          - assign pengawas to room (body: `user_id` UUID string)
- `DELETE /api/v1/admin/rooms/:id/supervisors/:user_id` - unassign pengawas from room
- `POST   /api/v1/admin/rooms/:id/students`             - assign siswa to room (body: `user_id` UUID string)
- `DELETE /api/v1/admin/rooms/:id/students/:user_id`    - unassign siswa from room

  SDUI & Remote Config:
- `GET /api/v1/sdui/screens/:name`       — public; returns JSON screen (login works without auth)
- `GET /api/v1/sdui/auth/screens/:name`  — requires auth; role-aware screens
- `GET /api/v1/config/public`            — public remote config
- `GET /api/v1/config`                   — requires auth; role-aware flags
  Exit Codes (admin + pengawas):
- `POST /api/v1/exit-codes/generate` - generate single-use exit codes per siswa. Body:
  - `room_id` (required; pengawas hanya bisa untuk ruangan yang diawasi)
  - `student_ids` (optional array) - generate kode hanya untuk siswa tertentu di ruangan tersebut
  - `all_students` (bool, optional) - jika `true`, generate kode untuk seluruh siswa di ruangan; tidak boleh bersamaan dengan `student_ids`
  - `length` (optional, default 6)
- `GET  /api/v1/exit-codes` - list exit codes with query params:
  - `limit`, `page`, `all`, `sort_by` (id, created_at, used_at, code, student_user_id), `sort_dir`
  - `room_id` atau `student_user_id` untuk filter tambahan
  - `used` - `true|false|all` (default `false`, hanya kode yang belum dipakai)
  - pengawas hanya melihat data untuk ruangan yang diawasi
- `POST /api/v1/exit-codes/:id/revoke` - revoke (mark as used now)
- `POST /api/v1/exit-codes/consume`    - konsumsi kode (siswa otomatis memakai kode miliknya; admin/pengawas wajib menyertakan `student_user_id` saat diperlukan)


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
- Semua tabel (users, rooms, majors, exit codes, dsb.) kini memakai UUID string sebagai ID. Path parameter `:id` / `:user_id` / `room_id` / `student_user_id` harus diisi dengan UUID versi terbaru.
- Admin dapat melakukan import massal via `POST /api/v1/admin/users/import` dengan mengunggah file CSV pada field `file`.
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
  - `kelas` — filter exact class (case-insensitive)
  - `jurusan` — filter exact major (case-insensitive)

**Admin User Import (CSV)**
- Endpoint: `POST /api/v1/admin/users/import` (multipart).
- Form field `file` berisi CSV dengan header minimal: `full_name,email,password`.
- Kolom opsional: `role`, `kelas`, `jurusan`, `active` (`true|false|1|0|yes|no`).
- Role default `siswa` bila kosong; hanya menerima `admin|pengawas|siswa`.
- Nilai `active` default `true` jika kolom dikosongkan.
- Respons berisi ringkasan jumlah baris berhasil/gagal beserta daftar error per baris.

  Exit Codes (admin + pengawas):
- `POST /api/v1/exit-codes/generate` - generate single-use exit codes per siswa. Body:
  - `room_id` (required; pengawas hanya bisa untuk ruangan yang diawasi)
  - `student_ids` (optional array) - generate kode hanya untuk siswa tertentu di ruangan tersebut
  - `all_students` (bool, optional) - jika `true`, generate kode untuk seluruh siswa di ruangan; tidak boleh bersamaan dengan `student_ids`
  - `length` (optional, default 6)
- `GET  /api/v1/exit-codes` - list exit codes with query params:
  - `limit`, `page`, `all`, `sort_by` (id, created_at, used_at, code, student_user_id), `sort_dir`
  - `room_id` atau `student_user_id` untuk filter tambahan
  - `used` - `true|false|all` (default `false`, hanya kode yang belum dipakai)
  - pengawas hanya melihat data untuk ruangan yang diawasi
- `POST /api/v1/exit-codes/:id/revoke` - revoke (mark as used now)
- `POST /api/v1/exit-codes/consume`    - konsumsi kode (siswa otomatis memakai kode miliknya; admin/pengawas wajib menyertakan `student_user_id` saat diperlukan)
- Exit codes: `code` unique. Revoke sets `used_at` to now.
 
  Monitoring (admin + pengawas):
 - `GET  /api/v1/monitoring/students` — list siswa status; query: `q`, `room_id`, pagination/sort
 - `POST /api/v1/monitoring/students/:id/logout` — force logout + block from exam
 - `POST /api/v1/monitoring/students/:id/allow` — allow siswa to start exam again
  - `GET /ws/monitoring` (WebSocket) — admin/pengawas menerima update realtime status siswa (pengawas harus sudah di-assign ke ruangan; jika belum, koneksi ditolak)
 
  Student App Status (siswa):
 - `GET  /api/v1/siswa/status` — get current app status
 - `POST /api/v1/siswa/status` — update status; body: `{ app_version, locked }`

**Notes (Exit Codes)**
- Pengawas hanya boleh generate/list/revoke untuk `room_id` yang menjadi pengawasnya.
- Admin dapat generate kode untuk semua ruangan, namun tetap wajib memilih `room_id`; setiap kode melekat pada `student_user_id` tertentu.
- Setiap exit code hanya berlaku untuk siswa yang ditunjuk (single-use).
- Mapping pengawas ke ruangan menggunakan tabel `room_supervisors` (dikelola admin atau via DB).
\n
