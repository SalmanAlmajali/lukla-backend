# Lukla Operations Engine — Architecture Document

**Status:** Draft v0.1  
**Tanggal:** 12 Oktober 2026  
**Tujuan:** Menjadi baseline teknis untuk diskusi, perencanaan implementasi, dan review bersama partner serta klien.  
**Status keputusan:** Dokumen ini memisahkan keputusan awal, rekomendasi teknis, dan hal yang masih harus dikonfirmasi. Ini bukan pengganti SRS atau persetujuan bisnis.

---

## 1. Ringkasan Eksekutif

Lukla Operations Engine adalah sistem operasional manufaktur garmen multi-klien untuk fasilitas produksi di Ciwidey. Sistem perlu menangani sample dan time study, order produksi, inventaris bahan/WIP/barang jadi, alokasi produksi, pencatatan output setiap 30 menit, QC, payroll, biaya, audit, dan portal klien.

Arsitektur awal yang direkomendasikan:

- **Arsitektur aplikasi:** modular monolith.
- **Backend:** Go dengan Gin.
- **Frontend:** TypeScript, Next.js, dan Ant Design.
- **Database:** PostgreSQL.
- **Komunikasi frontend-backend:** REST API melalui HTTPS.
- **Deployment awal:** cloud sebagai kandidat utama, menunggu validasi kondisi internet pabrik dan kebutuhan offline.
- **Tim efektif:** sekitar 1–2 developer; satu developer tambahan dapat membantu QA/dokumentasi secara fleksibel.
- **Message broker/microservices:** tidak digunakan pada fase awal tanpa kebutuhan terukur.
- **Database access:** melalui backend saja; frontend tidak mengakses database secara langsung.

### Prinsip utama

1. Aturan bisnis berada di backend, bukan di frontend.
2. Setiap modul memiliki kepemilikan atas aturan dan data domainnya.
3. Transaksi inventaris, output terverifikasi, QC, dan payroll harus konsisten dan dapat diaudit.
4. Data tenant/klien harus diisolasi pada otorisasi aplikasi dan diperkuat dengan constraint database serta RLS jika dikonfigurasi dengan benar.
5. Histori tarif dan payroll yang dikunci tidak boleh diam-diam ditimpa.
6. Implementasi dilakukan per vertical slice, bukan membangun seluruh modul secara terpisah tanpa integrasi.
7. Keputusan yang belum terkonfirmasi dicatat sebagai asumsi/pertanyaan terbuka.

---

## 2. Konteks dan Batasan

### 2.1 Konteks bisnis dari ringkasan SRS

- Fasilitas produksi melayani Snøhund dan klien maklon B2B.
- Skema order meliputi CMT, Semi-CMT, dan CMT + Bahan/Full Package.
- Inventaris harus ditandai dengan `client_id` dan `order_id` sejak penerimaan.
- Sistem harus menolak pengeluaran bahan untuk order yang tidak cocok dengan pemilik bahan.
- Output produksi dicatat per interval 30 menit dan diverifikasi mandor.
- Produk hasil assembly masuk ke antrean QC.
- Payroll mencakup borongan mingguan, uang makan, pool bonus QC, dan overhead.
- Portal klien hanya boleh melihat data milik klien tersebut.
- SRS menyebut delapan peran RBAC, tetapi juga menyebut Kepala Operasional sebagai penerima bonus QC; hal ini perlu diklarifikasi.

### 2.2 Batasan tim dan delivery

- Backend dan frontend dikerjakan oleh developer berbeda.
- API contract dan mock data perlu tersedia sedini mungkin agar frontend tidak menunggu backend selesai.
- Hindari abstraksi, infrastruktur, dan pola desain yang menambah beban pemeliharaan tanpa manfaat yang jelas.
- Satu codebase backend dan satu deployment aplikasi merupakan baseline awal.

### 2.3 Di luar keputusan final saat ini

Hal-hal berikut belum dapat diputuskan hanya berdasarkan informasi yang tersedia:

- Penyedia cloud, region, biaya, dan topologi jaringan.
- Kebutuhan serta tingkat kemampuan offline.
- Detail formula bisnis payroll dan QC.
- Apakah PO bahan, invoice, dan fulfillment termasuk rilis pertama.
- Kebijakan retensi data, target RPO/RTO, dan kebutuhan compliance spesifik.
- Pilihan library migrasi, driver PostgreSQL, logging, dan observability final.

---

## 3. Gambaran Arsitektur

```mermaid
flowchart TD
    U[Pengguna internal] --> WEB[Next.js + Ant Design]
    C[Klien B2B] --> WEB
    WEB -->|HTTPS / REST JSON| API[Go + Gin API]
    API --> IAM[Identity & Access]
    API --> DOM[Domain modules]
    DOM --> DB[(PostgreSQL)]
    DOM --> AUD[Audit trail]
    DOM --> OUTBOX[Outbox table, jika dibutuhkan]
    OUTBOX --> WORKER[Internal worker / dispatcher]
    DB --> BACKUP[Automated backup & restore testing]
```

Diagram tersebut bersifat logis. Modul bukan service terpisah. Komunikasi internal dilakukan melalui public service/interface di dalam aplikasi Go. Tidak ada koneksi database langsung dari browser.

### 3.1 Komponen

| Komponen | Tanggung jawab |
|---|---|
| Next.js | Routing UI, halaman, layout, pengalaman pengguna |
| Ant Design | Komponen UI, form, tabel, modal, feedback |
| Go + Gin | HTTP routing, middleware, validasi request, pemanggilan application service |
| Application/domain modules | Use case dan aturan bisnis |
| Repository/data access | Query database yang dimiliki modul |
| PostgreSQL | Penyimpanan transaksi, constraint, indeks, locking, RLS jika dipakai |
| Migration runner | Mengelola perubahan skema secara terkontrol |
| Internal worker (opsional) | Pekerjaan tertunda/retry seperti outbox dispatch atau laporan berat |
| Observability | Log terstruktur, metrics/health, error tracking |
| Backup process | Backup terjadwal dan pengujian restore |

---

## 4. Modul dan Batas Tanggung Jawab

Nama modul berikut adalah batas domain logis. Tidak berarti semua modul harus memiliki banyak package atau abstraksi sejak hari pertama.

| Modul | Tanggung jawab utama | Catatan |
|---|---|---|
| `identity` | User, login, sesi/token, status akun | Hindari menyimpan aturan bisnis produksi di sini |
| `access` | Role, permission, policy, akses per tenant | Dapat digabung dengan `identity` pada awal jika lebih sederhana |
| `masterdata` | Klien, artikel, komponen, penjahit, kategori defect/scrap | Pisahkan data referensi global dan data tenant |
| `ratecard` | Tarif komersial dan/atau tarif tenaga kerja, versi dan tanggal berlaku | Definisi jenis rate card harus dikonfirmasi |
| `sample` | Versi sample, konsumsi bahan, time study, approval | Versi yang disetujui harus dapat ditelusuri |
| `order` | Order produksi, skema CMT, status order, snapshot spesifikasi | Order menjadi konteks utama pekerjaan produksi |
| `inventory` | Penerimaan, stok, reservasi, pengeluaran, transfer, scrap, stock opname | Validasi tenant/order harus terjadi sebelum posting transaksi |
| `production` | Line, alokasi pekerja, input output 30 menit, verifikasi, assembly | Output terverifikasi harus memiliki histori dan sumber |
| `quality` | Inspeksi, defect, repair/reinspection, reject/scrap, hasil QC | Terminologi dan transisi status menunggu konfirmasi |
| `payroll` | Periode, upah borongan, uang makan, pool bonus, overhead, locking | Simpan snapshot input dan hasil kalkulasi |
| `reporting` | Dashboard, rekap, ekspor, read model | Read-only terhadap domain; tidak mengubah data domain |
| `portal` | Pengalaman klien B2B dan query terbatasi tenant | Secara teknis bisa menjadi area UI/API di atas domain yang ada |
| `audit` | Catatan tindakan penting dan perubahan sensitif | Append-only sejauh praktis; jangan hanya mengandalkan log aplikasi biasa |

### 4.1 Aturan kepemilikan data

- Setiap tabel mempunyai satu modul pemilik.
- Modul lain tidak boleh melakukan `INSERT`, `UPDATE`, atau `DELETE` langsung ke tabel milik modul lain.
- Jika perlu data lintas modul, gunakan public application service, query/read model terkontrol, atau event internal yang terdokumentasi.
- Hindari repository generik yang membuat semua modul dapat mengakses semua tabel.
- Query reporting lintas modul boleh dibuat melalui read model atau query layer khusus dengan akses baca yang disengaja dan terdokumentasi.
- Jika suatu use case membutuhkan transaksi atomik lintas modul, desain batas transaksi secara eksplisit. Jangan menyamarkan transaksi lintas domain sebagai event asynchronous bila hasilnya harus langsung konsisten.

### 4.2 Dependensi konseptual

```mermaid
flowchart TD
    MD[Master Data] --> ORD[Order]
    RATE[Rate Card] --> PROD[Production]
    RATE --> PAY[Payroll]
    SAMPLE[Sample] --> ORD
    ORD --> INV[Inventory]
    ORD --> PROD
    INV <--> PROD
    PROD --> QC[Quality]
    QC --> PAY
    PROD --> PAY
    MD --> SAMPLE
    MD --> RATE
    MD --> INV
    MD --> PROD
    AUD[Audit] -. cross-cutting .-> ORD
    AUD -. cross-cutting .-> INV
    AUD -. cross-cutting .-> PROD
    AUD -. cross-cutting .-> QC
    AUD -. cross-cutting .-> PAY
    REP[Reporting] -. read-only .-> ORD
    REP -. read-only .-> INV
    REP -. read-only .-> PROD
    REP -. read-only .-> QC
    REP -. read-only .-> PAY
```

Diagram ini adalah peta konseptual, bukan izin untuk membuat circular imports. Arah dependensi kode harus lebih ketat daripada hubungan bisnis. Modul harus menggunakan kontrak yang dimiliki modul penyedia, dan modul dasar tidak boleh mengimpor modul yang lebih tinggi.

---

## 5. Struktur Repository yang Direkomendasikan

Contoh struktur awal untuk satu repository:

```text
lukla-operations-engine/
├── cmd/
│   ├── api/
│   │   └── main.go
│   └── worker/                 # opsional; jangan dibuat sebelum dibutuhkan
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── app.go              # wiring dependencies
│   │   └── config.go
│   ├── platform/
│   │   ├── database/
│   │   │   ├── postgres.go
│   │   │   └── transaction.go
│   │   ├── httpx/
│   │   │   ├── response.go
│   │   │   └── errors.go
│   │   ├── logger/
│   │   │   └── logger.go
│   │   ├── authn/              # authentication middleware/helpers
│   │   ├── authz/              # authorization policies/helpers
│   │   ├── clock/
│   │   ├── id/
│   │   └── validation/
│   ├── modules/
│   │   ├── identity/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── model.go
│   │   │   └── routes.go
│   │   ├── masterdata/
│   │   ├── ratecard/
│   │   ├── sample/
│   │   ├── order/
│   │   ├── inventory/
│   │   ├── production/
│   │   ├── quality/
│   │   ├── payroll/
│   │   ├── reporting/
│   │   └── audit/
│   └── jobs/                   # pekerjaan terjadwal bila diperlukan
├── migrations/
│   ├── 000001_create_identity.up.sql
│   ├── 000001_create_identity.down.sql
│   ├── 000002_create_masterdata.up.sql
│   ├── 000002_create_masterdata.down.sql
│   └── ...
├── web/                        # frontend; alternatif: repository terpisah
│   ├── src/
│   ├── public/
│   ├── package.json
│   └── ...
├── docs/
│   ├── architecture.md
│   ├── api-conventions.md
│   ├── adr/
│   └── runbooks/
├── scripts/
├── .env.example
├── .gitignore
├── Dockerfile
├── compose.yaml                # local development
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

### 5.1 Catatan struktur

- `internal/` mencegah package aplikasi digunakan sembarangan dari luar module Go.
- `platform/` berisi komponen teknis bersama, bukan tempat menaruh semua fungsi yang sulit dikategorikan.
- Setiap modul boleh berkembang menjadi subfolder `domain/`, `application/`, `transport/http/`, dan `persistence/` jika kompleksitasnya benar-benar membutuhkan itu.
- Jangan membuat folder kosong untuk semua pola arsitektur hanya demi mengikuti template.
- Jika frontend dan backend dikerjakan terpisah dan membutuhkan siklus release berbeda, `web/` dapat menjadi repository terpisah. Keputusan monorepo vs multi-repo merupakan keputusan tim, bukan syarat modular monolith.

---

## 6. Pola Internal per Modul

Untuk modul yang kompleks, struktur berikut dapat digunakan:

```text
internal/modules/inventory/
├── model.go              # entity/value types domain
├── service.go            # use case dan aturan bisnis
├── repository.go         # interface penyimpanan milik modul
├── postgres_repository.go
├── handler.go            # HTTP handler, jika modul mengekspos REST
├── routes.go
├── dto.go                # request/response DTO
├── errors.go
└── service_test.go
```

### Tanggung jawab layer

- **Handler/transport:** membaca HTTP request, parsing, validasi bentuk input, memanggil service, memetakan error ke HTTP.
- **Service/application:** menjalankan use case dan aturan bisnis; tidak bergantung pada Gin.
- **Domain model:** tipe dan invariant bisnis yang relevan.
- **Repository:** query dan persistensi data yang dimiliki modul.
- **Platform:** koneksi database, logger, clock, ID generator, dan helper teknis lintas modul.

Untuk modul kecil, beberapa file dapat digabung. Yang penting adalah arah dependensinya, bukan jumlah file.

---

## 7. Konfigurasi Aplikasi dan Environment

### 7.1 Environment variables

Gunakan environment variables atau secret manager pada deployment. Jangan commit file `.env` berisi secret.

Contoh `.env.example`:

```dotenv
APP_ENV=development
APP_NAME=lukla-operations-engine
HTTP_ADDR=:8080
HTTP_READ_TIMEOUT=10s
HTTP_WRITE_TIMEOUT=30s
HTTP_IDLE_TIMEOUT=60s
HTTP_SHUTDOWN_TIMEOUT=15s

DATABASE_URL=postgres://lukla_app:change_me@localhost:5432/lukla?sslmode=disable
DB_MAX_OPEN_CONNS=20
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=30m
DB_CONN_MAX_IDLE_TIME=5m

LOG_LEVEL=debug
LOG_FORMAT=json

# Contoh placeholder; tentukan mekanisme session/token dan secret management sebelum produksi.
AUTH_ACCESS_TOKEN_TTL=15m
AUTH_REFRESH_TOKEN_TTL=168h
AUTH_TOKEN_SECRET=replace_with_local_only_secret

# Gunakan hanya bila implementasi RLS memerlukan konteks tenant/session.
TENANT_CONTEXT_MODE=transaction_local
```

**Penting:**
- `.env.example` berisi placeholder saja.
- `.env` harus masuk `.gitignore`.
- Production secrets harus disimpan di secret manager/platform environment yang aman.
- Jangan mencatat password, token, secret, atau data sensitif ke log.
- `sslmode=disable` hanya contoh lokal. Production harus menggunakan TLS sesuai konfigurasi penyedia database.
- TTL token di atas hanyalah contoh awal, bukan keputusan keamanan final.

### 7.2 Konfigurasi terpusat

Buat satu struct konfigurasi yang divalidasi saat startup. Aplikasi harus gagal startup jika konfigurasi wajib hilang atau tidak valid; jangan diam-diam memakai default berbahaya.

---

## 8. Koneksi Database PostgreSQL

### 8.1 Pilihan driver

Pilih satu pendekatan database access dan dokumentasikan:

- `pgx`/`pgxpool` untuk driver/pool PostgreSQL native; atau
- `database/sql` dengan driver PostgreSQL yang sesuai bila tim ingin memakai abstraksi standar Go.

Untuk implementasi baru yang khusus PostgreSQL, `pgxpool` merupakan pilihan yang masuk akal. Jika tim memilih `database/sql`, gunakan driver yang terpelihara dan konsisten. Jangan mencampur dua pool/driver tanpa alasan.

### 8.2 Aturan koneksi

- Buat pool sekali saat startup, bukan setiap request.
- Tetapkan batas koneksi sesuai kapasitas database dan jumlah instance aplikasi.
- Atur connection timeout, max lifetime, dan idle time.
- Gunakan context dengan deadline untuk query.
- Lakukan `Ping` saat startup.
- Tutup pool saat graceful shutdown.
- Jangan membuka satu koneksi global tanpa pool.
- Jangan membangun SQL dengan menggabungkan input pengguna; gunakan parameterized queries.
- Transaksi harus memiliki batas yang jelas dan rollback saat gagal.

### 8.3 Contoh `pgxpool`

Contoh konseptual; sesuaikan API dengan versi dependency yang dipilih dan tambahkan observability yang diperlukan.

```go
package database

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
    URL               string
    MaxConns          int32
    MinConns          int32
    MaxConnLifetime   time.Duration
    MaxConnIdleTime   time.Duration
    HealthCheckPeriod time.Duration
}

func NewPostgresPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
    parsed, err := pgxpool.ParseConfig(cfg.URL)
    if err != nil {
        return nil, fmt.Errorf("parse database config: %w", err)
    }

    if cfg.MaxConns > 0 {
        parsed.MaxConns = cfg.MaxConns
    }
    if cfg.MinConns >= 0 {
        parsed.MinConns = cfg.MinConns
    }
    if cfg.MaxConnLifetime > 0 {
        parsed.MaxConnLifetime = cfg.MaxConnLifetime
    }
    if cfg.MaxConnIdleTime > 0 {
        parsed.MaxConnIdleTime = cfg.MaxConnIdleTime
    }
    if cfg.HealthCheckPeriod > 0 {
        parsed.HealthCheckPeriod = cfg.HealthCheckPeriod
    }

    pool, err := pgxpool.NewWithConfig(ctx, parsed)
    if err != nil {
        return nil, fmt.Errorf("create database pool: %w", err)
    }

    pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    if err := pool.Ping(pingCtx); err != nil {
        pool.Close()
        return nil, fmt.Errorf("ping database: %w", err)
    }

    return pool, nil
}
```

Catatan:
- Jika `MinConns` tidak diset, tentukan perilaku default secara eksplisit dalam konfigurasi.
- Jangan mengasumsikan `MaxConns=20` selalu benar. Total koneksi dari seluruh instance aplikasi, worker, migration, dan admin harus berada dalam kapasitas PostgreSQL.
- Pool koneksi bukan pengganti pengelolaan transaksi.

### 8.4 Database roles

Gunakan user database berbeda sesuai kebutuhan:

- `lukla_migrator`: menjalankan migration, hanya digunakan pada deployment/migration job.
- `lukla_app`: hak minimum untuk operasi aplikasi.
- `lukla_readonly` (opsional): akses baca khusus reporting/analitik jika arsitektur membutuhkannya.

Aplikasi runtime tidak seharusnya memakai superuser atau role pemilik seluruh database. Hak DDL tidak diberikan ke role aplikasi runtime.

---

## 9. Transaksi Database

Gunakan transaksi untuk operasi yang harus berhasil atau gagal secara keseluruhan, contohnya:

- Validasi dan posting pengeluaran bahan.
- Verifikasi output produksi.
- Assembly dan perubahan status produk.
- Posting hasil inspeksi QC yang memicu perubahan stok/status.
- Finalisasi payroll dan snapshot kalkulasi.
- Perubahan status yang juga menulis audit record atau outbox event bila konsistensi atomik diwajibkan.

### Aturan implementasi

1. Validasi request di handler/service, tetapi ulangi invariant kritis pada level transaksi/database.
2. Mulai transaksi sedekat mungkin dengan bagian yang membutuhkan atomicity.
3. Ambil row lock atau gunakan conditional update bila dua pengguna dapat mengubah stok/output yang sama.
4. Periksa jumlah row yang terpengaruh; jangan menganggap update berhasil hanya karena tidak ada error SQL.
5. Commit hanya setelah seluruh langkah wajib berhasil.
6. Rollback saat error; jangan menelan error rollback/commit yang relevan.
7. Hindari memanggil API eksternal selama transaksi database terbuka.
8. Gunakan idempotency key atau constraint unik untuk mencegah submit ganda pada operasi berisiko.

### Contoh alur pengeluaran bahan

```text
Begin transaction
  -> load/lock inventory balance or eligible lots
  -> validate client_id and order_id match
  -> validate available quantity
  -> insert inventory movement
  -> update/derive stock balance consistently
  -> write audit record
Commit
```

Jika validasi gagal, transaksi rollback dan tidak boleh ada transaksi pengeluaran yang tercatat. Pilih satu sumber kebenaran stok—ledger pergerakan atau saldo materialized yang selalu diperbarui konsisten—dan dokumentasikan rekonsiliasinya.

---

## 10. Migration Strategy

### 10.1 Tujuan

Migration adalah perubahan skema database yang berversi, dapat direview, dapat dijalankan berulang secara aman sesuai tool, dan tercatat di deployment. Jangan mengandalkan perubahan manual di production.

### 10.2 Tool

Pilih satu migration tool, misalnya `golang-migrate/migrate` atau `pressly/goose`. Keputusan final ditentukan berdasarkan familiaritas tim, dukungan PostgreSQL, dan cara deployment. Jangan menggunakan beberapa tool migration sekaligus untuk database yang sama.

### 10.3 Penamaan file

```text
migrations/
├── 000001_create_identity.up.sql
├── 000001_create_identity.down.sql
├── 000002_create_masterdata.up.sql
├── 000002_create_masterdata.down.sql
├── 000003_create_orders.up.sql
└── 000003_create_orders.down.sql
```

Gunakan urutan konsisten. Satu migration sebaiknya mewakili satu perubahan yang dapat direview dengan jelas.

### 10.4 Aturan migration

- Semua perubahan schema masuk version control.
- Migration diuji di database kosong dan database yang sudah berisi versi sebelumnya.
- Migration production dijalankan sebagai langkah deployment terkontrol, bukan otomatis dari setiap replica API startup.
- Jalankan backup/snapshot sesuai risiko sebelum perubahan besar.
- Migration yang berpotensi lama atau mengunci tabel harus ditinjau khusus.
- Untuk perubahan yang memerlukan kompatibilitas antarversi aplikasi, gunakan pola expand-and-contract: tambah struktur kompatibel, deploy aplikasi, migrasikan data, baru hapus struktur lama pada deployment berikutnya.
- Jangan menganggap `down` selalu aman untuk data production; rollback schema yang menghapus data dapat menyebabkan kehilangan data.
- Seed data referensi dan data awal perlu dipisahkan dari migration schema atau memiliki strategi idempotent yang terdokumentasi.
- Jangan edit migration lama yang sudah diterapkan di lingkungan bersama; buat migration baru.

### 10.5 Urutan schema konseptual

Urutan persis mengikuti foreign key dan keputusan domain. Kandidat urutan:

1. Identity/access.
2. Master data dan tenant/client.
3. Rate card.
4. Sample dan order.
5. Inventory.
6. Production.
7. Quality.
8. Payroll/costing.
9. Audit/outbox/reporting support.

Audit atau tabel pendukung teknis bisa dibuat lebih awal bila dibutuhkan oleh migration modul awal.

### 10.6 Migration dan RLS

Jika RLS digunakan:
- Buat policy melalui migration yang direview.
- Tentukan role mana yang memiliki `BYPASSRLS` atau kepemilikan tabel; hindari memberikan kemampuan bypass kepada aplikasi runtime.
- Pertimbangkan `FORCE ROW LEVEL SECURITY` bila sesuai kebutuhan.
- Uji akses lintas tenant dengan role runtime sebenarnya, bukan hanya sebagai database owner.
- Konteks tenant harus di-set per transaksi menggunakan mekanisme transaction-local agar tidak bocor saat koneksi dipakai ulang oleh pool.
- Jangan menganggap RLS menggantikan validasi relasi `client_id` dan `order_id`.

---

## 11. Database Schema dan Konvensi

### 11.1 Konvensi umum

- Nama tabel dan kolom: `snake_case`.
- Primary key: pilih satu standar, misalnya UUID, dan konsisten.
- Waktu: simpan waktu absolut menggunakan `timestamptz`; tentukan timezone operasional untuk tampilan/periode payroll.
- Kuantitas dan uang: gunakan tipe numerik yang sesuai; hindari floating point untuk nilai uang.
- Kolom status: gunakan enum database hanya jika daftar status benar-benar stabil; jika status sering dikonfigurasi, tabel referensi atau constraint dapat lebih fleksibel.
- Foreign key dan unique constraint harus digunakan untuk invariant data yang bisa ditegakkan di database.
- Tambahkan indeks berdasarkan pola query nyata, bukan semua kolom.
- Tabel transaksi perlu metadata pelaku dan waktu yang relevan.
- Soft delete hanya digunakan bila kebutuhan bisnis jelas; untuk ledger dan payroll, koreksi biasanya lebih aman melalui reversal/adjustment daripada menghapus transaksi.

### 11.2 Tenant/client isolation

Untuk tabel tenant-scoped:
- `client_id` wajib dan tidak nullable jika data memang selalu milik satu tenant.
- `order_id` wajib pada item/transaksi yang oleh SRS harus selalu terkait order.
- Gunakan foreign key komposit bila diperlukan untuk menjamin order milik client yang sama.
- Data global harus dibedakan eksplisit dari data tenant.
- Semua query harus memiliki otorisasi yang benar; jangan mengandalkan nilai `client_id` dari request tanpa memeriksa identitas pengguna.
- Tentukan bagaimana data internal Snøhund direpresentasikan: sebagai tenant/client khusus atau tipe organisasi lain. Jangan mengunci keputusan sebelum disepakati.

### 11.3 Histori dan locking

- Rate card perlu histori versi dan tanggal berlaku.
- Order harus menyimpan snapshot spesifikasi/sample yang disetujui atau referensi versi yang immutable.
- Payroll final harus menyimpan snapshot input, versi aturan, parameter, dan hasil.
- Transaksi yang dikunci tidak boleh diedit langsung oleh UI atau endpoint umum.
- Koreksi dilakukan melalui alur khusus yang diaudit.

---

## 12. API Design

### 12.1 Konvensi REST

- Prefix versi: `/api/v1`.
- JSON request/response.
- HTTPS wajib di luar local development.
- Gunakan plural nouns untuk resource jika sesuai.
- Gunakan HTTP status yang konsisten.
- Pagination untuk list besar.
- Filter, sort, dan search memiliki allowlist field.
- Gunakan request ID/correlation ID untuk menelusuri error.
- Jangan membocorkan stack trace, query SQL, atau detail secret ke client.

Contoh endpoint konseptual:

```text
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout

GET    /api/v1/clients
GET    /api/v1/orders
POST   /api/v1/orders
GET    /api/v1/orders/{orderID}

GET    /api/v1/inventory/lots
POST   /api/v1/inventory/receipts
POST   /api/v1/inventory/issues

POST   /api/v1/production/output-records
POST   /api/v1/production/output-records/{id}/verify
POST   /api/v1/production/assemblies

POST   /api/v1/quality/inspections
POST   /api/v1/payroll/periods/{id}/calculate
POST   /api/v1/payroll/periods/{id}/lock
```

Endpoint di atas hanya ilustrasi. Nama, lifecycle, dan permission final harus mengikuti domain model dan persetujuan SRS.

### 12.2 Format error

Gunakan format konsisten, misalnya:

```json
{
  "error": {
    "code": "INVENTORY_CLIENT_ORDER_MISMATCH",
    "message": "Bahan tidak dapat dikeluarkan untuk order dari klien yang berbeda.",
    "request_id": "req_example"
  }
}
```

- `code` stabil untuk frontend dan automated test.
- `message` aman untuk ditampilkan kepada pengguna.
- Detail validasi boleh menyertakan daftar field error.
- Hindari menampilkan detail internal database.
- Pesan yang ditampilkan harus cukup eksplisit untuk membantu pengguna memperbaiki input.

### 12.3 API contract untuk frontend

- Dokumentasikan OpenAPI/Swagger atau spesifikasi API yang setara.
- Buat contoh request/response dan error.
- Sepakati format pagination, tanggal, uang, ID, dan status.
- Gunakan mock data berdasarkan contract yang sama.
- Jangan menggandakan aturan bisnis kritis hanya di frontend.

---

## 13. Authentication, Authorization, dan Security

### 13.1 Authentication

Pilih mekanisme autentikasi berdasarkan bentuk deployment dan kebutuhan browser. Bila memakai access token dan refresh token:
- Access token berumur pendek.
- Refresh token disimpan dan dirotasi/dicabut sesuai desain keamanan.
- Untuk aplikasi browser, evaluasi risiko penyimpanan token; cookie `HttpOnly`, `Secure`, dan `SameSite` sering lebih aman daripada menyimpan token jangka panjang di `localStorage`, tetapi harus dipasangkan dengan mitigasi CSRF yang tepat.
- Jangan menyimpan password plaintext; gunakan password hashing yang sesuai.
- Logout, deaktivasi akun, dan pencabutan sesi harus didefinisikan.
- Rate-limit endpoint login dan recovery.
- Gunakan secret manager dan prosedur rotasi secret.

### 13.2 Authorization

- Terapkan RBAC untuk peran yang disebutkan dalam SRS.
- Terapkan pemeriksaan scope/tenant pada setiap operasi yang menyentuh data tenant.
- Hak akses UI hanya untuk pengalaman pengguna; backend tetap sumber kebenaran.
- Buat permission granular untuk tindakan sensitif: verifikasi output, hasil QC final, ubah tarif, finalisasi payroll, buka kunci, dan konfigurasi role.
- Pisahkan siapa yang membuat, memeriksa, dan menyetujui bila proses bisnis memerlukan segregation of duties.
- Catat tindakan sensitif di audit trail.

### 13.3 RLS

RLS adalah pertahanan tambahan, bukan pengganti otorisasi aplikasi. Implementasi RLS harus diuji terhadap pool koneksi, transaksi, background jobs, dan role runtime. Bila konfigurasi tenant tidak aman, RLS bisa memberi rasa aman palsu.

### 13.4 Security baseline

- TLS di production.
- Validasi input di backend.
- Query parameterized.
- CORS dibatasi pada origin yang diperlukan.
- Proteksi CSRF jika autentikasi memakai cookie.
- Security headers.
- Rate limiting pada endpoint sensitif.
- Secret tidak masuk repository/log.
- Dependency scanning dan update terjadwal.
- Backup terenkripsi sesuai kemampuan penyedia.
- Uji authorization negatif, khususnya lintas klien.
- Audit perubahan tarif, payroll, role/permission, dan transaksi penting.

---

## 14. Audit Trail dan Business History

Audit trail bukan sekadar log debug. Untuk tindakan penting, simpan informasi terstruktur seperti:

- actor/user ID;
- waktu;
- action/event;
- jenis dan ID entitas;
- tenant/client context bila relevan;
- request/correlation ID;
- alasan perubahan bila dibutuhkan;
- perubahan sebelum/sesudah untuk atribut sensitif, dengan pertimbangan privasi;
- sumber tindakan (UI, job, integrasi) bila relevan.

### Aturan

- Audit tidak boleh diedit oleh pengguna aplikasi biasa.
- Hindari mencatat password, token, atau informasi rahasia.
- Audit bisnis dan application log mempunyai tujuan berbeda.
- Tentukan retensi dan kebijakan akses audit.
- Jika audit record harus atomik dengan transaksi bisnis, tulis dalam transaksi yang sama.
- Untuk histori payroll/rate card, gunakan model histori domain yang sesuai, bukan hanya audit JSON generik.

---

## 15. Internal Events dan Background Jobs

### 15.1 Default awal

Gunakan pemanggilan service langsung untuk proses yang harus konsisten dan segera selesai. Jangan membuat event untuk setiap perubahan CRUD.

### 15.2 Event kandidat

Event internal hanya dibuat jika ada kebutuhan lintas modul yang jelas, misalnya:

- `OutputVerified`
- `ProductAssembled`
- `InspectionPassed`
- `PayrollPeriodLocked`

Nama event dan payload final harus didefinisikan bersama pemilik domain. Event harus berisi ID referensi dan metadata minimum, bukan menyalin seluruh objek tanpa kebutuhan.

### 15.3 Reliable delivery

Jika suatu event wajib diproses walaupun proses aplikasi berhenti, pertimbangkan transactional outbox:
- tulis perubahan domain dan outbox record dalam transaksi database yang sama;
- dispatcher membaca event yang belum diproses;
- pemrosesan event harus idempotent;
- simpan status retry/error;
- hindari asumsi exactly-once delivery; desain untuk at-least-once delivery dan deduplikasi.

Outbox/worker tidak perlu ditambahkan sebelum ada use case yang membutuhkannya. Jangan memanggil sistem eksternal selama transaksi bisnis terbuka.

### 15.4 Scheduled jobs

Untuk pekerjaan terjadwal seperti rekap atau pengingat, tentukan:
- timezone;
- perilaku saat job terlambat;
- lock agar job tidak berjalan ganda pada beberapa instance;
- retry dan observability;
- idempotency;
- audit hasil job.

Payroll Sabtu tidak boleh otomatis final tanpa aturan persetujuan bisnis yang eksplisit.

---

## 16. Payroll, Rate Card, dan Perhitungan Biaya

### 16.1 Rate card

SRS belum memastikan apakah rate card adalah tarif komersial ke klien, tarif upah penjahit, atau keduanya. Sampai dikonfirmasi, desain harus menjaga kemungkinan memisahkan:
- tarif komersial;
- tarif tenaga kerja/borongan;
- tarif bonus QC;
- aturan lain yang dapat dikonfigurasi.

Setiap rate perlu memiliki konteks yang jelas, misalnya artikel/komponen/klien, satuan, tanggal mulai berlaku, tanggal akhir bila ada, pembuat, dan histori.

### 16.2 Snapshot payroll

Payroll yang dihitung perlu menyimpan:
- periode dan status;
- pekerja yang termasuk;
- output terverifikasi yang digunakan;
- rate dan versi aturan yang digunakan;
- uang makan dan parameter periode;
- komponen bonus QC;
- basis overhead;
- hasil per pekerja dan ringkasan;
- waktu perhitungan, pembuat, approver, waktu lock;
- koreksi/adjustment setelah lock.

Jangan menghitung ulang payroll yang telah dikunci dengan mengambil rate terbaru. Recalculation harus mempunyai aturan versi dan persetujuan yang jelas.

### 16.3 Idempotency dan lock

- Satu periode tidak boleh menghasilkan pembayaran ganda karena tombol ditekan berulang.
- Finalisasi perlu transaksi dan concurrency control.
- Setelah dikunci, perubahan dilakukan melalui adjustment yang diaudit.
- Definisikan siapa yang boleh calculate, approve, lock, dan reopen.
- Definisikan apakah reopen diizinkan, oleh siapa, dan bagaimana histori tetap dipertahankan.

### 16.4 Pertanyaan bisnis wajib

- Kapan rate baru berlaku?
- Apakah perubahan rate berlaku retroaktif?
- Apa definisi output yang dapat dibayar?
- Bagaimana output repair/rework diperlakukan?
- Kapan produk dianggap menghasilkan bonus QC?
- Bagaimana pembagian bonus jika penerima berubah atau tidak aktif?
- Apa basis uang makan 10–15% dan siapa yang menentukan persentasenya?
- Unit mana yang masuk pembagi overhead bulanan?
- Apa kebijakan koreksi payroll setelah lock?

---

## 17. Inventory dan Production Integrity

### 17.1 Inventory ledger

Pertimbangkan ledger pergerakan inventaris sebagai catatan transaksi utama:
- receipt;
- issue;
- transfer;
- return;
- scrap/waste;
- stock adjustment;
- reservation/release bila diperlukan.

Saldo stok dapat dihitung dari ledger atau disimpan sebagai saldo materialized yang diperbarui secara atomik. Jika saldo disimpan, sediakan prosedur rekonsiliasi dan constraint/locking yang memadai.

### 17.2 Validasi anti-kontaminasi silang

Sebelum pengeluaran:
1. Identifikasi lot/item yang akan dikeluarkan.
2. Pastikan `client_id` dan `order_id` sesuai aturan bisnis.
3. Pastikan jumlah tersedia dan tidak telah direservasi/dikeluarkan.
4. Lakukan operasi dengan locking/conditional update untuk menghindari race condition.
5. Catat movement dan audit secara atomik.
6. Jika tidak cocok, tolak tanpa mencatat issue yang gagal sebagai transaksi stok sukses. Kegagalan dapat dicatat di security/application log atau audit event terpisah jika kebijakan mengharuskan.

### 17.3 Production output

- Catatan output mempunyai pekerja, line/order/komponen, interval, kuantitas, dan status verifikasi sesuai model bisnis.
- Verifikasi harus mencegah perubahan diam-diam pada catatan yang telah disetujui.
- Koreksi setelah verifikasi harus melalui prosedur koreksi yang diaudit.
- Definisikan idempotency/deduplication untuk input berulang.
- Tentukan apakah satu pekerja dapat dialokasikan ke banyak line pada interval yang sama dan aturan kapasitasnya.

---

## 18. Frontend Architecture

Contoh struktur Next.js:

```text
web/src/
├── app/
│   ├── (auth)/
│   ├── (dashboard)/
│   ├── portal/
│   └── api/                  # hanya jika benar-benar diperlukan; backend Go tetap API utama
├── components/
│   ├── layout/
│   ├── data-display/
│   ├── forms/
│   └── feedback/
├── features/
│   ├── inventory/
│   ├── production/
│   ├── quality/
│   ├── payroll/
│   ├── orders/
│   └── reporting/
├── lib/
│   ├── api-client.ts
│   ├── auth.ts
│   ├── formatters.ts
│   └── validators.ts
├── types/
└── styles/
```

### Aturan frontend

- Organisasikan fitur berdasarkan domain bisnis.
- Pisahkan API client dari komponen visual.
- Jangan menaruh aturan bisnis otoritatif di React component.
- Tampilkan loading, empty, validation, success, dan error state dengan konsisten.
- Gunakan permission untuk menyembunyikan tindakan yang tidak relevan, tetapi jangan menganggapnya sebagai security boundary.
- Terminal produksi/QC perlu diuji pada ukuran layar, input sentuh, dan skenario penggunaan cepat.
- Tabel besar harus memiliki pagination/filter server-side bila diperlukan.
- Hindari state management library tambahan sebelum kompleksitas state membutuhkannya.
- Jangan menyimpan data tenant sensitif secara persisten di browser tanpa kebutuhan yang jelas.

---

## 19. Utilities dan Shared Helpers

Utilities harus kecil, generik, dan bebas aturan bisnis domain. Contoh yang masuk akal:

| Utility | Fungsi |
|---|---|
| `clock` | Abstraksi waktu agar periode dan test deterministik |
| `id` | Pembuatan/validasi ID sesuai standar |
| `validation` | Validasi bentuk input yang reusable, bukan pengganti validasi domain |
| `httpx` | Format response dan error HTTP konsisten |
| `logger` | Structured logging dan redaction |
| `database` | Pool, transaksi, health check |
| `authn` | Identifikasi pengguna/sesi |
| `authz` | Evaluasi permission/policy |
| `pagination` | Parsing pagination yang konsisten bila benar-benar dipakai lintas modul |
| `money` | Tipe/operasi nilai uang bila membantu menghindari floating point |

Jangan membuat `utils` menjadi tempat menumpuk fungsi yang tidak diketahui pemiliknya. Fungsi khusus domain tetap tinggal di modul domain. Contohnya validasi kecocokan `client_id` dan `order_id` merupakan aturan Inventory, bukan generic utility.

---

## 20. Error Handling, Logging, dan Observability

### 20.1 Error handling

- Definisikan error domain yang dapat dipetakan ke kode API.
- Jangan membandingkan string pesan error untuk menentukan perilaku.
- Bedakan validation error, not found, conflict, forbidden, dan internal error.
- Log detail teknis di server dengan redaction; respons client tetap aman.
- Sertakan request ID untuk korelasi.

### 20.2 Logging

Gunakan structured logs, idealnya JSON di production. Field umum:
- timestamp;
- level;
- service/app;
- request ID;
- user ID jika sesuai kebijakan;
- route/method;
- duration;
- error code.

Jangan log body request penuh secara default, terutama untuk autentikasi, payroll, atau data klien.

### 20.3 Health checks

- Liveness: proses aplikasi berjalan.
- Readiness: aplikasi siap menerima traffic dan dependency kritis tersedia sesuai kebijakan.
- Database health check tidak boleh membuat endpoint health menjadi query berat.
- Jangan membocorkan detail infrastruktur ke endpoint publik.

### 20.4 Metrics dan alerting

Mulai dengan metrik yang dapat ditindaklanjuti:
- request rate, latency, error rate;
- kegagalan koneksi/query database;
- job/outbox backlog bila ada;
- error finalisasi payroll;
- kegagalan backup;
- disk/CPU/memory sesuai hosting.

Alert harus diarahkan kepada pihak yang bertanggung jawab dan memiliki runbook.

---

## 21. Testing Strategy

### 21.1 Unit tests

Uji aturan domain tanpa database, misalnya:
- perhitungan upah;
- pemilihan rate berdasarkan tanggal berlaku;
- transisi status QC;
- aturan uang makan/bonus/overhead;
- permission policy yang dapat diuji terpisah.

### 21.2 Integration tests

Gunakan PostgreSQL nyata untuk menguji:
- migrations;
- foreign key/unique/check constraints;
- transaksi rollback;
- locking dan concurrent updates;
- RLS dengan role aplikasi runtime;
- query repository;
- atomicity antara transaksi bisnis dan audit/outbox.

Jangan mengandalkan mock database untuk membuktikan perilaku PostgreSQL.

### 21.3 API tests

Uji:
- authentication;
- authorization per role;
- akses lintas tenant;
- format error;
- pagination/filter;
- duplicate submission/idempotency;
- status transition yang dilarang.

### 21.4 Acceptance tests dari SRS

Minimal skenario:
1. Bahan klien A tidak dapat dikeluarkan untuk order klien B.
2. Tambah penjahit di tengah minggu dan output terverifikasi dihitung dengan benar pada payroll.
3. Pool bonus QC dihitung dan dibagi sesuai aturan final yang telah dikonfirmasi.
4. Sample v1 direvisi ke v2, v2 disetujui, dan order massal merujuk spesifikasi yang benar.
5. User klien A tidak dapat membaca/mengubah data klien B.
6. Payroll yang dikunci tidak dapat diedit lewat API langsung.
7. Submit berulang tidak menggandakan transaksi.
8. Restore backup dapat dilakukan dan hasilnya diverifikasi.

---

## 22. CI/CD dan Workflow Developer

### 22.1 Backend pipeline

Minimal:
1. `gofmt` check.
2. `go vet`.
3. Unit tests.
4. Integration tests dengan PostgreSQL.
5. Build binary.
6. Dependency/security checks yang sesuai.
7. Build container image.
8. Deployment ke staging.
9. Smoke tests.
10. Approval/deployment production sesuai proses tim.

### 22.2 Frontend pipeline

Minimal:
1. Format/lint.
2. Type check.
3. Unit/component tests yang tersedia.
4. Production build.
5. Dependency/security checks.
6. Deploy staging.
7. Smoke/E2E tests untuk alur kritis bila tersedia.

### 22.3 Branch dan review

- Semua perubahan schema dan business rule direview.
- Perubahan authorization, tenant isolation, payroll, dan inventory perlu reviewer tambahan bila memungkinkan.
- Jangan commit secret.
- Migration harus diuji bersama versi aplikasi yang akan dirilis.
- Gunakan pull request kecil agar mudah direview oleh tim kecil.

---

## 23. Deployment dan Infrastruktur

### 23.1 Baseline deployment cloud

Konfigurasi minimum yang perlu dipilih:
- tempat menjalankan backend Go;
- hosting Next.js (platform yang mendukung kebutuhan rendering aplikasi) atau static/frontend hosting jika cocok;
- PostgreSQL managed atau server database yang dikelola;
- TLS/domain;
- secret management;
- backup otomatis;
- log/metrics;
- staging environment;
- akses administratif terbatas.

Managed PostgreSQL dapat mengurangi beban patching dan backup operasional, tetapi biaya, lokasi data, konektivitas, dan prosedur restore harus dievaluasi.

### 23.2 Environment

Pisahkan:
- local development;
- test/CI;
- staging;
- production.

Jangan menggunakan database production untuk pengembangan. Hindari menyalin data personal/finansial production ke staging tanpa masking dan izin.

### 23.3 Container dan local development

Docker/Compose dapat menyederhanakan setup lokal, misalnya backend, PostgreSQL, dan frontend. Compose untuk development bukan berarti production harus menggunakan orkestrasi kompleks.

### 23.4 Backup dan disaster recovery

- Tetapkan RPO (berapa banyak data yang dapat diterima hilang) dan RTO (berapa lama sistem boleh tidak tersedia) bersama pemilik bisnis.
- Aktifkan backup sesuai kemampuan database/penyedia.
- Batasi akses dan enkripsi backup.
- Tentukan retensi.
- Uji restore berkala.
- Catat siapa yang bertanggung jawab saat pemulihan.
- Jangan menyatakan backup memadai sebelum restore pernah diuji.

---

## 24. Cloud vs Offline

Cloud adalah kandidat awal, tetapi kelayakannya bergantung pada konektivitas di pabrik.

### Jika koneksi stabil

- Terminal mengirim request ke API cloud.
- Gunakan timeout dan feedback yang jelas.
- Hindari retry otomatis untuk transaksi non-idempotent tanpa idempotency key.
- Monitor latency dan ketersediaan jaringan.

### Jika koneksi sering terputus

Sebelum membuat offline mode, tentukan:
- tindakan apa yang boleh dilakukan offline;
- perangkat mana yang menjadi sumber data lokal;
- strategi ID dan idempotency;
- bagaimana konflik diselesaikan;
- bagaimana mencegah stok/output/payroll terposting ganda;
- apakah data lokal harus dienkripsi;
- bagaimana sinkronisasi diaudit.

Offline-first untuk inventaris dan payroll dapat menambah kompleksitas besar. Jangan menjanjikan mode offline sebelum requirement dan risiko operasional disetujui. Alternatifnya dapat berupa koneksi cadangan, misalnya jalur internet kedua, jika lebih murah dan cukup andal.

---

## 25. Keputusan Arsitektur (ADR Ringkas)

| ID | Keputusan awal | Status | Alasan |
|---|---|---|---|
| ADR-001 | Modular monolith | Proposed | Konsistensi transaksi dan kapasitas tim kecil |
| ADR-002 | Go + Gin backend | Proposed | Sesuai rencana tim |
| ADR-003 | Next.js + TypeScript + Ant Design | Proposed | Sesuai rencana tim dan kebutuhan dashboard |
| ADR-004 | PostgreSQL | Proposed | Transaksi relasional, constraint, dan opsi RLS |
| ADR-005 | REST API | Proposed | Kontrak sederhana untuk frontend/backend terpisah |
| ADR-006 | Tidak ada microservices awal | Proposed | Belum ada kebutuhan operasional yang membenarkan kompleksitas |
| ADR-007 | Tidak ada broker awal | Proposed | Internal calls cukup untuk sebagian besar alur awal |
| ADR-008 | Migration berversi | Proposed | Perubahan database dapat direview dan dilacak |
| ADR-009 | Cloud sebagai kandidat deployment | Tentative | Menunggu validasi jaringan, biaya, dan offline |
| ADR-010 | RLS sebagai defense-in-depth | Under design | Perlu rancangan role/konteks tenant dan integration tests |
| ADR-011 | Transactional outbox | Conditional | Dipakai jika event harus reliable lintas proses |
| ADR-012 | Pemisahan schema per modul | Open | Bandingkan manfaat ownership dengan biaya migrasi/operasi |

Status `Proposed` berarti rekomendasi untuk disepakati, bukan keputusan yang sudah disetujui stakeholder.

---

## 26. Pertanyaan Terbuka / Decision Log

Pertanyaan ini harus dijawab atau diberi keputusan sementara sebelum modul terkait diimplementasikan.

### Bisnis

- [ ] Rate card: tarif klien, upah penjahit, atau keduanya?
- [ ] Tanggal berlaku tarif dan kebijakan perubahan retroaktif?
- [ ] Definisi final status QC: pass, repair, reject, scrap, dan alur inspeksi ulang?
- [ ] Output repair/rework masuk basis upah atau bonus?
- [ ] Siapa Kepala Operasional dan bagaimana perannya pada RBAC?
- [ ] Basis dan persentase uang makan; siapa yang boleh mengubahnya?
- [ ] Basis pool bonus QC dan kondisi produk dianggap eligible?
- [ ] Unit apa yang masuk pembagi overhead?
- [ ] Kapan payroll dikunci; siapa yang boleh approve/reopen; bagaimana adjustment?
- [ ] Apakah PO bahan, invoice, dan fulfillment termasuk scope rilis pertama?
- [ ] Bagaimana Snøhund direpresentasikan dalam model tenant?
- [ ] Bagaimana alokasi penjahit multi-line dan batas overlap waktunya?

### Infrastruktur/operasional

- [ ] Apakah internet pabrik stabil selama jam operasional?
- [ ] Apakah input produksi harus tetap berjalan saat koneksi putus?
- [ ] Cloud provider, region, budget, dan siapa pengelola deployment?
- [ ] Target RPO/RTO dan retensi backup?
- [ ] Apakah ada kebutuhan integrasi printer, ekspor, email, atau sistem lain?
- [ ] Berapa lama data transaksi/audit harus disimpan?
- [ ] Apakah frontend/backend satu repository atau dua repository?
- [ ] Siapa yang menjadi reviewer untuk perubahan kritis saat hanya 1–2 developer aktif?

### Teknologi

- [v] Menggunakan pgx/v5/stdlib.
- [v] Menggunakan golang-migrate.
- [v] Putuskan strategi authentication/session dan CSRF sesuai deployment:
    - frontend mengirim username/email dan password melalui https
    - backend verifikasi password hash, status akun, dan hak akses
    - membuat session baru; generate id baru, simpan record session, kirim cookie httponly
    - setiap request API perlu 
        - validasi session (authentication)
        - status user (masih aktif?)
        - permission (authorization)
        - tenant scope (data dalam cangkupan user yang diizinkan)
        - business operation

- [v] Putuskan apakah menggunakan RLS pada rilis awal setelah threat model dan test tersedia.
    - untuk saat ini di rilis awal tidak menggunakan RLS
- [v] Putuskan apakah butuh worker/outbox setelah use case event reliable terdefinisi.
    - tidak perlu worker/outbox untuk saat ini karena berada dalam monolith
- [v] Tentukan observability dan error tracking sesuai budget.
    - log terstruktur dengan isi:
        - ID permintaan
        - metrik dasar
        - pelacakan error
    - endpoint kesehatan (/health)
    - lingkungan: dev, staging, production.

---

## 27. Rencana Implementasi Bertahap

### Phase 0 — Discovery dan desain

- Konfirmasi pertanyaan bisnis kritis.
- Setujui arsitektur dan ADR.
- Definisikan API conventions.
- Buat domain glossary dan lifecycle status.
- Rancang threat model tenant isolation.
- Definisikan skenario acceptance test.

### Phase 1 — Foundation

- Repo dan build pipeline.
- Konfigurasi aplikasi dan structured logging.
- PostgreSQL connection pool.
- Migration tool dan database roles.
- HTTP server, middleware, error response.
- Authentication/authorization dasar.
- Audit foundation.
- Test harness dan staging.

### Phase 2 — Vertical slice pertama

Pilih satu alur end-to-end dengan nilai bisnis tinggi, misalnya:
- master client/order;
- penerimaan dan pengeluaran bahan;
- validasi client/order;
- output produksi;
- verifikasi mandor;
- audit dan laporan sederhana.

Jangan menganggap urutan tersebut final sebelum discovery; pilih alur yang paling menguji risiko inti.

### Phase 3 — Sample, QC, dan payroll

- Versioning sample/time study.
- Assembly dan alur QC.
- Finalisasi definisi tarif.
- Kalkulasi payroll dengan snapshot.
- Locking dan adjustment.
- Uji acceptance untuk skenario finansial.

### Phase 4 — Reporting, portal, dan hardening

- Dashboard internal.
- Portal klien dengan isolasi tenant.
- Ekspor laporan.
- Performance testing berdasarkan pola penggunaan nyata.
- Backup restore drill.
- Security review.
- UAT dan pelatihan pengguna.

Fase dapat disusun ulang setelah scope dan dependensi bisnis disetujui.

---

## 28. Definition of Done Minimum

Fitur belum dianggap selesai hanya karena endpoint atau UI sudah bekerja pada happy path.

- [ ] Acceptance criteria disetujui.
- [ ] Validasi backend untuk input dan aturan bisnis.
- [ ] Permission dan tenant scope diuji.
- [ ] Transaksi/constraint database sesuai invariant.
- [ ] Error response konsisten.
- [ ] Audit record tersedia untuk tindakan sensitif.
- [ ] Unit/integration/API tests relevan lulus.
- [ ] Tidak ada secret di source code/log.
- [ ] Dokumentasi API diperbarui.
- [ ] Migration diuji dan direview.
- [ ] UI mempunyai loading/error/empty state.
- [ ] Log dan request ID membantu debugging.
- [ ] Perubahan siap deploy dan memiliki cara rollback/mitigasi yang jelas.

---

## 29. Catatan Penutup

Dokumen ini merupakan baseline arsitektur, bukan spesifikasi implementasi final. Bagian yang bergantung pada keputusan bisnis—khususnya rate card, status QC, payroll, locking, tenant model, dan offline—harus diselesaikan sebelum ERD dan kode terkait dikunci.

Urutan kerja yang disarankan:

1. Review dokumen dengan partner.
2. Catat keputusan dan pertanyaan terbuka.
3. Finalisasi domain model dan aturan transaksi kritis.
4. Buat ERD dan migration plan.
5. Definisikan API contract.
6. Implementasikan foundation dan satu vertical slice.
7. Perluas fitur berdasarkan hasil uji dan umpan balik pengguna.
