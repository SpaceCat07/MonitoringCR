# MonitoringCR API

Backend API untuk autentikasi user untuk Monitoring CR

## Tech Stack

- Go
- Gin (`github.com/gin-gonic/gin`)
- GORM (`gorm.io/gorm`)
- PostgreSQL (`gorm.io/driver/postgres`)
- JWT (`github.com/golang-jwt/jwt/v5`)
- Godotenv (`github.com/joho/godotenv`)

## Fitur

- Register user
- Login user
- Generate JWT saat login
- Middleware proteksi endpoint dengan `Authorization: Bearer <token>`
- Auto-migrate tabel `users`
- CORS aktif

## Struktur Project

```text
.
|-- config/
|   `-- db.go
|-- controllers/
|   `-- auth_controller.go
|-- middleware/
|   `-- auth_middleware.go
|-- models/
|   `-- users.go
|-- routes/
|   `-- route.go
|-- utils/
|   `-- helperjwt.go
|-- .env.example
|-- go.mod
|-- go.sum
`-- main.go
```

## Prasyarat

- Go terpasang
- PostgreSQL berjalan

## Konfigurasi Environment

Buat file `.env` di root project dan isi seperti contoh berikut:

```env
# Database PostgreSQL
DB_HOST=localhost
DB_PORT=5433
DB_USER=postgres
DB_PASSWORD=1234
DB_NAME=monitoringcr

# JWT
JWT_SECRET=jdsklfjdskajflkasjfdskafdlaskfdjkslafjdslka
```

## Menjalankan Project

```bash
go mod tidy
go run main.go
```

Server berjalan default di:

```text
http://localhost:8080
```

## API Endpoints

Base URL: `http://localhost:8080`

### Public

1. `POST /api/register`

Body JSON:

```json
{
  "fullname": "John Doe",
  "email": "john@example.com",
  "password": "rahasia123"
}
```

2. `POST /api/login`

Body JSON:

```json
{
  "email": "john@example.com",
  "password": "rahasia123"
}
```

Contoh response sukses:

```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "id": 1,
    "full_name": "John Doe"
  },
  "token": "<jwt_token>"
}
```

### Protected (Wajib JWT)

1. `GET /api/cek`

Header wajib:

```http
Authorization: Bearer <jwt_token>
```

Jika token valid: endpoint dapat diakses.
Jika token tidak valid/tidak ada: response `401 Unauthorized`.

### Health Check

1. `GET /oke`

## Catatan

- `DBConnect()` melakukan `AutoMigrate(&models.Users{})` saat aplikasi berjalan.
- Pastikan `.env` di-ignore dari Git dan hanya commit `.env.example`.
- Untuk production, gunakan nilai `JWT_SECRET` yang kuat dan acak (minimal 32 karakter).
