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

type PostmanItem struct {
	Name     string                 `json:"name"`
	Event    []map[string]interface{} `json:"event,omitempty"`
	Request  map[string]interface{} `json:"request"`
	Response []map[string]interface{} `json:"response"`
}

type PostmanFolder struct {
	Name     string        `json:"name"`
	Item     []PostmanItem `json:"item"`
	Response interface{}   `json:"response"`
}

func TestApp_GenerateOverheadPostmanCollection(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Login Executive
	loginBody := map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	}
	bodyBytes, _ := json.Marshal(loginBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for login, got %d: %s", w.Code, w.Body.String())
	}
	var loginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &loginResp)
	token := loginResp.Data.Token

	// 16.1 GET /overhead/categories
	reqCats, _ := http.NewRequest(http.MethodGet, "/api/v1/overhead/categories", nil)
	reqCats.Header.Set("Authorization", "Bearer "+token)
	wCats := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCats, reqCats)
	if wCats.Code != http.StatusOK {
		t.Fatalf("expected 200 for categories, got %d", wCats.Code)
	}
	respCatsBody := wCats.Body.String()

	var catData struct {
		Data []struct {
			ID   string `json:"id"`
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wCats.Body.Bytes(), &catData)

	elecCatID := ""
	rentCatID := ""
	for _, c := range catData.Data {
		if c.Code == "ELECTRICITY" {
			elecCatID = c.ID
		}
		if c.Code == "RENT" {
			rentCatID = c.ID
		}
	}
	if elecCatID == "" && len(catData.Data) > 0 {
		elecCatID = catData.Data[0].ID
	}
	if rentCatID == "" && len(catData.Data) > 1 {
		rentCatID = catData.Data[1].ID
	}

	// 16.2 POST /overhead/categories
	customCode := fmt.Sprintf("SECURITY_SERVICES_%d", time.Now().Unix()%10000)
	customBodyMap := map[string]interface{}{
		"code":        customCode,
		"name":        "Jasa Pengamanan & Sekuriti Pabrik",
		"description": "Biaya jasa sekuriti dan penjagaan fasilitas pabrik operasional bulanan",
		"isDefault":   false,
	}
	customJSON, _ := json.Marshal(customBodyMap)
	reqCustom, _ := http.NewRequest(http.MethodPost, "/api/v1/overhead/categories", bytes.NewBuffer(customJSON))
	reqCustom.Header.Set("Authorization", "Bearer "+token)
	reqCustom.Header.Set("Content-Type", "application/json")
	wCustom := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCustom, reqCustom)
	if wCustom.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create category, got %d: %s", wCustom.Code, wCustom.Body.String())
	}
	respCustomBody := wCustom.Body.String()

	// Bersihkan periode 10/2026 jika sudah ada
	_, _ = application.db.Exec("DELETE FROM overhead_allocations WHERE overhead_period_id IN (SELECT id FROM overhead_periods WHERE period_month = 10 AND period_year = 2026)")
	_, _ = application.db.Exec("DELETE FROM overhead_expenses WHERE overhead_period_id IN (SELECT id FROM overhead_periods WHERE period_month = 10 AND period_year = 2026)")
	_, _ = application.db.Exec("DELETE FROM overhead_periods WHERE period_month = 10 AND period_year = 2026")

	// 16.3 POST /overhead/periods (Buka Periode Oktober 2026)
	openPeriodMap := map[string]interface{}{
		"periodMonth": 10,
		"periodYear":  2026,
		"notes":       "Pembukuan Biaya Overhead Pabrik Periode Oktober 2026",
	}
	openPeriodJSON, _ := json.Marshal(openPeriodMap)
	reqOpen, _ := http.NewRequest(http.MethodPost, "/api/v1/overhead/periods", bytes.NewBuffer(openPeriodJSON))
	reqOpen.Header.Set("Authorization", "Bearer "+token)
	reqOpen.Header.Set("Content-Type", "application/json")
	wOpen := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wOpen, reqOpen)
	if wOpen.Code != http.StatusCreated {
		t.Fatalf("expected 201 for open period, got %d: %s", wOpen.Code, wOpen.Body.String())
	}
	respOpenBody := wOpen.Body.String()

	var periodResp struct {
		Data struct {
			ID         string `json:"id"`
			PeriodCode string `json:"periodCode"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wOpen.Body.Bytes(), &periodResp)
	overheadPeriodID := periodResp.Data.ID

	// 16.4 GET /overhead/periods
	reqListPeriods, _ := http.NewRequest(http.MethodGet, "/api/v1/overhead/periods?year=2026", nil)
	reqListPeriods.Header.Set("Authorization", "Bearer "+token)
	wListPeriods := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wListPeriods, reqListPeriods)
	if wListPeriods.Code != http.StatusOK {
		t.Fatalf("expected 200 for list periods, got %d", wListPeriods.Code)
	}
	respListPeriodsBody := wListPeriods.Body.String()

	// 16.5 GET /overhead/periods/:id
	reqPeriodDetail, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/overhead/periods/%s", overheadPeriodID), nil)
	reqPeriodDetail.Header.Set("Authorization", "Bearer "+token)
	wPeriodDetail := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wPeriodDetail, reqPeriodDetail)
	if wPeriodDetail.Code != http.StatusOK {
		t.Fatalf("expected 200 for period detail, got %d", wPeriodDetail.Code)
	}
	respPeriodDetailBody := wPeriodDetail.Body.String()

	// 16.6 POST /overhead/periods/:id/expenses (Listrik & Air)
	exp1Map := map[string]interface{}{
		"categoryId":       elecCatID,
		"expenseDate":      "2026-10-10",
		"amount":           15000000.00,
		"invoiceReference": "PLN-OKT-2026-883",
		"notes":            "Tagihan listrik PLN mesin jahit dan kompresor pabrik",
	}
	exp1JSON, _ := json.Marshal(exp1Map)
	reqExp1, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/overhead/periods/%s/expenses", overheadPeriodID), bytes.NewBuffer(exp1JSON))
	reqExp1.Header.Set("Authorization", "Bearer "+token)
	reqExp1.Header.Set("Content-Type", "application/json")
	wExp1 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wExp1, reqExp1)
	if wExp1.Code != http.StatusCreated {
		t.Fatalf("expected 201 for expense 1, got %d: %s", wExp1.Code, wExp1.Body.String())
	}
	respExp1Body := wExp1.Body.String()

	var exp1Resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wExp1.Body.Bytes(), &exp1Resp)
	overheadExpenseID := exp1Resp.Data.ID

	// 16.7 POST /overhead/periods/:id/expenses (Sewa Gedung)
	exp2Map := map[string]interface{}{
		"categoryId":       rentCatID,
		"expenseDate":      "2026-10-01",
		"amount":           10000000.00,
		"invoiceReference": "RENT-OKT-2026-01",
		"notes":            "Alokasi sewa bulanan fasilitas workshop produksi",
	}
	exp2JSON, _ := json.Marshal(exp2Map)
	reqExp2, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/overhead/periods/%s/expenses", overheadPeriodID), bytes.NewBuffer(exp2JSON))
	reqExp2.Header.Set("Authorization", "Bearer "+token)
	reqExp2.Header.Set("Content-Type", "application/json")
	wExp2 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wExp2, reqExp2)
	if wExp2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for expense 2, got %d: %s", wExp2.Code, wExp2.Body.String())
	}
	respExp2Body := wExp2.Body.String()

	// 16.8 POST /overhead/periods/:id/calculate
	calcMap := map[string]interface{}{
		"weightGarment": 1.00,
		"weightBag":     1.25,
		"notes":         "Kalkulasi final alokasi overhead periode Oktober 2026",
	}
	calcJSON, _ := json.Marshal(calcMap)
	reqCalc, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/overhead/periods/%s/calculate", overheadPeriodID), bytes.NewBuffer(calcJSON))
	reqCalc.Header.Set("Authorization", "Bearer "+token)
	reqCalc.Header.Set("Content-Type", "application/json")
	wCalc := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCalc, reqCalc)
	if wCalc.Code != http.StatusOK {
		t.Fatalf("expected 200 for calculate, got %d: %s", wCalc.Code, wCalc.Body.String())
	}
	respCalcBody := wCalc.Body.String()

	// 16.9 GET /overhead/periods/:id/summary
	reqSum, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/overhead/periods/%s/summary", overheadPeriodID), nil)
	reqSum.Header.Set("Authorization", "Bearer "+token)
	wSum := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSum, reqSum)
	if wSum.Code != http.StatusOK {
		t.Fatalf("expected 200 for summary, got %d: %s", wSum.Code, wSum.Body.String())
	}
	respSumBody := wSum.Body.String()

	// 16.10 POST /overhead/periods/:id/lock
	reqLock, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/overhead/periods/%s/lock", overheadPeriodID), nil)
	reqLock.Header.Set("Authorization", "Bearer "+token)
	wLock := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLock, reqLock)
	if wLock.Code != http.StatusOK {
		t.Fatalf("expected 200 for lock, got %d: %s", wLock.Code, wLock.Body.String())
	}
	respLockBody := wLock.Body.String()

	// 16.11 GET /reports/cogs/:orderId
	var massOrderID string
	_ = application.db.QueryRow("SELECT id FROM orders ORDER BY created_at DESC LIMIT 1").Scan(&massOrderID)
	reqCOGS, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/reports/cogs/%s", massOrderID), nil)
	reqCOGS.Header.Set("Authorization", "Bearer "+token)
	wCOGS := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCOGS, reqCOGS)
	if wCOGS.Code != http.StatusOK {
		t.Fatalf("expected 200 for COGS report, got %d: %s", wCOGS.Code, wCOGS.Body.String())
	}
	respCOGSBody := wCOGS.Body.String()

	// Bangun Postman Folder 16
	folder16 := PostmanFolder{
		Name:     "16. Tahap 7.1: Factory Overhead Engine & Cost Allocation (BPMN 09)",
		Response: nil,
		Item: []PostmanItem{
			{
				Name: "16.1 GET Daftar Master Kategori Biaya Overhead",
				Request: map[string]interface{}{
					"description": "Mengambil seluruh kategori biaya overhead operasional pabrik yang aktif.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
					},
					"method": "GET",
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"overhead", "categories"},
						"raw":  "{{base_url}}/overhead/categories",
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
								"pm.test(\"Overhead categories list returned\", function () {",
								"    pm.expect(jsonData.success).to.eql(true);",
								"    pm.expect(jsonData.data.length).to.be.above(0);",
								"});",
								"var elec = jsonData.data.find(function(c) { return c.code === 'ELECTRICITY'; });",
								"if (elec) { pm.collectionVariables.set('overhead_category_id', elec.id); }",
								"var rent = jsonData.data.find(function(c) { return c.code === 'RENT'; });",
								"if (rent) { pm.collectionVariables.set('overhead_rent_category_id', rent.id); }",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respCatsBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "16.2 POST Daftarkan Kategori Overhead Kustom",
				Request: map[string]interface{}{
					"description": "Menambahkan kategori biaya overhead kustom baru yang spesifik untuk pabrik.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"code\": \"SECURITY_SERVICES\",\n    \"name\": \"Jasa Pengamanan & Sekuriti Pabrik\",\n    \"description\": \"Biaya jasa sekuriti dan penjagaan fasilitas pabrik operasional bulanan\",\n    \"isDefault\": false\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"overhead", "categories"},
						"raw":  "{{base_url}}/overhead/categories",
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
								"pm.test(\"Category created successfully\", function () {",
								"    pm.expect(jsonData.success).to.eql(true);",
								"    pm.expect(jsonData.data.id).to.be.a('string');",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respCustomBody,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "16.3 POST Buka Periode Overhead Bulanan Baru",
				Request: map[string]interface{}{
					"description": "Membuka periode buku alokasi biaya overhead bulanan baru dengan status OPEN.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"periodMonth\": 10,\n    \"periodYear\": 2026,\n    \"notes\": \"Pembukuan Biaya Overhead Pabrik Periode Oktober 2026\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"overhead", "periods"},
						"raw":  "{{base_url}}/overhead/periods",
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
								"pm.test(\"Overhead period opened with OPEN status\", function () {",
								"    pm.expect(jsonData.success).to.eql(true);",
								"    pm.expect(jsonData.data.status).to.eql('OPEN');",
								"    pm.collectionVariables.set('overhead_period_id', jsonData.data.id);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respOpenBody,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "16.4 GET Daftar Seluruh Periode Overhead",
				Request: map[string]interface{}{
					"description": "Melihat riwayat seluruh periode overhead bulanan yang tercatat di sistem.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
					},
					"method": "GET",
					"url": map[string]interface{}{
						"host":  []string{"{{base_url}}"},
						"path":  []string{"overhead", "periods"},
						"query": []map[string]interface{}{{"key": "year", "value": "2026"}},
						"raw":   "{{base_url}}/overhead/periods?year=2026",
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
						"body":                     respListPeriodsBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "16.5 GET Detail Periode Overhead Aktif",
				Request: map[string]interface{}{
					"description": "Mengambil rincian periode overhead lengkap dengan daftar biaya pengeluaran dan ringkasan tarif.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
					},
					"method": "GET",
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"overhead", "periods", "{{overhead_period_id}}"},
						"raw":  "{{base_url}}/overhead/periods/{{overhead_period_id}}",
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
						"body":                     respPeriodDetailBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "16.6 POST Catat Biaya Pengeluaran 1 - Tagihan Listrik & Air",
				Request: map[string]interface{}{
					"description": "Mencatat biaya riil operasional utilitas listrik & air pabrik ke periode aktif.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"categoryId\": \"{{overhead_category_id}}\",\n    \"expenseDate\": \"2026-10-10\",\n    \"amount\": 15000000,\n    \"invoiceReference\": \"PLN-OKT-2026-883\",\n    \"notes\": \"Tagihan listrik PLN mesin jahit dan kompresor pabrik\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"overhead", "periods", "{{overhead_period_id}}", "expenses"},
						"raw":  "{{base_url}}/overhead/periods/{{overhead_period_id}}/expenses",
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
								"pm.test(\"Expense recorded successfully\", function () {",
								"    pm.expect(jsonData.success).to.eql(true);",
								"    pm.expect(jsonData.data.amount).to.eql(15000000);",
								"    pm.collectionVariables.set('overhead_expense_id', jsonData.data.id);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respExp1Body,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "16.7 POST Catat Biaya Pengeluaran 2 - Sewa Gedung Pabrik",
				Request: map[string]interface{}{
					"description": "Mencatat biaya riil sewa gedung fasilitas produksi ke periode aktif.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"categoryId\": \"{{overhead_rent_category_id}}\",\n    \"expenseDate\": \"2026-10-01\",\n    \"amount\": 10000000,\n    \"invoiceReference\": \"RENT-OKT-2026-01\",\n    \"notes\": \"Alokasi sewa bulanan fasilitas workshop produksi\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"overhead", "periods", "{{overhead_period_id}}", "expenses"},
						"raw":  "{{base_url}}/overhead/periods/{{overhead_period_id}}/expenses",
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
								"pm.test(\"Expense recorded successfully\", function () {",
								"    pm.expect(jsonData.success).to.eql(true);",
								"    pm.expect(jsonData.data.amount).to.eql(10000000);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respExp2Body,
						"code":                     201,
						"name":                     "Success 201 Created",
						"status":                   "Created",
					},
				},
			},
			{
				Name: "16.8 POST Kalkulasi Alokasi Tarif Overhead Flat & Terbobot",
				Request: map[string]interface{}{
					"description": "Menjalankan kalkulasi tarif overhead flat rate dan weighted rate (Garment bobot 1.0 vs Tas bobot 1.25) terhadap total unit lolos QC.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
						{"key": "Content-Type", "type": "text", "value": "application/json"},
					},
					"method": "POST",
					"body": map[string]interface{}{
						"mode": "raw",
						"raw":  "{\n    \"weightGarment\": 1.0,\n    \"weightBag\": 1.25,\n    \"notes\": \"Kalkulasi final alokasi overhead periode Oktober 2026\"\n}",
					},
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"overhead", "periods", "{{overhead_period_id}}", "calculate"},
						"raw":  "{{base_url}}/overhead/periods/{{overhead_period_id}}/calculate",
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
								"pm.test(\"Overhead calculation returns calculated status and rates\", function () {",
								"    pm.expect(jsonData.success).to.eql(true);",
								"    pm.expect(jsonData.data.status).to.eql('CALCULATED');",
								"    pm.expect(jsonData.data.totalExpenseAmount).to.eql(25000000);",
								"    pm.expect(jsonData.data.flatRatePerUnit).to.be.above(0);",
								"    pm.expect(jsonData.data.bagRatePerUnit).to.be.above(jsonData.data.garmentRatePerUnit);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respCalcBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "16.9 GET Laporan Ringkasan Alokasi Overhead Bulanan",
				Request: map[string]interface{}{
					"description": "Menampilkan rekapitulasi persentase pengeluaran per kategori, total unit produksi, dan ringkasan tarif.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
					},
					"method": "GET",
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"overhead", "periods", "{{overhead_period_id}}", "summary"},
						"raw":  "{{base_url}}/overhead/periods/{{overhead_period_id}}/summary",
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
								"pm.test(\"Summary report contains expense breakdown and rates\", function () {",
								"    pm.expect(jsonData.success).to.eql(true);",
								"    pm.expect(jsonData.data.expenseBreakdown.length).to.be.above(0);",
								"    pm.expect(jsonData.data.totalExpenseAmount).to.eql(25000000);",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respSumBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "16.10 POST Kunci Periode Overhead Bulanan (Lock & Immutable)",
				Request: map[string]interface{}{
					"description": "Mengunci periode overhead menjadi status LOCKED sehingga tidak dapat diubah atau ditambah biaya lagi.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
					},
					"method": "POST",
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"overhead", "periods", "{{overhead_period_id}}", "lock"},
						"raw":  "{{base_url}}/overhead/periods/{{overhead_period_id}}/lock",
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
								"pm.test(\"Period successfully locked\", function () {",
								"    pm.expect(jsonData.success).to.eql(true);",
								"    pm.expect(jsonData.data.status).to.eql('LOCKED');",
								"});",
							},
							"type": "text/javascript",
						},
					},
				},
				Response: []map[string]interface{}{
					{
						"_postman_previewlanguage": "json",
						"body":                     respLockBody,
						"code":                     200,
						"name":                     "Success 200 OK",
						"status":                   "OK",
					},
				},
			},
			{
				Name: "16.11 GET Laporan HPP / COGS Komprehensif Order (BPMN 01)",
				Request: map[string]interface{}{
					"description": "Menampilkan analisis HPP komprehensif order mencakup Biaya Tenaga Kerja Langsung, QC Bonus, Alokasi Overhead Pabrik, dan Gross Profit Margin.",
					"header": []map[string]interface{}{
						{"key": "Authorization", "type": "text", "value": "Bearer {{auth_token}}"},
					},
					"method": "GET",
					"url": map[string]interface{}{
						"host": []string{"{{base_url}}"},
						"path": []string{"reports", "cogs", "{{mass_order_id}}"},
						"raw":  "{{base_url}}/reports/cogs/{{mass_order_id}}",
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
								"pm.test(\"COGS report verifies labor, qc bonus, and overhead components\", function () {",
								"    pm.expect(jsonData.success).to.eql(true);",
								"    pm.expect(jsonData.data.costComponents.directLaborPerUnit).to.be.above(0);",
								"    pm.expect(jsonData.data.costComponents.overheadPerUnit).to.be.above(0);",
								"    pm.expect(jsonData.data.totalCostPerUnit).to.be.above(0);",
								"    pm.expect(jsonData.data.grossMarginPerUnit).to.be.a('number');",
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

	// Update Postman collection file
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

	// Filter out Folder 16 if already present, to avoid duplication
	newItems := make([]interface{}, 0, len(items)+1)
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if ok {
			name, _ := m["name"].(string)
			if len(name) >= 2 && name[:2] == "16" {
				continue
			}
		}
		newItems = append(newItems, it)
	}

	// Marshal folder 16 to interface{}
	f16Bytes, _ := json.Marshal(folder16)
	var f16Map map[string]interface{}
	_ = json.Unmarshal(f16Bytes, &f16Map)

	newItems = append(newItems, f16Map)
	rootCollection["item"] = newItems

	// Ensure variables are in place
	vars, _ := rootCollection["variable"].([]interface{})
	varMap := make(map[string]bool)
	for _, v := range vars {
		vm, _ := v.(map[string]interface{})
		k, _ := vm["key"].(string)
		varMap[k] = true
	}

	if !varMap["overhead_period_id"] {
		vars = append(vars, map[string]interface{}{"key": "overhead_period_id", "type": "string", "value": overheadPeriodID})
	} else {
		for i, v := range vars {
			vm, _ := v.(map[string]interface{})
			if vm["key"] == "overhead_period_id" {
				vars[i] = map[string]interface{}{"key": "overhead_period_id", "type": "string", "value": overheadPeriodID}
			}
		}
	}
	if !varMap["overhead_category_id"] {
		vars = append(vars, map[string]interface{}{"key": "overhead_category_id", "type": "string", "value": elecCatID})
	}
	if !varMap["overhead_rent_category_id"] {
		vars = append(vars, map[string]interface{}{"key": "overhead_rent_category_id", "type": "string", "value": rentCatID})
	}
	if !varMap["overhead_expense_id"] {
		vars = append(vars, map[string]interface{}{"key": "overhead_expense_id", "type": "string", "value": overheadExpenseID})
	}
	rootCollection["variable"] = vars

	// Write back
	outputBytes, err := json.MarshalIndent(rootCollection, "", "    ")
	if err != nil {
		t.Fatalf("marshal output error: %v", err)
	}

	if err := os.WriteFile(postmanPath, outputBytes, 0644); err != nil {
		t.Fatalf("write postman file error: %v", err)
	}

	t.Logf("Tahap 7.1 PASS: Postman Collection berhasil disinkronisasi dengan 11 endpoint dan respon aktual!")
}
