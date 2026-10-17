# Panduan Cepat Deployment ke AWS EC2 (Hackathon)

Panduan ini disiapkan agar proses deployment ke instance EC2 dapat diselesaikan dalam waktu kurang dari 3 menit menggunakan Docker Compose.

---

## 1. Prasyarat di Instance EC2

Pastikan instance EC2 Anda sudah terpasang Git, Docker, dan Docker Compose:

```bash
# Update package list
sudo apt-get update -y

# Install Docker & Compose plugin (Ubuntu/Debian)
sudo apt-get install -y docker.io docker-compose-v2 git

# Berikan izin docker ke user ubuntu (opsional agar tidak perlu sudo terus menerus)
sudo usermod -aG docker $USER
newgrp docker
```

Pastikan Security Group EC2 membuka inbound port:
- Port `22` (SSH)
- Port `8080` (Akses API backend Go)

---

## 2. Clone Repositori di EC2

Jalankan perintah ini di dalam server EC2:

```bash
# Clone branch main (versi stabil production)
git clone -b main <URL_REPOSITORY_ANDA> lukla-backend
cd lukla-backend/deploy
```

---

## 3. Siapkan Konfigurasi Lingkungan

Salin template environment produksi:

```bash
cp .env.production.example .env
```

*(Opsional)* Ubah `POSTGRES_PASSWORD` atau `AUTH_TOKEN_SECRET` jika ingin menggunakan kata sandi khusus:
```bash
nano .env
```

> **Catatan IAM Role:** Karena instance EC2 telah memiliki IAM Role dari panitia, Anda **TIDAK PERLU** memasukkan `AWS_ACCESS_KEY_ID` atau `AWS_SECRET_ACCESS_KEY`. Backend Go otomatis mengotentikasi request S3 dan Bedrock melalui instance profile!

---

## 4. Jalankan Service

Cukup jalankan satu perintah:

```bash
docker compose up -d --build
```

**Alur otomatis yang akan berjalan:**
1. Container `lukla-postgres` aktif dan siap menerima koneksi.
2. Container `lukla-migrate` otomatis menjalankan seluruh 34 file migrasi database dari folder `migrations/`.
3. Container `lukla-backend` otomatis di-build secara multi-stage (ringan ~25MB) dan berjalan di port `8080`.

---

## 5. Verifikasi Deployment

Uji apakah backend dan koneksi database sudah hidup:

```bash
# Cek status container
docker compose ps

# Cek log aplikasi
docker compose logs -f backend

# Test endpoint health
curl http://localhost:8080/health
```

Jika sukses, API Anda kini siap diakses dari publik melalui:
`http://<PUBLIC_IP_EC2>:8080`
