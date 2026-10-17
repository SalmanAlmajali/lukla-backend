# Lukla Operations Engine — Development & Design Materials

Folder ini mengumpulkan seluruh aset perancangan, kontrak API, dokumentasi arsitektur, dan koleksi pengujian yang digunakan selama proses pengembangan dan evaluasi hackathon.

---

## Struktur Folder

```text
docs/
├── openapi/                 # Spesifikasi kontrak OpenAPI (v3.0.3) per fase rilis
│   ├── openapi-phase0-fondasi.yaml
│   ├── openapi-phase1-produksi.yaml
│   ├── openapi-phase2-payroll.yaml
│   ├── openapi-phase3-gudang.yaml
│   ├── openapi-phase4-qc-bonus.yaml
│   ├── openapi-phase5-portal-klien.yaml
│   ├── openapi-phase6-sample-time-study.yaml
│   └── openapi-phase7-overhead-e2e.yaml
│
├── postman/                 # Koleksi pengujian endpoint otomatis via Postman
│   └── Lukla_Articles_and_Components.postman_collection.json
│
├── diagrams/                # Diagram alur proses bisnis manufaktur
│   ├── Lukla_BPMN_v2.drawio
│   └── diagram2_gudang.xml
│
├── architecture/            # Panduan teknis dan catatan arsitektur
│   ├── Lukla_Operations_Engine_Architecture.md
│   ├── GIN_BACKEND_STARTER_GUIDE.md
│   ├── Pertanyaan_untuk_Client_Lukla.md
│   └── urutan pengerjaan.txt
│
└── history-pengerjaan/      # Log implementasi bertahap (Tahap 2 s.d. 7)
```

---

## Panduan Penggunaan untuk Pengembang (Dev)

1. **API Contracts:**  
   Rujuk file di [openapi/](file:///c:/go/lukla-backend/docs/openapi/) untuk payload request, response DTO, dan response code saat integrasi dengan frontend Next.js.
2. **Postman Collection:**  
   Import file di [postman/](file:///c:/go/lukla-backend/docs/postman/) ke Postman lokal untuk validasi alur pembuatan artikel, komponen, dan otorisasi role.
3. **Arsitektur & Aturan Bisnis:**  
   Buka [architecture/Lukla_Operations_Engine_Architecture.md](file:///c:/go/lukla-backend/docs/architecture/Lukla_Operations_Engine_Architecture.md) untuk memahami modul modular monolith, aturan anti-kontaminasi silang gudang, locking payroll, dan integrasi Amazon Bedrock.
