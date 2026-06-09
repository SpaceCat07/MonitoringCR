# MonitoringCR

MonitoringCR adalah backend service berbasis Go untuk mengelola Change Request, activity, subtask, user, statistik dashboard, attachment file, dan autentikasi JWT.

## Fitur Utama

- Login user dengan JWT
- Role based access untuk endpoint admin
- CRUD Change Request
- CRUD subtask
- CRUD activity
- Dashboard KPI dan chart
- Endpoint lazy load per status CR
- Upload attachment ke storage lokal
- Export Change Request ke PDF
- Swagger UI untuk dokumentasi API
- Seeder user default saat database masih kosong
- CORS aktif
- Rate limit per user berbasis memory

## Tech Stack

- Go 1.25
- Gin
- GORM
- PostgreSQL
- JWT
- Swaggo / Swagger
- gin-contrib/cors
- golang.org/x/time untuk rate limiter

## Struktur Project

```text
.
|-- config/
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

- Go sudah terpasang
- PostgreSQL aktif
- File `.env` tersedia di root project

## Konfigurasi Environment

Buat file `.env` di root project dengan isi seperti berikut:

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

Server akan berjalan di:

```text
http://localhost:8080
```

## Endpoint

Base URL: `http://localhost:8080`

### Public Endpoint

- `POST /api/login`
- `GET /api/roles`
- `GET /oke`
- `GET /swagger/*any`

### Protected Endpoint

Semua endpoint berikut wajib memakai header:

```http
Authorization: Bearer <jwt_token>
```

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

### Endpoint Khusus Admin

- `POST /api/users`
- `GET /api/users`
- `GET /api/users/:id`
- `PUT /api/users/:id`
- `DELETE /api/users/:id`

## Login

Request body login:

```json
{
	"email": "admin@mail.com",
	"password": "admin123"
}
```

## Upload Attachment

Endpoint upload menerima multipart form-data dengan key `files`.

Contoh:

```bash
POST /api/cr/attachments/upload
```

## Seeder Default

Jika tabel user masih kosong, aplikasi akan membuat user awal berikut:

- `admin@mail.com / admin123` dengan role `Admin`
- `manager@mail.com / manager123` dengan role `Manager`
- `pic@mail.com / pic123` dengan role `PIC`
- `collaborator@mail.com / collaborator123` dengan role `Collaborator`

## Database

Saat aplikasi dijalankan, database akan:

- auto migrate tabel users, change requests, activities, dan subtasks
- menggunakan connection pool agar lebih stabil

## Swagger

Dokumentasi API bisa dibuka di:

```text
/swagger/index.html
```

## Catatan

- Folder `uploads/` digunakan sebagai storage file lokal dan disajikan lewat route `/uploads`.
- Rate limiter menyimpan data di memory server, jadi cocok untuk satu instance aplikasi.
- Gunakan `JWT_SECRET` yang panjang dan acak untuk environment production.
