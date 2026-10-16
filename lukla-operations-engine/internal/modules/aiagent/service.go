package aiagent

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// Config holds configuration parameters for AWS Bedrock Runtime.
type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	ModelID         string
}

// Service provides AI Agent orchestration, task decomposition, and streaming inference.
type Service struct {
	db             *sql.DB
	tools          *ToolsExecutor
	bedrockRuntime *bedrockruntime.Client
	cfg            Config
}

// NewService instantiates the AI Agent service with AWS Bedrock Runtime client.
func NewService(ctx context.Context, db *sql.DB, cfg Config) (*Service, error) {
	if cfg.Region == "" {
		cfg.Region = "ap-southeast-1"
	}
	if cfg.ModelID == "" {
		cfg.ModelID = "amazon.nova-pro-v1:0"
	}

	var optFns []func(*awsconfig.LoadOptions) error
	optFns = append(optFns, awsconfig.WithRegion(cfg.Region))

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		optFns = append(optFns, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for Bedrock: %w", err)
	}

	svc := &Service{
		db:             db,
		tools:          NewToolsExecutor(db),
		bedrockRuntime: bedrockruntime.NewFromConfig(awsCfg),
		cfg:            cfg,
	}

	return svc, nil
}

// GetHealth returns connectivity and resource metadata.
func (s *Service) GetHealth() HealthStatus {
	return HealthStatus{
		Status:          "ACTIVE",
		Engine:          "Go Gin + Amazon Bedrock ConverseStream (PostgreSQL Direct Grounded)",
		Region:          s.cfg.Region,
		ModelID:         s.cfg.ModelID,
		GuardrailActive:  true,
		AvailableTools: []string{
			"get_operational_stats",
			"track_production_order",
			"analyze_line_performance",
			"get_operational_recommendations",
			"check_inventory_stock",
		},
		DatabaseEngine: "PostgreSQL (Prisma + operational_entities)",
		Timestamp:      time.Now(),
	}
}

// StreamAgenticLoop executes the autonomous Decomposer -> Tools -> ConverseStream loop.
func (s *Service) StreamAgenticLoop(ctx context.Context, req AgentRequest, sendEvent func(AgentStreamEvent) error) error {
	startTime := time.Now()
	userPrompt := req.GetUserPrompt()

	// 1. Guardrail Domain Validation
	isValid, refusalMsg := ValidateLuklaDomain(userPrompt)
	if !isValid {
		_ = sendEvent(AgentStreamEvent{
			Type: "token",
			Data: map[string]string{"text": refusalMsg},
		})
		_ = sendEvent(AgentStreamEvent{
			Type: "done",
			Data: AgentFinalResult{
				FullText:        refusalMsg,
				ModelID:         s.cfg.ModelID,
				ExecutionTimeMs: time.Since(startTime).Milliseconds(),
			},
		})
		return nil
	}

	// 2. Decompose user query into structured subtasks
	plan := s.decomposeQuery(userPrompt, req.ContextRole)
	if err := sendEvent(AgentStreamEvent{Type: "plan", Data: plan}); err != nil {
		return err
	}

	// 3. Execute SubTasks (Database Tools)
	var executedTools []ToolResult
	var dataContextBuilder strings.Builder

	for _, task := range plan.SubTasks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if task.Type == "DATABASE_TOOL" {
			_ = sendEvent(AgentStreamEvent{
				Type: "tool_call",
				Data: map[string]any{"tool": task.ToolName, "parameters": task.Parameters},
			})

			res := s.tools.ExecuteTool(ctx, task.ToolName, task.Parameters, req.ContextRole, req.ClientID)
			executedTools = append(executedTools, res)

			_ = sendEvent(AgentStreamEvent{Type: "tool_result", Data: res})

			if res.Success && res.Count > 0 {
				recBytes, _ := json.MarshalIndent(res.Records, "", "  ")
				dataContextBuilder.WriteString(fmt.Sprintf("\n[HASIL DATABASE TOOL: %s]\n%s\n", task.ToolName, string(recBytes)))
			}
		}
	}

	// 4. Construct System Prompt & Messages for Bedrock Nova Pro
	systemPrompt := s.buildSystemPrompt(req.ContextRole, dataContextBuilder.String())
	messages := s.buildConverseMessages(req.Messages, userPrompt)

	// 5. Call Amazon Bedrock ConverseStream
	activeModel := s.cfg.ModelID
	// In ap-southeast-1, Nova Pro requires regional/cross-region prefix if using APAC inference profile
	if !strings.HasPrefix(activeModel, "apac.") && s.cfg.Region == "ap-southeast-1" && strings.Contains(activeModel, "nova") {
		activeModel = "apac." + activeModel
	}

	converseInput := &bedrockruntime.ConverseStreamInput{
		ModelId: aws.String(activeModel),
		System: []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{Value: systemPrompt},
		},
		Messages: messages,
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens:   aws.Int32(2048),
			Temperature: aws.Float32(0.7),
		},
	}

	streamOutput, err := s.bedrockRuntime.ConverseStream(ctx, converseInput)
	if err != nil {
		// Fallback to standard model ID without prefix if inference profile fails
		log.Printf("[WARN] Bedrock ConverseStream failed with %s: %v. Retrying with fallback...", activeModel, err)
		fallbackModel := s.cfg.ModelID
		converseInput.ModelId = aws.String(fallbackModel)
		streamOutput, err = s.bedrockRuntime.ConverseStream(ctx, converseInput)
		if err != nil {
			errEvent := AgentStreamEvent{
				Type: "error",
				Data: map[string]string{"message": fmt.Sprintf("Gagal memanggil Amazon Bedrock (%s): %v", fallbackModel, err)},
			}
			_ = sendEvent(errEvent)
			return err
		}
		activeModel = fallbackModel
	}

	// 6. Stream token deltas back to client
	var fullAccumulated strings.Builder
	eventStream := streamOutput.GetStream()
	defer eventStream.Close()

	for event := range eventStream.Events() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		switch v := event.(type) {
		case *types.ConverseStreamOutputMemberContentBlockDelta:
			if delta, ok := v.Value.Delta.(*types.ContentBlockDeltaMemberText); ok {
				fullAccumulated.WriteString(delta.Value)
				_ = sendEvent(AgentStreamEvent{
					Type: "token",
					Data: map[string]string{"text": delta.Value},
				})
			}
		}
	}

	if err := eventStream.Err(); err != nil {
		log.Printf("[WARN] Bedrock event stream terminated with error: %v", err)
	}

	// 7. Send "done" event
	finalRes := AgentFinalResult{
		FullText:          fullAccumulated.String(),
		ModelID:           activeModel,
		Plan:              &plan,
		Citations:         nil, // No external AWS KB needed
		ExecutedTools:     executedTools,
		TotalRecordsFound: len(executedTools),
		ExecutionTimeMs:   time.Since(startTime).Milliseconds(),
	}
	_ = sendEvent(AgentStreamEvent{Type: "done", Data: finalRes})

	return nil
}

// decomposeQuery splits the question into targeted sub-tasks.
func (s *Service) decomposeQuery(query string, role string) DecomposedPlan {
	lower := strings.ToLower(query)
	var subTasks []SubTask
	var reasons []string

	// Check if operational stats are requested
	if strings.Contains(lower, "statistik") || strings.Contains(lower, "rekap") ||
		strings.Contains(lower, "ringkasan") || strings.Contains(lower, "output") ||
		strings.Contains(lower, "utilitas") || strings.Contains(lower, "overview") ||
		strings.Contains(lower, "eksekutif") {
		subTasks = append(subTasks, SubTask{
			ID:       fmt.Sprintf("task-stats-%d", time.Now().UnixNano()),
			Type:     "DATABASE_TOOL",
			ToolName: "get_operational_stats",
			Purpose:  "Mengambil statistik operasional lini jahit & ringkasan output pabrik",
		})
		reasons = append(reasons, "Membutuhkan data statistik dan agregasi KPI operasional")
	}

	// Check if order tracking is requested
	if strings.Contains(lower, "pesanan") || strings.Contains(lower, "spk") ||
		strings.Contains(lower, "po") || strings.Contains(lower, "order") ||
		strings.Contains(lower, "tracking") || strings.Contains(lower, "lacak") ||
		strings.Contains(lower, "parka") || strings.Contains(lower, "snøhund") || strings.Contains(lower, "snohund") {
		search := "PO-2026-10-001"
		if strings.Contains(lower, "parka") {
			search = "Parka"
		} else if strings.Contains(lower, "snohund") || strings.Contains(lower, "snøhund") {
			search = "Snøhund"
		}
		subTasks = append(subTasks, SubTask{
			ID:         fmt.Sprintf("task-order-%d", time.Now().UnixNano()),
			Type:       "DATABASE_TOOL",
			ToolName:   "track_production_order",
			Parameters: map[string]any{"search": search},
			Purpose:    fmt.Sprintf("Melacak progres dan milestone pesanan \"%s\"", search),
		})
		reasons = append(reasons, "Membutuhkan pelacakan rincian pesanan dan status SPK")
	}

	// Check if line performance / defect analysis is requested
	if strings.Contains(lower, "line") || strings.Contains(lower, "lini") ||
		strings.Contains(lower, "performa") || strings.Contains(lower, "efisiensi") ||
		strings.Contains(lower, "defect") || strings.Contains(lower, "kualitas") ||
		strings.Contains(lower, "qc") || strings.Contains(lower, "cacat") {
		lineCode := "LINE-01"
		if strings.Contains(lower, "line 2") || strings.Contains(lower, "line-02") {
			lineCode = "LINE-02"
		} else if strings.Contains(lower, "line 3") || strings.Contains(lower, "line-03") {
			lineCode = "LINE-03"
		}
		subTasks = append(subTasks, SubTask{
			ID:         fmt.Sprintf("task-line-%d", time.Now().UnixNano()),
			Type:       "DATABASE_TOOL",
			ToolName:   "analyze_line_performance",
			Parameters: map[string]any{"lineCode": lineCode},
			Purpose:    fmt.Sprintf("Menganalisis performa, efisiensi waktu, dan defect rate pada %s", lineCode),
		})
		reasons = append(reasons, "Membutuhkan analisa performa dan kontrol kualitas lini")
	}

	// Check if recommendations are requested
	if strings.Contains(lower, "rekomendasi") || strings.Contains(lower, "saran") ||
		strings.Contains(lower, "tindakan") || strings.Contains(lower, "solusi") ||
		strings.Contains(lower, "perbaikan") || strings.Contains(lower, "bottleneck") {
		subTasks = append(subTasks, SubTask{
			ID:       fmt.Sprintf("task-rec-%d", time.Now().UnixNano()),
			Type:     "DATABASE_TOOL",
			ToolName: "get_operational_recommendations",
			Purpose:  "Mengidentifikasi bottleneck dan menyusun rekomendasi tindakan korektif operasional",
		})
		reasons = append(reasons, "Membutuhkan rekomendasi aksi pencegahan dan penyeimbangan operasional")
	}

	// Check if raw materials stock is requested
	if strings.Contains(lower, "stok") || strings.Contains(lower, "kain") ||
		strings.Contains(lower, "taslan") || strings.Contains(lower, "cordura") ||
		strings.Contains(lower, "resleting") || strings.Contains(lower, "zipper") ||
		strings.Contains(lower, "ykk") || strings.Contains(lower, "gudang") || strings.Contains(lower, "rak") {
		search := "Taslan"
		if strings.Contains(lower, "cordura") {
			search = "Cordura"
		} else if strings.Contains(lower, "resleting") || strings.Contains(lower, "zipper") || strings.Contains(lower, "ykk") {
			search = "YKK"
		}
		subTasks = append(subTasks, SubTask{
			ID:         fmt.Sprintf("task-inv-%d", time.Now().UnixNano()),
			Type:       "DATABASE_TOOL",
			ToolName:   "check_inventory_stock",
			Parameters: map[string]any{"search": search},
			Purpose:    fmt.Sprintf("Mengecek stok fisik bahan \"%s\" dan lokasi rak gudang", search),
		})
		reasons = append(reasons, "Membutuhkan pengecekan kuantitas fisik material gudang")
	}

	// If no specific tools matched, default to operational stats
	if len(subTasks) == 0 {
		subTasks = append(subTasks, SubTask{
			ID:       fmt.Sprintf("task-default-%d", time.Now().UnixNano()),
			Type:     "DATABASE_TOOL",
			ToolName: "get_operational_stats",
			Purpose:  "Mengambil ikhtisar operasional pabrik terkini",
		})
		reasons = append(reasons, "Menyajikan data konteks operasional pabrik secara umum")
	}

	return DecomposedPlan{
		OriginalQuery: query,
		IsComplex:     len(subTasks) > 1,
		Reasoning:     strings.Join(reasons, "; "),
		SubTasks:      subTasks,
	}
}

// buildSystemPrompt creates tailored instructions for Amazon Nova Pro based on domain knowledge.
func (s *Service) buildSystemPrompt(role, dataContext string) string {
	roleContext := "Level Akses: Eksekutif Pabrik (Full Overview)"
	if role == RoleClientPortal {
		roleContext = "Level Akses: Klien Maklon B2B (Data terbatas pada pesanan milik klien terkait, jangan tampilkan data finansial internal/overhead pabrik)."
	}

	return fmt.Sprintf(`Anda adalah "Lukla Factory AI Copilot", asisten intelijen operasional resmi untuk fasilitas manufaktur garmen teknis & tas Lukla di Ciwidey, Jawa Barat.

PROFIL PABRIK & STANDAR OPERASIONAL LUKLA (SRS & BPMN):
- Fasilitas: Ciwidey Technical Garment & Bag Atelier, Bandung Selatan.
- Spesialisasi: Jaket Alpine Waterproof, Parka Ekspedisi 3-Layer (Gore-Tex, Taslan), Tas Carrier Cordura, Seam Sealing Tape 20mm, Resleting YKK Aquaguard.
- Alur 4 Tahap Maklon:
  1. Sampling & R&D (Tech Pack, Component Time Study, Standard Allowed Minutes / SAM).
  2. Inventory & Pemotongan Bahan (Rak Gudang RK-A s/d RK-C, Fabric Spreading, Cutting).
  3. Lini Jahit & Perakitan (Line 1 Parka Utama, Line 2 Tas & Carrier, Line 3 Sub-Assembly).
  4. Quality Control & Poka-Yoke (Pemeriksaan 100%%, Stiker Panah Merah untuk Rework/Repair, Label Hijau QC Passed ber-QR unik untuk verifikasi lolos).
- Standar Mutu QC: Defect rate maksimum yang dapat diterima adalah < 2.0%%. Cacat dominan yang dipantau meliputi jahitan loncat (skip stitch), kerutan seam sealing, dan jahitan meleset.

TUGAS UTAMA ANDA:
1. Statistik Operasional: Sajikan rekap output harian, utilisasi line jahit, dan progres pesanan secara akurat pada tingkat eksekutif dan klien.
2. Tracking Pesanan: Lacak nomor SPK, status pengerjaan, kuantitas target vs selesai, dan estimasi waktu penyelesaian.
3. Analisa Performa Operasional: Bandingkan efisiensi line jahit, utilisasi SAM, serta rincian defect rate.
4. Rekomendasi Operasional: Berikan saran tindakan pencegahan cacat mutu, reorder bahan baku gudang, atau rebalancing lini.

PANDUAN KETAT:
- HANYA jawab pertanyaan seputar operasional manufaktur, produk garmen teknis/tas, dan data pabrik Lukla Ciwidey.
- Gunakan data aktual di bawah ini sebagai sumber kebenaran utama. Jangan berhalusinasi.
- Tampilkan jawaban secara profesional, terstruktur rapi dengan Markdown (tabel, poin, kutipan penting).
- Selalu cantumkan fakta konkret (angka kuantitas pcs, persentase efisiensi, kode lini LINE-01 dsb.).
- %s

DATA OPERASIONAL AKTUAL DARI DATABASE PABRIK:
%s
`, roleContext, dataContext)
}

// buildConverseMessages prepares past history and the current user prompt.
func (s *Service) buildConverseMessages(history []ChatMessage, currentPrompt string) []types.Message {
	var msgs []types.Message

	for _, h := range history {
		if h.Content == "" {
			continue
		}
		var role types.ConversationRole
		if h.Role == "assistant" {
			role = types.ConversationRoleAssistant
		} else {
			role = types.ConversationRoleUser
		}

		msgs = append(msgs, types.Message{
			Role: role,
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{Value: h.Content},
			},
		})
	}

	// Append current user prompt if not already the last message
	if len(msgs) == 0 || msgs[len(msgs)-1].Role != types.ConversationRoleUser {
		msgs = append(msgs, types.Message{
			Role: types.ConversationRoleUser,
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{Value: currentPrompt},
			},
		})
	}

	return msgs
}
