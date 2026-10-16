package aiagent

import (
	"strings"
)

// AllowedDomainKeywords contains vocabulary related to Lukla manufacturing operations.
var AllowedDomainKeywords = []string{
	// Pabrik & Lokasi
	"lukla", "ciwidey", "pabrik", "fasilitas", "manufaktur", "atelier",
	// Produk & Garmen
	"garmen", "jaket", "parka", "windbreaker", "tas", "carrier", "alpine", "snøhund", "snohund", "artikel", "tech pack", "bom",
	// Produksi & Line
	"line", "lini", "jahit", "sewing", "potong", "cutting", "finishing", "bartack", "output", "kapasitas", "shift", "operator", "mandor", "spk", "batch",
	// Quality Control
	"qc", "kualitas", "defect", "cacat", "reject", "repair", "rework", "inspeksi", "stiker", "panah", "lolos", "loncat", "poka-yoke", "poka yoke",
	// Bahan & Gudang
	"bahan", "kain", "taslan", "gore-tex", "goretex", "cordura", "fleece", "waterproof", "resleting", "zipper", "ykk", "benang", "webbing", "buckle", "duraflex", "seam", "tape", "rak", "gudang", "inventory", "inventaris", "stok", "supplier",
	// Biaya & Waktu
	"sam", "time study", "stopwatch", "menit", "durasi", "standar", "rate card", "upah", "borongan", "payroll", "gaji", "overhead", "biaya", "harga", "faktur", "invoice", "po", "purchase order",
	// Pesanan & Klien
	"pesanan", "order", "klien", "client", "portal", "target", "deadline", "progres", "status", "rekomendasi", "analisa", "analisis", "performa", "statistik", "eksekutif",
	// RAG & Regulasi
	"srs", "bpmn", "sop", "aturan", "toleransi", "spesifikasi", "alur",
	// Greetings & Asisten
	"halo", "hai", "hello", "hi", "pagi", "siang", "sore", "malam", "siapa", "kamu", "bantuan", "bantu", "fitur", "tolong", "terima kasih", "thanks",
}

// StrictRefusalMessage is the standard polite refusal response when out-of-scope questions are posed.
const StrictRefusalMessage = "Maaf, sebagai AI Copilot Operasional Pabrik Lukla Ciwidey, saya hanya berwenang menjawab pertanyaan dan analisis seputar operasional manufaktur, pelacakan pesanan, inventaris bahan, performa line jahit, kontrol kualitas (QC), standar SAM, serta dokumen spesifikasi SRS pabrik Lukla.\n\nSilakan ajukan pertanyaan terkait operasional pabrik, misalnya:\n- *\"Bagaimana status dan output produksi Line 1 hari ini?\"*\n- *\"Lacak progres pesanan SPK Snøhund Arctic Expedition Parka.\"*\n- *\"Cek sisa stok kain Waterproof 3-Layer Taslan di rak gudang.\"*\n- *\"Beri rekomendasi penanganan defect jahitan loncat.\"*"

// ValidateLuklaDomain checks whether the user prompt relates to Lukla manufacturing operations.
func ValidateLuklaDomain(prompt string) (bool, string) {
	clean := strings.ToLower(strings.TrimSpace(prompt))
	if clean == "" {
		return false, "Silakan masukkan pertanyaan seputar operasional pabrik Lukla."
	}

	// Always allow very short conversational greetings
	if clean == "halo" || clean == "hai" || clean == "hi" || clean == "hello" || clean == "selamat pagi" || clean == "selamat siang" || clean == "siapa kamu?" {
		return true, ""
	}

	// Check if any domain keyword is present
	for _, kw := range AllowedDomainKeywords {
		if strings.Contains(clean, kw) {
			return true, ""
		}
	}

	// Reject out of scope query
	return false, StrictRefusalMessage
}
