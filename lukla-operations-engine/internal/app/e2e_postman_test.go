package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestApp_GenerateTahap72PostmanCollection(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Setup Auth Tokens
	_, _ = application.db.Exec(`UPDATE inventory_lots SET current_quantity = 150.50, status = 'AVAILABLE' WHERE lot_number = 'LOT-SNO-20261014-01'`)
	_, _ = application.db.Exec(`UPDATE inventory_lots SET current_quantity = 100.00, status = 'AVAILABLE' WHERE lot_number = 'LOT-AVE-20261014-01'`)
	_, _ = application.db.Exec(`DELETE FROM production_outputs WHERE operator_id IN (SELECT id FROM operators WHERE operator_code LIKE 'OP-E2E-%' OR operator_code LIKE 'OP-LOCK-%')`)
	_, _ = application.db.Exec(`DELETE FROM operators WHERE operator_code LIKE 'OP-E2E-%'`)

	execToken := getAuthToken(t, application, "executive@lukla.local", "password")
	clientToken := getAuthToken(t, application, "client@lukla.local", "password")
	prodToken := getAuthToken(t, application, "production@lukla.local", "password")
	whToken := getAuthToken(t, application, "warehouse@lukla.local", "password")
	finToken := getAuthToken(t, application, "finance@lukla.local", "password")

	var snoClientID, aveClientID, parkaArticleID, sampleComponentID string
	_ = application.db.QueryRow("SELECT client_id FROM users WHERE email = 'client@lukla.local'").Scan(&snoClientID)
	_ = application.db.QueryRow("SELECT id FROM clients WHERE id != $1 LIMIT 1", snoClientID).Scan(&aveClientID)
	_ = application.db.QueryRow("SELECT id FROM articles WHERE code = 'ART-SNO-PK01' LIMIT 1").Scan(&parkaArticleID)
	_ = application.db.QueryRow("SELECT id FROM components WHERE article_id = $1 LIMIT 1", parkaArticleID).Scan(&sampleComponentID)

	if snoClientID == "" || aveClientID == "" || parkaArticleID == "" {
		t.Fatalf("prerequisites missing: snoClientID=%s, aveClientID=%s, parkaArticleID=%s", snoClientID, aveClientID, parkaArticleID)
	}

	var snoMatID string
	_ = application.db.QueryRow("SELECT id FROM materials WHERE code = 'MAT-CTN-30S-BLK' LIMIT 1").Scan(&snoMatID)

	var lineID string
	_ = application.db.QueryRow("SELECT id FROM lines WHERE status = 'ACTIVE' LIMIT 1").Scan(&lineID)

	// =========================================================================
	// EKSEKUSI PIPELINE DENGAN PEREKAMAN RESPON AKTUAL RIIL
	// =========================================================================

	// 1. Health Pipeline
	reqHealth, _ := http.NewRequest(http.MethodGet, "/api/v1/health/pipeline", nil)
	wHealth := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wHealth, reqHealth)
	if wHealth.Code != http.StatusOK {
		t.Fatalf("expected 200 for health pipeline, got %d", wHealth.Code)
	}
	respHealthBody := wHealth.Body.String()

	// 2. Buat Sample Request v1
	ts := time.Now().UnixNano() % 100000
	uniqueSampleCode := fmt.Sprintf("SMP-E2E-PM-%d", ts)
	sampleReqBody := map[string]interface{}{
		"sampleCode":  uniqueSampleCode,
		"clientId":    snoClientID,
		"articleId":   parkaArticleID,
		"sampleType":  "PROTO",
		"targetDate":  "2026-11-01",
		"description": "Prototipe Jaket Parka Uji End-to-End Tahap 7.2 Postman",
	}
	sBodyBytes, _ := json.Marshal(sampleReqBody)
	reqSample, _ := http.NewRequest(http.MethodPost, "/api/v1/samples", bytes.NewBuffer(sBodyBytes))
	reqSample.Header.Set("Authorization", "Bearer "+execToken)
	reqSample.Header.Set("Content-Type", "application/json")
	wSample := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSample, reqSample)
	if wSample.Code != http.StatusCreated {
		t.Fatalf("expected 201 for sample request, got %d. Body: %s", wSample.Code, wSample.Body.String())
	}
	respSampleBody := wSample.Body.String()

	var sResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSample.Body.Bytes(), &sResp)
	sampleID := sResp.Data.ID

	// 3. Time Study v1
	ts1Body := map[string]interface{}{
		"componentId":         sampleComponentID,
		"observedSeconds":     270.0,
		"allowancePercentage": 15.0,
		"notes":               "Observasi time study awal prototipe v1",
	}
	ts1Bytes, _ := json.Marshal(ts1Body)
	reqTS1, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/samples/%s/versions/1/time-studies", sampleID), bytes.NewBuffer(ts1Bytes))
	reqTS1.Header.Set("Authorization", "Bearer "+execToken)
	reqTS1.Header.Set("Content-Type", "application/json")
	wTS1 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wTS1, reqTS1)
	if wTS1.Code != http.StatusCreated {
		t.Fatalf("expected 201 for time study v1, got %d. Body: %s", wTS1.Code, wTS1.Body.String())
	}
	respTS1Body := wTS1.Body.String()

	// 4. Ajukan v1 ke Klien
	reqSub1, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/samples/%s/submit-review", sampleID), nil)
	reqSub1.Header.Set("Authorization", "Bearer "+execToken)
	wSub1 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSub1, reqSub1)
	if wSub1.Code != http.StatusOK {
		t.Fatalf("expected 200 for submit review v1, got %d", wSub1.Code)
	}
	respSub1Body := wSub1.Body.String()

	// 5. Klien Request Revision via Portal
	revBody := map[string]interface{}{
		"revisionNotes": "Pola saku dada dinaikkan 3 cm dan zipper diganti matte black waterproof",
	}
	revBytes, _ := json.Marshal(revBody)
	reqRev, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/portal/samples/%s/request-revision", sampleID), bytes.NewBuffer(revBytes))
	reqRev.Header.Set("Authorization", "Bearer "+clientToken)
	reqRev.Header.Set("Content-Type", "application/json")
	wRev := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wRev, reqRev)
	if wRev.Code != http.StatusOK {
		t.Fatalf("expected 200 for client request revision, got %d", wRev.Code)
	}
	respRevBody := wRev.Body.String()

	// 6. Pabrik Terbitkan Versi 2 (v2)
	v2Body := map[string]interface{}{
		"revisionNotes": "Perbaikan pola saku dada naik 3 cm dan zipper matte black",
	}
	v2Bytes, _ := json.Marshal(v2Body)
	reqV2, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/samples/%s/versions", sampleID), bytes.NewBuffer(v2Bytes))
	reqV2.Header.Set("Authorization", "Bearer "+execToken)
	reqV2.Header.Set("Content-Type", "application/json")
	wV2 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wV2, reqV2)
	if wV2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create version 2, got %d", wV2.Code)
	}
	respV2Body := wV2.Body.String()

	// 7. Rekam Time Study v2
	ts2Body := map[string]interface{}{
		"componentId":         sampleComponentID,
		"observedSeconds":     288.0,
		"allowancePercentage": 15.0,
		"notes":               "Time study akurat versi 2 final",
	}
	ts2Bytes, _ := json.Marshal(ts2Body)
	reqTS2, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/samples/%s/versions/2/time-studies", sampleID), bytes.NewBuffer(ts2Bytes))
	reqTS2.Header.Set("Authorization", "Bearer "+execToken)
	reqTS2.Header.Set("Content-Type", "application/json")
	wTS2 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wTS2, reqTS2)
	if wTS2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for time study v2, got %d", wTS2.Code)
	}
	respTS2Body := wTS2.Body.String()

	// 8. Tetapkan Commercial Rate Card (Rp 45.000 / pcs)
	rateBody := map[string]interface{}{
		"proposedUnitPrice": 45000.00,
		"currency":          "IDR",
		"priceNotes":        "Tarif jasa CMT Jaket Parka v2 approved mencakup cutting, sewing, finishing & packing",
	}
	rateBytes, _ := json.Marshal(rateBody)
	reqRate, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/samples/%s/versions/2/commercial-rate", sampleID), bytes.NewBuffer(rateBytes))
	reqRate.Header.Set("Authorization", "Bearer "+execToken)
	reqRate.Header.Set("Content-Type", "application/json")
	wRate := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wRate, reqRate)
	if wRate.Code != http.StatusOK {
		t.Fatalf("expected 200 for commercial rate, got %d", wRate.Code)
	}
	respRateBody := wRate.Body.String()

	// Submit review v2
	reqSub2, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/samples/%s/submit-review", sampleID), nil)
	reqSub2.Header.Set("Authorization", "Bearer "+execToken)
	wSub2 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSub2, reqSub2)

	// 9. Digital Approval Klien via Portal
	apprBody := map[string]interface{}{
		"feedbackNotes": "Spesifikasi sample v2 dan penawaran harga Rp 45.000 disetujui",
	}
	apprBytes, _ := json.Marshal(apprBody)
	reqAppr, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/portal/samples/%s/approve", sampleID), bytes.NewBuffer(apprBytes))
	reqAppr.Header.Set("Authorization", "Bearer "+clientToken)
	reqAppr.Header.Set("Content-Type", "application/json")
	wAppr := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAppr, reqAppr)
	if wAppr.Code != http.StatusOK {
		t.Fatalf("expected 200 for client approval, got %d", wAppr.Code)
	}
	respApprBody := wAppr.Body.String()

	// 10. Konversi Sample v2 ke Order Massal 4.000 Pcs
	massOrderNumber := fmt.Sprintf("PO-SNO-E2E-PM-%d", ts)
	convBody := map[string]interface{}{
		"orderNumber":    massOrderNumber,
		"targetQuantity": 4000,
		"targetDate":     "2026-12-20",
		"notes":          "Order Produksi Massal 4000 pcs dari Sample v2 Approved",
	}
	convBytes, _ := json.Marshal(convBody)
	reqConv, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/samples/%s/convert-to-order", sampleID), bytes.NewBuffer(convBytes))
	reqConv.Header.Set("Authorization", "Bearer "+execToken)
	reqConv.Header.Set("Content-Type", "application/json")
	wConv := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wConv, reqConv)
	if wConv.Code != http.StatusCreated {
		t.Fatalf("expected 201 for convert to order, got %d. Body: %s", wConv.Code, wConv.Body.String())
	}
	respConvBody := wConv.Body.String()

	var convResp struct {
		Data struct {
			OrderID string `json:"orderId"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wConv.Body.Bytes(), &convResp)
	e2eOrderID := convResp.Data.OrderID

	var receiptID, dummyOrderID string
	_ = application.db.QueryRow("SELECT receipt_id, order_id FROM inventory_lots WHERE receipt_id IS NOT NULL LIMIT 1").Scan(&receiptID, &dummyOrderID)

	var aveLotID, snoLotID string
	err = application.db.QueryRow(`
		INSERT INTO inventory_lots (receipt_id, order_id, lot_number, material_id, client_id, initial_quantity, current_quantity, unit, roll_count, status, warehouse_location, notes)
		VALUES ($1, $2, $3, $4, $5, 500.0, 500.0, 'KG', 1, 'AVAILABLE', 'RAK-A-01', 'Test Lot E2E Postman')
		RETURNING id`, receiptID, dummyOrderID, fmt.Sprintf("LOT-E2E-PM-AVE-%d", ts), snoMatID, aveClientID).Scan(&aveLotID)
	if err != nil {
		t.Fatalf("failed to insert test ave lot: %v", err)
	}

	err = application.db.QueryRow(`
		INSERT INTO inventory_lots (receipt_id, order_id, lot_number, material_id, client_id, initial_quantity, current_quantity, unit, roll_count, status, warehouse_location, notes)
		VALUES ($1, $2, $3, $4, $5, 500.0, 500.0, 'KG', 1, 'AVAILABLE', 'RAK-A-01', 'Test Lot E2E Postman')
		RETURNING id`, receiptID, e2eOrderID, fmt.Sprintf("LOT-E2E-PM-SNO-%d", ts), snoMatID, snoClientID).Scan(&snoLotID)
	if err != nil {
		t.Fatalf("failed to insert test sno lot: %v", err)
	}

	// 11. Skenario 1 (Negatif): Blokade Kontaminasi Silang Antar Klien (400 Bad Request)
	crossBody := map[string]interface{}{
		"targetOrderId":      e2eOrderID,
		"cuttingTableNumber": "MEJA-CUTTING-E2E",
		"issuedDate":         "2026-10-15",
		"notes":              "Percobaan kontaminasi silang",
		"items": []map[string]interface{}{
			{"lotId": aveLotID, "quantity": 10.0, "notes": "Salah tarik kain Aventura"},
		},
	}
	crossBytes, _ := json.Marshal(crossBody)
	reqCross, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/issues", bytes.NewBuffer(crossBytes))
	reqCross.Header.Set("Authorization", "Bearer "+whToken)
	reqCross.Header.Set("Content-Type", "application/json")
	wCross := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCross, reqCross)
	if wCross.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for cross contamination, got %d", wCross.Code)
	}
	respCrossBody := wCross.Body.String()

	// 12. Skenario 1 (Positif): Pengeluaran Legal Bahan Baku Snohund ke Cutting
	legalIssueBody := map[string]interface{}{
		"targetOrderId":      e2eOrderID,
		"cuttingTableNumber": "MEJA-CUTTING-E2E",
		"issuedDate":         "2026-10-15",
		"notes":              "Pengeluaran sah kain Snøhund",
		"items": []map[string]interface{}{
			{"lotId": snoLotID, "quantity": 20.0, "notes": "Pengeluaran roll sah"},
		},
	}
	legalBytes, _ := json.Marshal(legalIssueBody)
	reqLegal, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/issues", bytes.NewBuffer(legalBytes))
	reqLegal.Header.Set("Authorization", "Bearer "+whToken)
	reqLegal.Header.Set("Content-Type", "application/json")
	wLegal := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLegal, reqLegal)
	if wLegal.Code != http.StatusCreated {
		t.Fatalf("expected 201 for legal issue, got %d: %s", wLegal.Code, wLegal.Body.String())
	}
	respLegalBody := wLegal.Body.String()

	// 13. Skenario 1: Pencatatan Scrap Pemotongan Kain (Toleransi < 5%)
	validScrapBody := map[string]interface{}{
		"orderId":       e2eOrderID,
		"materialId":    snoMatID,
		"scrapCategory": "PERCA_KECIL",
		"quantity":      1.00,
		"unit":          "KG",
		"disposition":   "SIMPAN_GUDANG_SISA",
		"notes":         "Limbah potongan pinggir pola jaket parka",
	}
	vScrapBytes, _ := json.Marshal(validScrapBody)
	reqVScrap, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/scraps", bytes.NewBuffer(vScrapBytes))
	reqVScrap.Header.Set("Authorization", "Bearer "+prodToken)
	reqVScrap.Header.Set("Content-Type", "application/json")
	wVScrap := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wVScrap, reqVScrap)
	if wVScrap.Code != http.StatusCreated {
		t.Fatalf("expected 201 for scrap, got %d: %s", wVScrap.Code, wVScrap.Body.String())
	}
	respScrapBody := wVScrap.Body.String()

	// Setup operator & output produksi (3.000 pcs)
	midweekCode := fmt.Sprintf("OP-E2E-PM-%d", ts)
	var e2eOperatorID string
	_ = application.db.QueryRow(`
		INSERT INTO operators (operator_code, name, join_date, status, default_line)
		VALUES ($1, 'Siti Penjahit Midweek Postman', '2026-11-10', 'ACTIVE', 'LINE-01')
		RETURNING id
	`, midweekCode).Scan(&e2eOperatorID)

	t.Cleanup(func() {
		_, _ = application.db.Exec(`DELETE FROM production_outputs WHERE operator_id = $1`, e2eOperatorID)
		_, _ = application.db.Exec(`DELETE FROM operators WHERE id = $1`, e2eOperatorID)
		_, _ = application.db.Exec(`DELETE FROM inventory_lots WHERE id IN ($1, $2)`, aveLotID, snoLotID)
	})

	_, err = application.db.Exec(`
		INSERT INTO production_outputs (line_id, order_id, operator_id, component_id, quantity, time_slot, work_date, snapshot_sewing_rate)
		VALUES ($1, $2, $3, $4, 3000, '09:00-09:30', '2026-11-11', 4000.00)
	`, lineID, e2eOrderID, e2eOperatorID, sampleComponentID)
	if err != nil {
		t.Fatalf("failed to insert production outputs: %v", err)
	}

	// Assembly 3.000 pcs
	asmBody := map[string]interface{}{
		"orderId":           e2eOrderID,
		"articleId":         parkaArticleID,
		"quantityAssembled": 3000,
		"assembledDate":     "2026-11-11",
		"notes":             "Jahit massal 3000 pcs selesai",
	}
	asmBytes, _ := json.Marshal(asmBody)
	reqAsm, _ := http.NewRequest(http.MethodPost, "/api/v1/assemblies", bytes.NewBuffer(asmBytes))
	reqAsm.Header.Set("Authorization", "Bearer "+prodToken)
	reqAsm.Header.Set("Content-Type", "application/json")
	wAsm := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAsm, reqAsm)
	if wAsm.Code != http.StatusCreated {
		t.Fatalf("expected 201 for assembly, got %d", wAsm.Code)
	}
	var asmResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wAsm.Body.Bytes(), &asmResp)
	e2eAssemblyID := asmResp.Data.ID

	// 14. Buka Periode Payroll Mingguan
	payPeriodCode := fmt.Sprintf("PAY-E2E-PM-%d", ts)
	openPayBody := map[string]interface{}{
		"periodCode":              payPeriodCode,
		"startDate":               "2026-11-09",
		"endDate":                 "2026-11-15",
		"mealAllowancePercentage": 10.00,
		"notes":                   "Payroll mingguan e2e test",
	}
	pBytes, _ := json.Marshal(openPayBody)
	reqP, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods", bytes.NewBuffer(pBytes))
	reqP.Header.Set("Authorization", "Bearer "+finToken)
	reqP.Header.Set("Content-Type", "application/json")
	wP := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wP, reqP)
	if wP.Code != http.StatusCreated {
		t.Fatalf("expected 201 for open payroll, got %d", wP.Code)
	}
	respPayOpenBody := wP.Body.String()

	var pResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wP.Body.Bytes(), &pResp)
	payPeriodID := pResp.Data.ID

	// 15. Skenario 2: Hitung Payroll Midweek
	reqCalcPay, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/payroll/periods/%s/calculate", payPeriodID), bytes.NewBufferString(`{}`))
	reqCalcPay.Header.Set("Authorization", "Bearer "+finToken)
	reqCalcPay.Header.Set("Content-Type", "application/json")
	wCalcPay := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCalcPay, reqCalcPay)
	if wCalcPay.Code != http.StatusOK {
		t.Fatalf("expected 200 for payroll calculate, got %d: %s", wCalcPay.Code, wCalcPay.Body.String())
	}
	respCalcPayBody := wCalcPay.Body.String()

	// 16. Skenario 2: Kunci Periode Payroll
	lockPayBody := map[string]interface{}{"notes": "Payroll disetujui & dikunci"}
	lpBytes, _ := json.Marshal(lockPayBody)
	reqLockPay, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/payroll/periods/%s/lock", payPeriodID), bytes.NewBuffer(lpBytes))
	reqLockPay.Header.Set("Authorization", "Bearer "+execToken)
	reqLockPay.Header.Set("Content-Type", "application/json")
	wLockPay := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLockPay, reqLockPay)
	if wLockPay.Code != http.StatusOK {
		t.Fatalf("expected 200 for payroll lock, got %d", wLockPay.Code)
	}
	respLockPayBody := wLockPay.Body.String()

	// 17. Skenario 3: Input Inspeksi Kualitas 3.000 Pcs Lolos Mutu
	inspBody := map[string]interface{}{
		"assemblyId":     e2eAssemblyID,
		"orderId":        e2eOrderID,
		"inspectionDate": "2026-10-15",
		"passedQuantity": 3000,
		"repairQuantity": 0,
		"rejectQuantity": 0,
		"notes":          "Inspeksi massal lolos kualifikasi mutu",
		"defects":        []interface{}{},
	}
	inBytes, _ := json.Marshal(inspBody)
	reqInsp, _ := http.NewRequest(http.MethodPost, "/api/v1/qc/inspections", bytes.NewBuffer(inBytes))
	reqInsp.Header.Set("Authorization", "Bearer "+execToken)
	reqInsp.Header.Set("Content-Type", "application/json")
	wInsp := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wInsp, reqInsp)
	if wInsp.Code != http.StatusCreated {
		t.Fatalf("expected 201 for qc inspection, got %d. Body: %s", wInsp.Code, wInsp.Body.String())
	}
	respInspBody := wInsp.Body.String()

	// Buka periode bonus QC
	openQCBody := map[string]interface{}{
		"name":      fmt.Sprintf("Pool Bonus QC - Skenario Uji 3 Postman %d", ts),
		"startDate": "2026-10-01",
		"endDate":   "2026-10-31",
		"notes":     "Bonus QC periode Oktober 2026",
	}
	qcBytes, _ := json.Marshal(openQCBody)
	reqQC, _ := http.NewRequest(http.MethodPost, "/api/v1/qc-bonus/periods", bytes.NewBuffer(qcBytes))
	reqQC.Header.Set("Authorization", "Bearer "+execToken)
	reqQC.Header.Set("Content-Type", "application/json")
	wQC := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wQC, reqQC)
	if wQC.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create qc bonus period, got %d. Body: %s", wQC.Code, wQC.Body.String())
	}
	var qcResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wQC.Body.Bytes(), &qcResp)
	qcPeriodID := qcResp.Data.ID

	// 18. Skenario 3: Hitung Pool Bonus QC 3-Arah
	reqCalcQC, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/qc-bonus/periods/%s/calculate", qcPeriodID), nil)
	reqCalcQC.Header.Set("Authorization", "Bearer "+execToken)
	wCalcQC := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCalcQC, reqCalcQC)
	if wCalcQC.Code != http.StatusOK {
		t.Fatalf("expected 200 for qc bonus calculate, got %d", wCalcQC.Code)
	}
	respCalcQCBody := wCalcQC.Body.String()

	// 19. Skenario 3: Kunci Pool Bonus QC
	reqLockQC, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/qc-bonus/periods/%s/lock", qcPeriodID), nil)
	reqLockQC.Header.Set("Authorization", "Bearer "+execToken)
	wLockQC := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLockQC, reqLockQC)
	if wLockQC.Code != http.StatusOK {
		t.Fatalf("expected 200 for qc bonus lock, got %d", wLockQC.Code)
	}
	respLockQCBody := wLockQC.Body.String()

	// Overhead alokasi agar laporan COGS terisi overhead
	testOHMonth := 11
	testOHYear := 2026
	_, _ = application.db.Exec("DELETE FROM overhead_allocations WHERE overhead_period_id IN (SELECT id FROM overhead_periods WHERE period_month = $1 AND period_year = $2)", testOHMonth, testOHYear)
	_, _ = application.db.Exec("DELETE FROM overhead_expenses WHERE overhead_period_id IN (SELECT id FROM overhead_periods WHERE period_month = $1 AND period_year = $2)", testOHMonth, testOHYear)
	_, _ = application.db.Exec("DELETE FROM overhead_periods WHERE period_month = $1 AND period_year = $2", testOHMonth, testOHYear)

	ohOpenBody := map[string]interface{}{
		"periodMonth": testOHMonth,
		"periodYear":  testOHYear,
		"notes":       "Buku overhead November 2026 Postman",
	}
	ohBytes, _ := json.Marshal(ohOpenBody)
	reqOH, _ := http.NewRequest(http.MethodPost, "/api/v1/overhead/periods", bytes.NewBuffer(ohBytes))
	reqOH.Header.Set("Authorization", "Bearer "+execToken)
	reqOH.Header.Set("Content-Type", "application/json")
	wOH := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wOH, reqOH)
	var ohResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wOH.Body.Bytes(), &ohResp)
	ohPeriodID := ohResp.Data.ID

	var elecCatID string
	_ = application.db.QueryRow("SELECT id FROM overhead_categories WHERE code = 'ELECTRICITY' LIMIT 1").Scan(&elecCatID)
	ohExpBody := map[string]interface{}{
		"categoryId":       elecCatID,
		"expenseDate":      "2026-11-10",
		"amount":           16000000.00,
		"invoiceReference": fmt.Sprintf("PLN-PM-%d", ts),
		"notes":            "Beban listrik pabrik November",
	}
	ohExpBytes, _ := json.Marshal(ohExpBody)
	reqOHExp, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/overhead/periods/%s/expenses", ohPeriodID), bytes.NewBuffer(ohExpBytes))
	reqOHExp.Header.Set("Authorization", "Bearer "+execToken)
	reqOHExp.Header.Set("Content-Type", "application/json")
	wOHExp := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wOHExp, reqOHExp)

	ohCalcBody := map[string]interface{}{
		"weightGarment": 1.0,
		"weightBag":     1.25,
		"notes":         "Kalkulasi overhead November",
	}
	ohCBytes, _ := json.Marshal(ohCalcBody)
	reqOHCalc, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/overhead/periods/%s/calculate", ohPeriodID), bytes.NewBuffer(ohCBytes))
	reqOHCalc.Header.Set("Authorization", "Bearer "+execToken)
	reqOHCalc.Header.Set("Content-Type", "application/json")
	wOHCalc := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wOHCalc, reqOHCalc)

	reqOHLock, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/overhead/periods/%s/lock", ohPeriodID), nil)
	reqOHLock.Header.Set("Authorization", "Bearer "+execToken)
	wOHLock := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wOHLock, reqOHLock)

	// 20. Selesaikan Order Massal & Idempotency Guard (BPMN 01)
	compOrderBody := map[string]interface{}{
		"notes": "Pesanan massal 3000 pcs selesai diproduksi & siap kirim ke warehouse klien",
	}
	coBytes, _ := json.Marshal(compOrderBody)
	reqCompOrder, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/orders/%s/complete", e2eOrderID), bytes.NewBuffer(coBytes))
	reqCompOrder.Header.Set("Authorization", "Bearer "+execToken)
	reqCompOrder.Header.Set("Content-Type", "application/json")
	wCompOrder := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCompOrder, reqCompOrder)
	if wCompOrder.Code != http.StatusOK {
		t.Fatalf("expected 200 for complete order, got %d. Body: %s", wCompOrder.Code, wCompOrder.Body.String())
	}
	respCompOrderBody := wCompOrder.Body.String()

	// 21. Laporan HPP / COGS Komprehensif
	reqCOGS, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/reports/cogs/%s", e2eOrderID), nil)
	reqCOGS.Header.Set("Authorization", "Bearer "+execToken)
	wCOGS := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCOGS, reqCOGS)
	if wCOGS.Code != http.StatusOK {
		t.Fatalf("expected 200 for COGS report, got %d. Body: %s", wCOGS.Code, wCOGS.Body.String())
	}
	respCOGSBody := wCOGS.Body.String()

	// =========================================================================
	// BANGUN POSTMAN FOLDER 17
	// =========================================================================
	folder17 := PostmanFolder{
		Name:     "17. Tahap 7.2: Comprehensive End-to-End Testing & 4 SRS Scenarios Acceptance (BPMN 01)",
		Response: nil,
		Item: []PostmanItem{
			{
				Name: "17.1 GET Audit Kesiapan Manufaktur Pipeline End-to-End (BPMN 01)",
				Request: map[string]interface{}{
					"description": "Audit status kesiapan 12 modul operasional engine serta 4 skenario uji SRS.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
					},
					"method": "GET",
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"health", "pipeline"},
						"raw":  "{{base_url}}/health/pipeline",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 200 OK\", function () {",
								"    pm.response.to.have.status(200);",
								"});",
								"var jsonData = pm.response.json();",
								"pm.test(\"Pipeline is operational and all modules verified\", function () {",
								"    pm.expect(jsonData.success).to.eql(true);",
								"    pm.expect(jsonData.data.status).to.eql(\"OPERATIONAL\");",
								"    pm.expect(jsonData.data.modules.length).to.eql(12);",
								"    pm.expect(jsonData.data.scenariosVerified.length).to.eql(4);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respHealthBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "17.2 POST Skenario 4: Buat Sample Request v1 (BPMN 03)",
				Request: map[string]interface{}{
					"description": "Membuat permintaan sample pakaian prototipe baru.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"sampleCode\": \"{{e2e_sample_code}}\",\n    \"clientId\": \"{{e2e_client_id}}\",\n    \"articleId\": \"{{e2e_article_id}}\",\n    \"sampleType\": \"PROTO\",\n    \"targetDate\": \"2026-11-01\",\n    \"description\": \"Prototipe Jaket Parka Uji End-to-End Tahap 7.2\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"samples"},
						"raw":  "{{base_url}}/samples",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 201 Created\", function () {",
								"    pm.response.to.have.status(201);",
								"});",
								"var jsonData = pm.response.json();",
								"if (jsonData.data && jsonData.data.id) {",
								"    pm.collectionVariables.set('e2e_sample_id', jsonData.data.id);",
								"}",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respSampleBody,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "17.3 POST Skenario 4: Observasi Time Study Sample v1 (BPMN 03)",
				Request: map[string]interface{}{
					"description": "Merekam observasi waktu kerja awal untuk sample versi 1.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"componentId\": \"{{e2e_component_id}}\",\n    \"observedSeconds\": 270.0,\n    \"allowancePercentage\": 15.0,\n    \"notes\": \"Observasi time study awal prototipe v1\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"samples", "{{e2e_sample_id}}", "versions", "1", "time-studies"},
						"raw":  "{{base_url}}/samples/{{e2e_sample_id}}/versions/1/time-studies",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 201 Created\", function () {",
								"    pm.response.to.have.status(201);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respTS1Body,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "17.4 POST Skenario 4: Ajukan Sample v1 ke Klien (BPMN 03)",
				Request: map[string]interface{}{
					"description": "Mengajukan sample versi 1 ke klien untuk proses fitting dan review.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
					},
					"method": "POST",
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"samples", "{{e2e_sample_id}}", "submit-review"},
						"raw":  "{{base_url}}/samples/{{e2e_sample_id}}/submit-review",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 200 OK\", function () {",
								"    pm.response.to.have.status(200);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respSub1Body,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "17.5 POST Skenario 4: Klien Request Revision via Portal (BPMN 11)",
				Request: map[string]interface{}{
					"description": "Klien mengajukan permintaan revisi pola saku dan zipper via portal klien.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{client_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"revisionNotes\": \"Pola saku dada dinaikkan 3 cm dan zipper diganti matte black waterproof\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"portal", "samples", "{{e2e_sample_id}}", "request-revision"},
						"raw":  "{{base_url}}/portal/samples/{{e2e_sample_id}}/request-revision",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 200 OK\", function () {",
								"    pm.response.to.have.status(200);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respRevBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "17.6 POST Skenario 4: Pabrik Terbitkan Versi 2 (v2)",
				Request: map[string]interface{}{
					"description": "Pabrik mempublikasikan sample versi 2 mengakomodasi catatan revisi klien.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"revisionNotes\": \"Perbaikan pola saku dada naik 3 cm dan zipper matte black\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"samples", "{{e2e_sample_id}}", "versions"},
						"raw":  "{{base_url}}/samples/{{e2e_sample_id}}/versions",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 201 Created\", function () {",
								"    pm.response.to.have.status(201);",
								"});",
								"var jsonData = pm.response.json();",
								"pm.test(\"Version 2 issued\", function () {",
								"    pm.expect(jsonData.data.versionNumber).to.eql(2);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respV2Body,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "17.7 POST Skenario 4: Rekam Time Study Versi 2 (BPMN 03)",
				Request: map[string]interface{}{
					"description": "Perekaman time study final akurat untuk sample versi 2.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"componentId\": \"{{e2e_component_id}}\",\n    \"observedSeconds\": 288.0,\n    \"allowancePercentage\": 15.0,\n    \"notes\": \"Time study akurat versi 2 final\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"samples", "{{e2e_sample_id}}", "versions", "2", "time-studies"},
						"raw":  "{{base_url}}/samples/{{e2e_sample_id}}/versions/2/time-studies",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 201 Created\", function () {",
								"    pm.response.to.have.status(201);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respTS2Body,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "17.8 PUT Skenario 4: Tetapkan Commercial Rate Card (Rp 45.000 / pcs)",
				Request: map[string]interface{}{
					"description": "Menetapkan penawaran harga komersial resmi per unit pakaian untuk sample versi 2.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "PUT",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"proposedUnitPrice\": 45000.00,\n    \"currency\": \"IDR\",\n    \"priceNotes\": \"Tarif jasa CMT Jaket Parka v2 approved mencakup cutting, sewing, finishing & packing\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"samples", "{{e2e_sample_id}}", "versions", "2", "commercial-rate"},
						"raw":  "{{base_url}}/samples/{{e2e_sample_id}}/versions/2/commercial-rate",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 200 OK\", function () {",
								"    pm.response.to.have.status(200);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respRateBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "17.9 POST Skenario 4: Digital Approval Klien via Portal (BPMN 11)",
				Request: map[string]interface{}{
					"description": "Klien menyetujui secara digital sample versi 2 beserta harga penawaran komersial.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{client_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"feedbackNotes\": \"Spesifikasi sample v2 dan penawaran harga Rp 45.000 disetujui\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"portal", "samples", "{{e2e_sample_id}}", "approve"},
						"raw":  "{{base_url}}/portal/samples/{{e2e_sample_id}}/approve",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 200 OK\", function () {",
								"    pm.response.to.have.status(200);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respApprBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "17.10 POST Skenario 4: Konversi Sample v2 Menjadi Order Massal 4.000 Pcs",
				Request: map[string]interface{}{
					"description": "Konversi sample v2 menjadi order massal 4.000 pcs (Total Nilai Kontrak Rp 180.000.000).",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"orderNumber\": \"{{e2e_order_number}}\",\n    \"targetQuantity\": 4000,\n    \"targetDate\": \"2026-12-20\",\n    \"notes\": \"Order Produksi Massal 4000 pcs dari Sample v2 Approved\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"samples", "{{e2e_sample_id}}", "convert-to-order"},
						"raw":  "{{base_url}}/samples/{{e2e_sample_id}}/convert-to-order",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 201 Created\", function () {",
								"    pm.response.to.have.status(201);",
								"});",
								"var jsonData = pm.response.json();",
								"if (jsonData.data && jsonData.data.orderId) {",
								"    pm.collectionVariables.set('e2e_order_id', jsonData.data.orderId);",
								"}",
								"pm.test(\"Bound to version 2 with correct contract price\", function () {",
								"    pm.expect(jsonData.data.approvedVersion).to.eql(2);",
								"    pm.expect(jsonData.data.unitPrice).to.eql(45000);",
								"    pm.expect(jsonData.data.totalAmount).to.eql(180000000);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respConvBody,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "17.11 POST Skenario 1 (Negatif): Blokade Kontaminasi Silang Antar Klien (400 Bad Request)",
				Request: map[string]interface{}{
					"description": "Pengujian blokade kontaminasi silang gudang: Kain lot milik Aventura ditolak saat hendak dikeluarkan ke Order Snøhund.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{warehouse_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"targetOrderId\": \"{{e2e_order_id}}\",\n    \"cuttingTableNumber\": \"MEJA-CUTTING-E2E\",\n    \"issuedDate\": \"2026-10-15\",\n    \"notes\": \"Percobaan kontaminasi silang\",\n    \"items\": [\n        {\n            \"lotId\": \"{{e2e_aventura_lot_id}}\",\n            \"quantity\": 10.0,\n            \"notes\": \"Salah tarik kain Aventura\"\n        }\n    ]\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"inventory", "issues"},
						"raw":  "{{base_url}}/inventory/issues",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 400 Bad Request (Blocked)\", function () {",
								"    pm.response.to.have.status(400);",
								"});",
								"var jsonData = pm.response.json();",
								"pm.test(\"Cross client contamination strictly prevented\", function () {",
								"    pm.expect(jsonData.success).to.eql(false);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respCrossBody,
						"code":                     400,
						"name":                     "Bad Request (Blocked Cross-Contamination)",
						"status":                   "Bad Request",
					},
				},
			},
			{
				Name: "17.12 POST Skenario 1 (Positif): Pengeluaran Legal Bahan Baku Snøhund ke Cutting",
				Request: map[string]interface{}{
					"description": "Pengeluaran sah bahan baku kain lot Snøhund untuk pemotongan order jahit.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{warehouse_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"targetOrderId\": \"{{e2e_order_id}}\",\n    \"cuttingTableNumber\": \"MEJA-CUTTING-E2E\",\n    \"issuedDate\": \"2026-10-15\",\n    \"notes\": \"Pengeluaran sah kain Snøhund\",\n    \"items\": [\n        {\n            \"lotId\": \"{{e2e_snohund_lot_id}}\",\n            \"quantity\": 20.0,\n            \"notes\": \"Pengeluaran roll sah\"\n        }\n    ]\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"inventory", "issues"},
						"raw":  "{{base_url}}/inventory/issues",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 201 Created\", function () {",
								"    pm.response.to.have.status(201);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respLegalBody,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "17.13 POST Skenario 1: Pencatatan Scrap Pemotongan Kain (Toleransi < 5%)",
				Request: map[string]interface{}{
					"description": "Pencatatan sisa potongan pinggir pola jaket (2% dari bahan, dalam batas toleransi scrap 5%).",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{production_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"orderId\": \"{{e2e_order_id}}\",\n    \"materialId\": \"{{e2e_material_id}}\",\n    \"scrapCategory\": \"PERCA_KECIL\",\n    \"quantity\": 1.00,\n    \"unit\": \"KG\",\n    \"disposition\": \"SIMPAN_GUDANG_SISA\",\n    \"notes\": \"Limbah potongan pinggir pola jaket parka\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"inventory", "scraps"},
						"raw":  "{{base_url}}/inventory/scraps",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 201 Created\", function () {",
								"    pm.response.to.have.status(201);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respScrapBody,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "17.14 POST Skenario 2: Buka Periode Payroll Mingguan (BPMN 07)",
				Request: map[string]interface{}{
					"description": "Pembukaan periode payroll mingguan oleh Finance untuk penjahit dan staff.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{finance_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"periodCode\": \"{{e2e_pay_period_code}}\",\n    \"startDate\": \"2026-10-19\",\n    \"endDate\": \"2026-10-25\",\n    \"mealAllowancePercentage\": 10.00,\n    \"notes\": \"Payroll mingguan e2e test\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"payroll", "periods"},
						"raw":  "{{base_url}}/payroll/periods",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 201 Created\", function () {",
								"    pm.response.to.have.status(201);",
								"});",
								"var jsonData = pm.response.json();",
								"if (jsonData.data && jsonData.data.id) {",
								"    pm.collectionVariables.set('e2e_pay_period_id', jsonData.data.id);",
								"}",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respPayOpenBody,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "17.15 POST Skenario 2: Kalkulasi Payroll Midweek & Prorata Uang Makan",
				Request: map[string]interface{}{
					"description": "Menghitung upah borongan operator jahit midweek berdasarkan output aktual per 30 menit.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{finance_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"payroll", "periods", "{{e2e_pay_period_id}}", "calculate"},
						"raw":  "{{base_url}}/payroll/periods/{{e2e_pay_period_id}}/calculate",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 200 OK\", function () {",
								"    pm.response.to.have.status(200);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respCalcPayBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "17.16 POST Skenario 2: Kunci Periode Payroll & Immutability Guard",
				Request: map[string]interface{}{
					"description": "Mengunci periode payroll menjadi LOCKED sehingga data upah tidak dapat diubah lagi.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"notes\": \"Payroll disetujui & dikunci\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"payroll", "periods", "{{e2e_pay_period_id}}", "lock"},
						"raw":  "{{base_url}}/payroll/periods/{{e2e_pay_period_id}}/lock",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 200 OK\", function () {",
								"    pm.response.to.have.status(200);",
								"});",
								"var jsonData = pm.response.json();",
								"pm.test(\"Payroll period is LOCKED\", function () {",
								"    pm.expect(jsonData.data.status).to.eql(\"LOCKED\");",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respLockPayBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "17.17 POST Skenario 3: Input Hasil Inspeksi Kualitas 3.000 Pcs Lolos QC",
				Request: map[string]interface{}{
					"description": "Pencatatan hasil inspeksi mutu: 3.000 pcs rakitan pakaian lolos kualifikasi QC tanpa cacat.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"assemblyId\": \"{{e2e_assembly_id}}\",\n    \"orderId\": \"{{e2e_order_id}}\",\n    \"inspectionDate\": \"2026-10-15\",\n    \"passedQuantity\": 3000,\n    \"repairQuantity\": 0,\n    \"rejectQuantity\": 0,\n    \"notes\": \"Inspeksi massal lolos kualifikasi mutu\",\n    \"defects\": []\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"qc", "inspections"},
						"raw":  "{{base_url}}/qc/inspections",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 201 Created\", function () {",
								"    pm.response.to.have.status(201);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respInspBody,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "17.18 POST Skenario 3: Kalkulasi Pool Bonus Mutu 3-Arah (Rp 1.470.000)",
				Request: map[string]interface{}{
					"description": "Menghitung pool bonus QC untuk 3.000 pcs produk lolos mutu (3.000 x Rp 500 = Rp 1.500.000 minus defect).",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
					},
					"method": "POST",
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"qc-bonus", "periods", "{{e2e_qc_bonus_period_id}}", "calculate"},
						"raw":  "{{base_url}}/qc-bonus/periods/{{e2e_qc_bonus_period_id}}/calculate",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 200 OK\", function () {",
								"    pm.response.to.have.status(200);",
								"});",
								"var jsonData = pm.response.json();",
								"pm.test(\"Total pool bonus calculated\", function () {",
								"    pm.expect(jsonData.data.totalPoolAmount).to.be.above(0);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respCalcQCBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "17.19 POST Skenario 3: Kunci Periode Pool Bonus QC",
				Request: map[string]interface{}{
					"description": "Mengunci periode bonus QC secara permanen.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
					},
					"method": "POST",
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"qc-bonus", "periods", "{{e2e_qc_bonus_period_id}}", "lock"},
						"raw":  "{{base_url}}/qc-bonus/periods/{{e2e_qc_bonus_period_id}}/lock",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 200 OK\", function () {",
								"    pm.response.to.have.status(200);",
								"});",
								"var jsonData = pm.response.json();",
								"pm.test(\"QC bonus period is LOCKED\", function () {",
								"    pm.expect(jsonData.data.status).to.eql(\"LOCKED\");",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respLockQCBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "17.20 POST Selesaikan Order Massal & Idempotency Guard (BPMN 01)",
				Request: map[string]interface{}{
					"description": "Penyelesaian pesanan massal 3.000 pcs dan pengujian idempotensi guard pencegahan penyelesaian ganda.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"notes\": \"Pesanan massal 3000 pcs selesai diproduksi & siap kirim ke warehouse klien\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"orders", "{{e2e_order_id}}", "complete"},
						"raw":  "{{base_url}}/orders/{{e2e_order_id}}/complete",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 200 OK\", function () {",
								"    pm.response.to.have.status(200);",
								"});",
								"var jsonData = pm.response.json();",
								"pm.test(\"Order marked as completed\", function () {",
								"    pm.expect(jsonData.data.status).to.eql(\"COMPLETED\");",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respCompOrderBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "17.21 GET Laporan HPP / COGS Komprehensif Order Massal & Margin (BPMN 01)",
				Request: map[string]interface{}{
					"description": "Laporan ringkasan analisis HPP utuh per order mengintegrasikan Direct Labor, QC Bonus, dan Overhead Pabrik.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
					},
					"method": "GET",
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"reports", "cogs", "{{e2e_order_id}}"},
						"raw":  "{{base_url}}/reports/cogs/{{e2e_order_id}}",
					},
				},
				Event: []map[string]interface{}{
					{
						"listen": "test",
						"script": map[string]interface{}{
							"exec": []string{
								"pm.test(\"Status code is 200 OK\", function () {",
								"    pm.response.to.have.status(200);",
								"});",
								"var jsonData = pm.response.json();",
								"pm.test(\"Comprehensive COGS report returned\", function () {",
								"    pm.expect(jsonData.success).to.eql(true);",
								"    pm.expect(jsonData.data.contractPricePerUnit).to.eql(45000);",
								"    pm.expect(jsonData.data.costComponents.directLaborPerUnit).to.be.above(0);",
								"    pm.expect(jsonData.data.costComponents.qcBonusPerUnit).to.eql(500);",
								"    pm.expect(jsonData.data.costComponents.overheadPerUnit).to.be.above(0);",
								"    pm.expect(jsonData.data.grossMarginPerUnit).to.be.above(0);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respCOGSBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
		},
	}

	// Cari file Postman collection
	candidates := []string{
		"../../../docs/postman/Lukla_Articles_and_Components.postman_collection.json",
		"../../docs/postman/Lukla_Articles_and_Components.postman_collection.json",
		"../docs/postman/Lukla_Articles_and_Components.postman_collection.json",
		"../../../Lukla_Articles_and_Components.postman_collection.json",
		"../../Lukla_Articles_and_Components.postman_collection.json",
		"../Lukla_Articles_and_Components.postman_collection.json",
		"Lukla_Articles_and_Components.postman_collection.json",
	}
	var postmanPath string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			postmanPath = c
			break
		}
	}
	if postmanPath == "" {
		t.Fatalf("could not find Lukla_Articles_and_Components.postman_collection.json in candidates")
	}

	fileBytes, err := os.ReadFile(postmanPath)
	if err != nil {
		t.Fatalf("read postman file error: %v", err)
	}

	var rootCollection map[string]interface{}
	if err := json.Unmarshal(fileBytes, &rootCollection); err != nil {
		t.Fatalf("unmarshal postman error: %v", err)
	}

	items, ok := rootCollection["item"].([]interface{})
	if !ok {
		t.Fatalf("items in collection not found or not array")
	}

	// Filter out Folder 17 jika sudah ada
	newItems := make([]interface{}, 0, len(items)+1)
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if ok {
			name, _ := m["name"].(string)
			if len(name) >= 2 && name[:2] == "17" {
				continue
			}
		}
		newItems = append(newItems, it)
	}

	// Marshal folder 17 to interface{}
	f17Bytes, _ := json.Marshal(folder17)
	var f17Map map[string]interface{}
	_ = json.Unmarshal(f17Bytes, &f17Map)

	newItems = append(newItems, f17Map)
	rootCollection["item"] = newItems

	// Ensure variables are in place
	vars, _ := rootCollection["variable"].([]interface{})
	varMap := make(map[string]bool)
	for _, v := range vars {
		vm, _ := v.(map[string]interface{})
		k, _ := vm["key"].(string)
		varMap[k] = true
	}

	upsertVar := func(k, val string) {
		if !varMap[k] {
			vars = append(vars, map[string]interface{}{"key": k, "type": "string", "value": val})
			varMap[k] = true
		} else {
			for i, v := range vars {
				vm, _ := v.(map[string]interface{})
				if vm["key"] == k {
					vars[i] = map[string]interface{}{"key": k, "type": "string", "value": val}
				}
			}
		}
	}

	upsertVar("client_token", clientToken)
	upsertVar("warehouse_token", whToken)
	upsertVar("production_token", prodToken)
	upsertVar("finance_token", finToken)
	upsertVar("e2e_sample_code", uniqueSampleCode)
	upsertVar("e2e_sample_id", sampleID)
	upsertVar("e2e_article_id", parkaArticleID)
	upsertVar("e2e_component_id", sampleComponentID)
	upsertVar("e2e_client_id", snoClientID)
	upsertVar("e2e_order_number", massOrderNumber)
	upsertVar("e2e_order_id", e2eOrderID)
	upsertVar("e2e_material_id", snoMatID)
	upsertVar("e2e_aventura_lot_id", aveLotID)
	upsertVar("e2e_snohund_lot_id", snoLotID)
	upsertVar("e2e_assembly_id", e2eAssemblyID)
	upsertVar("e2e_pay_period_code", payPeriodCode)
	upsertVar("e2e_pay_period_id", payPeriodID)
	upsertVar("e2e_qc_bonus_period_id", qcPeriodID)

	rootCollection["variable"] = vars

	// Simpan kembali file Postman Collection
	outputBytes, err := json.MarshalIndent(rootCollection, "", "    ")
	if err != nil {
		t.Fatalf("marshal output error: %v", err)
	}

	if err := os.WriteFile(postmanPath, outputBytes, 0644); err != nil {
		t.Fatalf("write postman file error: %v", err)
	}

	t.Logf("Tahap 7.2 PASS: Postman Collection Folder 17 berhasil disinkronisasi dengan 21 endpoint dan respon aktual!")
}
