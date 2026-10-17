# Panduan Cepat Deployment ke AWS EC2 (Hackathon)

Panduan ini disiapkan agar proses deployment ke instance EC2 dapat diselesaikan dalam waktu kurang dari 3 menit menggunakan Docker Compose.

---

## 1. Prasyarat di Instance EC2

Pilih instruksi instalasi sesuai Sistem Operasi EC2 yang Anda gunakan:

### 🔹 Opsi A: Amazon Linux 2023 / AL2 (Default AWS `ec2-user`) — *Direkomendasikan*

Jalankan perintah berikut di terminal EC2:

```bash
# 1. Update sistem dan install Docker & Git
sudo dnf update -y
sudo dnf install -y docker git

# 2. Nyalakan dan aktifkan service Docker saat boot
sudo systemctl enable --now docker

# 3. Berikan izin docker ke ec2-user (tanpa perlu sudo)
sudo usermod -aG docker ec2-user
newgrp docker

# 4. Pasang plugin Docker Compose & Docker Buildx resmi
sudo mkdir -p /usr/local/lib/docker/cli-plugins

# Docker Compose
sudo curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64 -o /usr/local/lib/docker/cli-plugins/docker-compose
sudo chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

# Docker Buildx (wajib untuk compose build)
sudo curl -SL "https://github.com/docker/buildx/releases/download/v0.18.0/buildx-v0.18.0.linux-amd64" -o /usr/local/lib/docker/cli-plugins/docker-buildx
sudo chmod +x /usr/local/lib/docker/cli-plugins/docker-buildx

# 5. Verifikasi instalasi
docker compose version
docker buildx version
```

---

### 🔹 Opsi B: Ubuntu / Debian (`ubuntu` user)

```bash
sudo apt-get update -y
sudo apt-get install -y docker.io docker-compose-v2 git
sudo usermod -aG docker $USER
newgrp docker
```

---

### 1.1 Optimasi Memori untuk `t3.micro` (Swapfile 2GB)
Khusus untuk instance `t3.micro` (RAM 1 GB), sangat disarankan mengaktifkan Swap Memory 2GB agar proses kompilasi container dan database berjalan lancar tanpa risiko kehabisan memori (OOM):

```bash
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

> 💡 **Tips:** Jika terminal menampilkan pesan `fallocate failed: Text file busy`, ketik `free -h`. Jika baris `Swap:` sudah menunjukkan angka (~2.0Gi), artinya swap memory sudah aktif dan Anda bisa langsung melanjutkan ke langkah berikutnya.

Pastikan Security Group EC2 membuka inbound port:
- Port `22` (SSH)
- Port `80` (HTTP - Akses API publik jika menggunakan APP_PORT=80)
- Port `8080` (Akses API backend default)

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

# Test endpoint health (sesuaikan dengan APP_PORT yang Anda pilih di .env)
curl http://localhost:80/health
# atau jika memakai port 8080: curl http://localhost:8080/health
```

Jika sukses, API Anda kini siap diakses dari publik melalui:
- **Port 80:** `http://<PUBLIC_IP_EC2>/health` (langsung tanpa port)
- **Port 8080:** `http://<PUBLIC_IP_EC2>:8080/health`
