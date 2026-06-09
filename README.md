# MonitoringCR

MonitoringCR adalah backend service berbasis Go untuk mengelola Change Request, activity, subtask, user, dashboard statistik, upload attachment, dan autentikasi JWT.

## Ringkasan Fitur

- Login user dengan JWT
- Middleware autentikasi dan role based access
- Manajemen Change Request
- Manajemen subtask dan activity
- Statistik dashboard dan chart
- Upload attachment ke storage lokal
- Export CR ke PDF
- Swagger UI untuk dokumentasi API
- CORS aktif
- Seeder user awal saat database kosong

## Tech Stack

- Go 1.25
- Gin
- GORM
- PostgreSQL
- JWT
- Swagger / Swaggo
- CORS middleware
- Rate limiter in-memory per user

## Struktur Project

```text
.
|-- config/
|   `-- db.go
|-- controllers/
|-- docs/
|-- middleware/
|-- models/
|-- routes/
|-- uploads/
|-- utils/
|-- Dockerfile
|-- docker-compose.yml
|-- go.mod
|-- main.go
`-- README.md
```

## Prasyarat

- Go terpasang
- PostgreSQL berjalan
- File `.env` di root project

## Konfigurasi Environment

Buat file `.env` di root project, lalu isi seperti contoh berikut:

```env
DB_HOST=localhost
DB_PORT=5433
DB_USER=postgres
DB_PASSWORD=1234
DB_NAME=monitoringcr

JWT_SECRET=isi_dengan_secret_yang_kuat_dan_panjang
```

## Menjalankan Project

### Jalankan lokal

```bash
go mod tidy
go run main.go
```

### Jalankan dengan Docker

```bash
docker compose up --build
```

Server berjalan pada:

```text
http://localhost:8080
```

## Endpoint Utama

Base URL: `http://localhost:8080`

### Public

- `POST /api/login`
- `GET /api/roles`
- `GET /oke`
- `GET /swagger/*any`

### Protected dengan JWT

- `GET /api/cek`
- `GET /api/cr/options`
- `GET /api/cr/charts`
- `GET /api/cr/export`
- `GET /api/cr/status/:status`
- `GET /api/cr/modul/:modul`
- `POST /api/cr/attachments/upload`
- `POST /api/cr/draft`
- `POST /api/cr`
- `GET /api/cr`
- `GET /api/cr/:id`
- `PUT /api/cr/:id`
- `DELETE /api/cr/:id`
- `GET /api/subtasks`
- `POST /api/subtasks`
- `GET /api/subtasks/:id`
- `PUT /api/subtasks/:id`
- `DELETE /api/subtasks/:id`
- `GET /api/activities`
- `POST /api/activities`
- `GET /api/activities/:id`
- `PUT /api/activities/:id`
- `DELETE /api/activities/:id`
- `GET /api/collaborator/:PIC_ID`
- `GET /api/dashboard/kpi-summary`
- `GET /api/dashboard/top-pic`
- `GET /api/dashboard/due-today`
- `GET /api/dashboard/module-category`
- `GET /api/dashboard/module-status`
- `GET /api/dashboard/module-health-overview`
- `GET /api/dashboard/lifecycle-line-chart`
- `GET /api/cr/lazy/draft`
- `GET /api/cr/lazy/issued`
- `GET /api/cr/lazy/in-progress`
- `GET /api/cr/lazy/approval-to-release`
- `GET /api/cr/lazy/release`
- `GET /api/cr/lazy/approval-to-complete`
- `GET /api/cr/lazy/complete`
- `GET /api/cr/lazy/cancel`

### Protected Khusus Admin

- `POST /api/users`
- `GET /api/users`
- `GET /api/users/:id`
- `PUT /api/users/:id`
- `DELETE /api/users/:id`

## Autentikasi

Semua endpoint protected wajib menyertakan header berikut:

```http
Authorization: Bearer <jwt_token>
```

## Seeder Default

Saat tabel user masih kosong, aplikasi akan menambahkan data awal berikut:

- `admin@mail.com / admin123` dengan role `Admin`
- `manager@mail.com / manager123` dengan role `Manager`
- `pic@mail.com / pic123` dengan role `PIC`
- `collaborator@mail.com / collaborator123` dengan role `Collaborator`

## Database

Saat aplikasi berjalan, koneksi database akan:

- melakukan auto migrate untuk tabel user, change request, activity, dan subtask
- mengatur connection pool agar lebih stabil

## Swagger

Dokumentasi API tersedia di:

```text
/swagger/index.html
```

## Catatan

- Folder `uploads/` dipakai untuk file attachment lokal.
- Rate limiter berbasis memory server, jadi cocok untuk 1 instance aplikasi.
- Gunakan `JWT_SECRET` yang kuat untuk environment production.
