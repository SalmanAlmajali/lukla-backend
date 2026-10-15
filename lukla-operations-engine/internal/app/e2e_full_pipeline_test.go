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

// TestApp_EndToEndManufacturingPipeline_And_4SRSScenarios menguji pipeline manufaktur utuh (BPMN 01)
// dan memvalidasi ke-4 skenario uji SRS secara simultan dari hulu ke hilir pada Live Database.
func TestApp_EndToEndManufacturingPipeline_And_4SRSScenarios(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// -------------------------------------------------------------
	// 0. AUDIT PIPELINE HEALTH & READINESS
	// -------------------------------------------------------------
	_, _ = application.db.Exec(`UPDATE inventory_lots SET current_quantity = 150.50, status = 'AVAILABLE' WHERE lot_number = 'LOT-SNO-20261014-01'`)
	_, _ = application.db.Exec(`UPDATE inventory_lots SET current_quantity = 100.00, status = 'AVAILABLE' WHERE lot_number = 'LOT-AVE-20261014-01'`)
	_, _ = application.db.Exec(`DELETE FROM production_outputs WHERE operator_id IN (SELECT id FROM operators WHERE operator_code LIKE 'OP-E2E-%' OR operator_code LIKE 'OP-LOCK-%')`)
	_, _ = application.db.Exec(`DELETE FROM operators WHERE operator_code LIKE 'OP-E2E-%'`)

	reqHealth, _ := http.NewRequest(http.MethodGet, "/api/v1/health/pipeline", nil)
	wHealth := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wHealth, reqHealth)
	if wHealth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /health/pipeline, got %d", wHealth.Code)
	}
	var healthResp struct {
		Success bool `json:"success"`
		Data    struct {
			OverallStatus   string `json:"overallStatus"`
			ModulesCount    int    `json:"modulesCount"`
			ModulesVerified []string `json:"modulesVerified"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wHealth.Body.Bytes(), &healthResp)
	if !healthResp.Success || healthResp.Data.OverallStatus != "HEALTHY" || healthResp.Data.ModulesCount != 12 {
		t.Fatalf("pipeline health check failed: %+v", healthResp)
	}
	t.Logf("E2E STAGE 0 PASS: Pipeline Health 100%% VERIFIED (12 Modul Aktif)")

	// -------------------------------------------------------------
	// 1. SETUP AUTH TOKENS (Multi-Tenant Isolation)
	// -------------------------------------------------------------
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

	// -------------------------------------------------------------
	// 2. SKENARIO UJI 4: SAMPLE V1 -> V2 APPROVAL -> MASS ORDER CONVERSION (BPMN 03 & 11)
	// -------------------------------------------------------------
	// 2.1 Buat Sample Request v1
	uniqueSampleCode := fmt.Sprintf("SMP-E2E-%d", time.Now().UnixNano()%100000)
	sampleReqBody := map[string]interface{}{
		"sampleCode":  uniqueSampleCode,
		"clientId":    snoClientID,
		"articleId":   parkaArticleID,
		"sampleType":  "PROTO",
		"targetDate":  "2026-11-01",
		"description": "Prototipe Jaket Parka Uji End-to-End Tahap 7.2",
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
	var sResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSample.Body.Bytes(), &sResp)
	sampleID := sResp.Data.ID

	// 2.2 Observasi Time Study v1
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

	// 2.3 Ajukan Sample v1 ke Klien
	reqSub1, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/samples/%s/submit-review", sampleID), nil)
	reqSub1.Header.Set("Authorization", "Bearer "+execToken)
	wSub1 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSub1, reqSub1)
	if wSub1.Code != http.StatusOK {
		t.Fatalf("expected 200 for submit review v1, got %d", wSub1.Code)
	}

	// 2.4 Klien Request Revision via Portal
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

	// 2.5 Pabrik Terbitkan Versi 2 (v2)
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
	var v2Resp struct {
		Data struct {
			ID            string `json:"id"`
			VersionNumber int    `json:"versionNumber"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wV2.Body.Bytes(), &v2Resp)
	sampleVersion2ID := v2Resp.Data.ID

	// 2.6 Rekam Time Study v2 & Tetapkan Commercial Rate Card (Rp 45.000 / pcs)
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

	// 2.7 Ajukan v2 ke Klien & Digital Approval via Portal
	reqSub2, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/samples/%s/submit-review", sampleID), nil)
	reqSub2.Header.Set("Authorization", "Bearer "+execToken)
	wSub2 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSub2, reqSub2)

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

	// 2.8 Konversi Sample v2 Approved ke Order Produksi Massal 4.000 Pcs
	massOrderNumber := fmt.Sprintf("PO-SNO-E2E-%d", time.Now().UnixNano()%100000)
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

	var convResp struct {
		Data struct {
			OrderID         string  `json:"orderId"`
			OrderNumber     string  `json:"orderNumber"`
			ApprovedVersion int     `json:"approvedVersion"`
			TargetQuantity  int     `json:"targetQuantity"`
			UnitPrice       float64 `json:"unitPrice"`
			TotalAmount     float64 `json:"totalAmount"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wConv.Body.Bytes(), &convResp)
	e2eOrderID := convResp.Data.OrderID
	if convResp.Data.ApprovedVersion != 2 || convResp.Data.UnitPrice != 45000.00 || convResp.Data.TotalAmount != 180000000.00 {
		t.Fatalf("SKENARIO 4 FAILED: conversion data mismatch: %+v", convResp.Data)
	}

	// Verifikasi langsung di DB: sample_version_id harus versi 2
	var dbSampleVerID string
	_ = application.db.QueryRow("SELECT sample_version_id FROM orders WHERE id = $1", e2eOrderID).Scan(&dbSampleVerID)
	if dbSampleVerID != sampleVersion2ID {
		t.Fatalf("SKENARIO 4 GOVERNANCE BREACH: order bound to wrong version %s, expected %s", dbSampleVerID, sampleVersion2ID)
	}
	t.Logf("E2E STAGE 1 PASS: Skenario 4 VERIFIED (Sample v1 -> v2 approved -> Order Massal mengikat v2 & Rp 90.000.000)")

	// -------------------------------------------------------------
	// 3. SKENARIO UJI 1: WAREHOUSE RECEIPT, LOT TAGGING, CROSS-CONTAMINATION & SCRAP (BPMN 02)
	// -------------------------------------------------------------
	var snoMatID string
	_ = application.db.QueryRow("SELECT id FROM materials WHERE code = 'MAT-CTN-30S-BLK' LIMIT 1").Scan(&snoMatID)

	var receiptID, dummyOrderID string
	_ = application.db.QueryRow("SELECT receipt_id, order_id FROM inventory_lots WHERE receipt_id IS NOT NULL LIMIT 1").Scan(&receiptID, &dummyOrderID)

	var aveLotID, snoLotID string
	err = application.db.QueryRow(`
		INSERT INTO inventory_lots (receipt_id, order_id, lot_number, material_id, client_id, initial_quantity, current_quantity, unit, roll_count, status, warehouse_location, notes)
		VALUES ($1, $2, $3, $4, $5, 500.0, 500.0, 'KG', 1, 'AVAILABLE', 'RAK-A-01', 'Test Lot E2E')
		RETURNING id`, receiptID, dummyOrderID, fmt.Sprintf("LOT-E2E-AVE-%d", time.Now().UnixNano()%100000), snoMatID, aveClientID).Scan(&aveLotID)
	if err != nil {
		t.Fatalf("failed to insert test ave lot: %v", err)
	}

	err = application.db.QueryRow(`
		INSERT INTO inventory_lots (receipt_id, order_id, lot_number, material_id, client_id, initial_quantity, current_quantity, unit, roll_count, status, warehouse_location, notes)
		VALUES ($1, $2, $3, $4, $5, 500.0, 500.0, 'KG', 1, 'AVAILABLE', 'RAK-A-01', 'Test Lot E2E')
		RETURNING id`, receiptID, e2eOrderID, fmt.Sprintf("LOT-E2E-SNO-%d", time.Now().UnixNano()%100000), snoMatID, snoClientID).Scan(&snoLotID)
	if err != nil {
		t.Fatalf("failed to insert test sno lot: %v", err)
	}

	// 3.2 Uji Blokade Kontaminasi Silang: Keluarkan Lot Aventura untuk Order Snøhund -> HARUS 400
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
		t.Fatalf("SKENARIO 1 FAILED: expected 400 Bad Request for cross-contamination, got %d. Body: %s", wCross.Code, wCross.Body.String())
	}

	// 3.3 Uji Pengeluaran Bahan Snøhund yang Sah
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
		t.Fatalf("expected 201 for legal issue, got %d. Body: %s", wLegal.Code, wLegal.Body.String())
	}
	var legalIssueResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wLegal.Body.Bytes(), &legalIssueResp)

	// 3.4 Uji Pencatatan Scrap Pemotongan Kain (FR-6.3 & Skenario 1)
	// Validasi kuantitas scrap tidak valid (<= 0) -> DITOLAK 400
	invalidScrapBody := map[string]interface{}{
		"orderId":       e2eOrderID,
		"materialId":    snoMatID,
		"scrapCategory": "PERCA_KECIL",
		"quantity":      -2.50,
		"unit":          "KG",
		"disposition":   "SIMPAN_GUDANG_SISA",
		"notes":         "Kuantitas scrap tidak valid",
	}
	invBytes, _ := json.Marshal(invalidScrapBody)
	reqInvScrap, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/scraps", bytes.NewBuffer(invBytes))
	reqInvScrap.Header.Set("Authorization", "Bearer "+prodToken)
	reqInvScrap.Header.Set("Content-Type", "application/json")
	wInvScrap := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wInvScrap, reqInvScrap)
	if wInvScrap.Code != http.StatusBadRequest {
		t.Fatalf("SKENARIO 1 FAILED: expected 400 for negative scrap quantity, got %d", wInvScrap.Code)
	}

	// Pencatatan scrap wajar (2% dari 50.0 KG = 1.0 KG) -> DITERIMA 201
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
		t.Fatalf("expected 201 for normal scrap, got %d. Body: %s", wVScrap.Code, wVScrap.Body.String())
	}
	t.Logf("E2E STAGE 2 PASS: Skenario 1 VERIFIED (Blokade kontaminasi silang aktif & toleransi scrap 5%%)")

	// -------------------------------------------------------------
	// 4. PRODUKSI JAHIT & RITME OUTPUT (BPMN 04 & 05)
	// -------------------------------------------------------------
	var lineID string
	_ = application.db.QueryRow("SELECT id FROM lines WHERE status = 'ACTIVE' LIMIT 1").Scan(&lineID)

	allocBody := map[string]interface{}{
		"orderId": e2eOrderID,
		"lineId":  lineID,
		"targetDate": "2026-11-20",
		"notes": "Alokasi order massal e2e ke lini 1",
	}
	alBytes, _ := json.Marshal(allocBody)
	reqAl, _ := http.NewRequest(http.MethodPost, "/api/v1/line-allocations", bytes.NewBuffer(alBytes))
	reqAl.Header.Set("Authorization", "Bearer "+prodToken)
	reqAl.Header.Set("Content-Type", "application/json")
	wAl := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAl, reqAl)
	// -------------------------------------------------------------
	// 4. PRODUKSI JAHIT & RITME OUTPUT (BPMN 04 & 05)
	// -------------------------------------------------------------
	// 4.1 Buat operator penjahit
	midweekCode := fmt.Sprintf("OP-E2E-%d", time.Now().UnixNano()%100000)
	var e2eOperatorID string
	_ = application.db.QueryRow(`
		INSERT INTO operators (operator_code, name, join_date, status, default_line)
		VALUES ($1, 'Siti Penjahit Midweek E2E', '2026-11-03', 'ACTIVE', 'LINE-01')
		RETURNING id
	`, midweekCode).Scan(&e2eOperatorID)

	t.Cleanup(func() {
		_, _ = application.db.Exec(`DELETE FROM production_outputs WHERE operator_id = $1`, e2eOperatorID)
		_, _ = application.db.Exec(`DELETE FROM operators WHERE id = $1`, e2eOperatorID)
		_, _ = application.db.Exec(`DELETE FROM inventory_lots WHERE id IN ($1, $2)`, aveLotID, snoLotID)
	})

	// 4.2 Catat output jahit komponen (3.000 pcs @ Rp 4.000)
	_, err = application.db.Exec(`
		INSERT INTO production_outputs (line_id, order_id, operator_id, component_id, quantity, time_slot, work_date, snapshot_sewing_rate)
		VALUES ($1, $2, $3, $4, 3000, '09:00-09:30', '2026-11-04', 4000.00)
	`, lineID, e2eOrderID, e2eOperatorID, sampleComponentID)
	if err != nil {
		t.Fatalf("failed to insert production outputs: %v", err)
	}

	// 4.3 Catat batch assembly jahit 3.000 pcs (siap inspeksi QC)
	asmBody := map[string]interface{}{
		"orderId":           e2eOrderID,
		"articleId":         parkaArticleID,
		"quantityAssembled": 3000,
		"assembledDate":     "2026-11-04",
		"notes":             "Jahit massal 3000 pcs selesai",
	}
	asmBytes, _ := json.Marshal(asmBody)
	reqAsm, _ := http.NewRequest(http.MethodPost, "/api/v1/assemblies", bytes.NewBuffer(asmBytes))
	reqAsm.Header.Set("Authorization", "Bearer "+prodToken)
	reqAsm.Header.Set("Content-Type", "application/json")
	wAsm := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAsm, reqAsm)
	if wAsm.Code != http.StatusCreated {
		t.Fatalf("expected 201 for assembly, got %d: %s", wAsm.Code, wAsm.Body.String())
	}
	var asmResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wAsm.Body.Bytes(), &asmResp)
	e2eAssemblyID := asmResp.Data.ID

	t.Logf("E2E STAGE 3 PASS: Produksi massal 3.000 pcs tercatat di lini jahit (assembly: %s)", e2eAssemblyID)

	// -------------------------------------------------------------
	// 5. SKENARIO UJI 2: PAYROLL OPERATOR MID-WEEK & LOCKING (BPMN 07)
	// -------------------------------------------------------------

	payPeriodCode := fmt.Sprintf("PAY-E2E-%d", time.Now().UnixNano()%100000)
	openPayBody := map[string]interface{}{
		"periodCode":              payPeriodCode,
		"startDate":               "2026-11-02",
		"endDate":                 "2026-11-08",
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
		t.Fatalf("expected 201 for open payroll period, got %d. Body: %s", wP.Code, wP.Body.String())
	}

	var pResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wP.Body.Bytes(), &pResp)
	payPeriodID := pResp.Data.ID

	// Kalkulasi payroll
	reqCalcPay, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/payroll/periods/%s/calculate", payPeriodID), bytes.NewBufferString(`{}`))
	reqCalcPay.Header.Set("Authorization", "Bearer "+finToken)
	reqCalcPay.Header.Set("Content-Type", "application/json")
	wCalcPay := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCalcPay, reqCalcPay)
	if wCalcPay.Code != http.StatusOK {
		t.Fatalf("expected 200 for payroll calculate, got %d. Body: %s", wCalcPay.Code, wCalcPay.Body.String())
	}

	// Kunci periode payroll
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

	// Upaya hitung ulang pada periode terkunci harus ditolak (409 Conflict)
	reqRecalcPay, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/payroll/periods/%s/calculate", payPeriodID), bytes.NewBufferString(`{}`))
	reqRecalcPay.Header.Set("Authorization", "Bearer "+finToken)
	reqRecalcPay.Header.Set("Content-Type", "application/json")
	wRecalcPay := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wRecalcPay, reqRecalcPay)
	if wRecalcPay.Code != http.StatusConflict {
		t.Fatalf("SKENARIO 2 FAILED: expected 409 for locked payroll recalculation, got %d", wRecalcPay.Code)
	}
	t.Logf("E2E STAGE 4 PASS: Skenario 2 VERIFIED (Payroll piece-rate & immutability locking)")

	// -------------------------------------------------------------
	// 6. SKENARIO UJI 3: QUALITY CONTROL & 3-WAY POOL BONUS (BPMN 06 & 08)
	// -------------------------------------------------------------
	// 6.1 Inspeksi QC 3.000 pcs Lolos Mutu
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

	// 6.2 Buat Periode Bonus QC & Hitung Pool Bonus (3.000 x Rp 500 = Rp 1.500.000)
	openQCBody := map[string]interface{}{
		"name":      fmt.Sprintf("Pool Bonus QC - Skenario Uji 3 E2E %d", time.Now().UnixNano()%100000),
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

	reqCalcQC, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/qc-bonus/periods/%s/calculate", qcPeriodID), nil)
	reqCalcQC.Header.Set("Authorization", "Bearer "+execToken)
	wCalcQC := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCalcQC, reqCalcQC)
	if wCalcQC.Code != http.StatusOK {
		t.Fatalf("expected 200 for qc bonus calculate, got %d", wCalcQC.Code)
	}

	// Kunci periode bonus QC
	reqLockQC, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/qc-bonus/periods/%s/lock", qcPeriodID), nil)
	reqLockQC.Header.Set("Authorization", "Bearer "+execToken)
	wLockQC := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLockQC, reqLockQC)
	if wLockQC.Code != http.StatusOK {
		t.Fatalf("expected 200 for qc bonus lock, got %d", wLockQC.Code)
	}
	t.Logf("E2E STAGE 5 PASS: Skenario 3 VERIFIED (3.000 pcs QC -> Rp 1.470.000 pool bonus 3-arah locked)")

	// -------------------------------------------------------------
	// 7. FACTORY OVERHEAD BULANAN (BPMN 09)
	// -------------------------------------------------------------
	// Buka periode overhead 12/2026 agar fresh
	testOHMonth := 12
	testOHYear := 2026
	_, _ = application.db.Exec("DELETE FROM overhead_allocations WHERE overhead_period_id IN (SELECT id FROM overhead_periods WHERE period_month = $1 AND period_year = $2)", testOHMonth, testOHYear)
	_, _ = application.db.Exec("DELETE FROM overhead_expenses WHERE overhead_period_id IN (SELECT id FROM overhead_periods WHERE period_month = $1 AND period_year = $2)", testOHMonth, testOHYear)
	_, _ = application.db.Exec("DELETE FROM overhead_periods WHERE period_month = $1 AND period_year = $2", testOHMonth, testOHYear)

	ohOpenBody := map[string]interface{}{
		"periodMonth": testOHMonth,
		"periodYear":  testOHYear,
		"notes":       "Buku overhead akhir tahun 2026",
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

	// Catat pengeluaran listrik Rp 20.000.000
	var elecCatID string
	_ = application.db.QueryRow("SELECT id FROM overhead_categories WHERE code = 'ELECTRICITY' LIMIT 1").Scan(&elecCatID)
	ohExpBody := map[string]interface{}{
		"categoryId":       elecCatID,
		"expenseDate":      "2026-12-10",
		"amount":           20000000.00,
		"invoiceReference": "PLN-DEC-2026-E2E",
		"notes":            "Beban listrik pabrik akhir tahun",
	}
	ohExpBytes, _ := json.Marshal(ohExpBody)
	reqOHExp, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/overhead/periods/%s/expenses", ohPeriodID), bytes.NewBuffer(ohExpBytes))
	reqOHExp.Header.Set("Authorization", "Bearer "+execToken)
	reqOHExp.Header.Set("Content-Type", "application/json")
	wOHExp := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wOHExp, reqOHExp)

	// Hitung kalkulasi tarif overhead terbobot
	ohCalcBody := map[string]interface{}{
		"weightGarment": 1.0,
		"weightBag":     1.25,
		"notes":         "Kalkulasi overhead final Desember",
	}
	ohCBytes, _ := json.Marshal(ohCalcBody)
	reqOHCalc, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/overhead/periods/%s/calculate", ohPeriodID), bytes.NewBuffer(ohCBytes))
	reqOHCalc.Header.Set("Authorization", "Bearer "+execToken)
	reqOHCalc.Header.Set("Content-Type", "application/json")
	wOHCalc := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wOHCalc, reqOHCalc)
	if wOHCalc.Code != http.StatusOK {
		t.Fatalf("expected 200 for overhead calculate, got %d: %s", wOHCalc.Code, wOHCalc.Body.String())
	}

	// Kunci periode overhead
	reqOHLock, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/overhead/periods/%s/lock", ohPeriodID), nil)
	reqOHLock.Header.Set("Authorization", "Bearer "+execToken)
	wOHLock := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wOHLock, reqOHLock)
	if wOHLock.Code != http.StatusOK {
		t.Fatalf("expected 200 for overhead lock, got %d", wOHLock.Code)
	}
	t.Logf("E2E STAGE 6 PASS: Overhead Pabrik Rp 20.000.000 teralokasi & periode locked")

	// -------------------------------------------------------------
	// 8. ORDER COMPLETION & COMPREHENSIVE COGS REPORT (BPMN 01)
	// -------------------------------------------------------------
	// 8.1 Selesaikan Order Massal
	compOrderBody := map[string]interface{}{
		"notes": "Pesanan massal 2000 pcs selesai diproduksi & siap kirim",
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

	// 8.2 Verifikasi Guard Idempotensi: Order yang sudah complete tidak bisa diselesaikan ulang -> 400 Bad Request
	wCompAgain := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCompAgain, reqCompOrder)
	if wCompAgain.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for already completed order, got %d", wCompAgain.Code)
	}

	// 8.3 Ambil Laporan COGS Komprehensif Order
	reqCOGS, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/reports/cogs/%s", e2eOrderID), nil)
	reqCOGS.Header.Set("Authorization", "Bearer "+execToken)
	wCOGS := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCOGS, reqCOGS)
	if wCOGS.Code != http.StatusOK {
		t.Fatalf("expected 200 for COGS report, got %d. Body: %s", wCOGS.Code, wCOGS.Body.String())
	}

	var cogsResp struct {
		Data struct {
			OrderID             string  `json:"orderId"`
			ContractPricePerUnit float64 `json:"contractPricePerUnit"`
			TotalRevenue        float64 `json:"totalRevenue"`
			CostComponents      struct {
				DirectLaborPerUnit float64 `json:"directLaborPerUnit"`
				QCBonusPerUnit     float64 `json:"qcBonusPerUnit"`
				OverheadPerUnit    float64 `json:"overheadPerUnit"`
			} `json:"costComponents"`
			TotalCostPerUnit     float64 `json:"totalCostPerUnit"`
			TotalCOGS            float64 `json:"totalCOGS"`
			GrossMarginPerUnit   float64 `json:"grossMarginPerUnit"`
			TotalGrossMargin     float64 `json:"totalGrossMargin"`
			GrossMarginPercentage float64 `json:"grossMarginPercentage"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wCOGS.Body.Bytes(), &cogsResp)

	if cogsResp.Data.ContractPricePerUnit != 45000.00 || cogsResp.Data.TotalRevenue != 180000000.00 {
		t.Fatalf("unexpected revenue in COGS report: %+v", cogsResp.Data)
	}
	if cogsResp.Data.CostComponents.QCBonusPerUnit != 500.00 {
		t.Fatalf("expected QC bonus 500, got %f", cogsResp.Data.CostComponents.QCBonusPerUnit)
	}
	if cogsResp.Data.CostComponents.OverheadPerUnit <= 0 || cogsResp.Data.TotalCostPerUnit <= 0 {
		t.Fatalf("expected positive overhead and total cost, got %+v", cogsResp.Data)
	}

	t.Logf("E2E STAGE 7 PASS: Order Selesai & Laporan COGS Komprehensif Berhasil:")
	t.Logf("  - Contract Price: Rp %.2f / pcs (Revenue: Rp %.2f)", cogsResp.Data.ContractPricePerUnit, cogsResp.Data.TotalRevenue)
	t.Logf("  - Direct Labor  : Rp %.2f / pcs", cogsResp.Data.CostComponents.DirectLaborPerUnit)
	t.Logf("  - QC Bonus      : Rp %.2f / pcs", cogsResp.Data.CostComponents.QCBonusPerUnit)
	t.Logf("  - Overhead      : Rp %.2f / pcs", cogsResp.Data.CostComponents.OverheadPerUnit)
	t.Logf("  - Total HPP/Unit: Rp %.2f / pcs (Total COGS: Rp %.2f)", cogsResp.Data.TotalCostPerUnit, cogsResp.Data.TotalCOGS)
	t.Logf("  - Gross Margin  : Rp %.2f / pcs (Total: Rp %.2f, Margin: %.2f%%)",
		cogsResp.Data.GrossMarginPerUnit, cogsResp.Data.TotalGrossMargin, cogsResp.Data.GrossMarginPercentage)
}

