package aiagent

import (
	"time"
)

// Role and permission context
const (
	RoleOwner          = "OWNER"
	RoleKepalaProduksi = "KEPALA_PRODUKSI"
	RoleTimSample      = "TIM_SAMPLE"
	RoleGudang         = "GUDANG"
	RoleMandor         = "MANDOR"
	RolePenjahit       = "PENJAHIT"
	RoleOperator       = "OPERATOR"
	RoleQCInspector    = "QC_INSPECTOR"
	RoleFinancePayroll = "FINANCE_PAYROLL"
	RoleClientPortal   = "CLIENT_PORTAL"
)

// ChatMessage represents a single message in the conversation history.
type ChatMessage struct {
	ID        string     `json:"id"`
	Role      string     `json:"role"` // "user", "assistant", "system"
	Content   string     `json:"content"`
	Timestamp int64      `json:"timestamp,omitempty"`
	Citations []Citation `json:"citations,omitempty"`
}

// Citation represents an excerpt retrieved from the Bedrock Knowledge Base.
type Citation struct {
	ID    string   `json:"id"`
	Index int      `json:"index"`
	Title string   `json:"title"`
	URI   string   `json:"uri,omitempty"`
	Text  string   `json:"text"`
	Score *float64 `json:"score,omitempty"`
}

// AgentRequest represents the inbound payload from the client.
type AgentRequest struct {
	Query       string        `json:"query"`
	Question    string        `json:"question,omitempty"`
	Prompt      string        `json:"prompt,omitempty"`
	Messages    []ChatMessage `json:"messages,omitempty"`
	ContextRole string        `json:"contextRole,omitempty"`
	ClientID    *string       `json:"clientId,omitempty"`
	Stream      *bool         `json:"stream,omitempty"`
	ModelID     string        `json:"modelId,omitempty"`
	Temperature float32       `json:"temperature,omitempty"`
	MaxTokens   int32         `json:"maxTokens,omitempty"`
}

// GetUserPrompt extracts the latest user question from the request.
func (r *AgentRequest) GetUserPrompt() string {
	if r.Query != "" {
		return r.Query
	}
	if r.Question != "" {
		return r.Question
	}
	if r.Prompt != "" {
		return r.Prompt
	}
	if len(r.Messages) > 0 {
		for i := len(r.Messages) - 1; i >= 0; i-- {
			if r.Messages[i].Role == "user" && r.Messages[i].Content != "" {
				return r.Messages[i].Content
			}
		}
	}
	return ""
}

// SubTask represents a decomposed action item.
type SubTask struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"` // "DATABASE_TOOL" | "RAG_LOOKUP"
	ToolName   string         `json:"toolName,omitempty"`
	Parameters map[string]any `json:"parameters,omitempty"`
	RagQuery   string         `json:"ragQuery,omitempty"`
	Purpose    string         `json:"purpose"`
}

// DecomposedPlan represents the planned sub-tasks for answering the prompt.
type DecomposedPlan struct {
	OriginalQuery string    `json:"originalQuery"`
	IsComplex     bool      `json:"isComplex"`
	Reasoning     string    `json:"reasoning"`
	SubTasks      []SubTask `json:"subTasks"`
}

// ToolResult represents the outcome of an internal database tool execution.
type ToolResult struct {
	Tool    string `json:"tool"`
	Success bool   `json:"success"`
	Count   int    `json:"count"`
	Records any    `json:"records"`
	Message string `json:"message,omitempty"`
}

// AgentStreamEvent defines an NDJSON event sent to the client stream.
type AgentStreamEvent struct {
	Type string `json:"type"` // "plan", "tool_call", "tool_result", "citations", "token", "done", "error"
	Data any    `json:"data"`
}

// AgentFinalResult is emitted on "done" event.
type AgentFinalResult struct {
	FullText          string       `json:"fullText"`
	ModelID           string       `json:"modelId"`
	Plan              *DecomposedPlan `json:"plan,omitempty"`
	Citations         []Citation   `json:"citations,omitempty"`
	ExecutedTools     []ToolResult `json:"executedTools,omitempty"`
	TotalRecordsFound int          `json:"totalRecordsFound"`
	ExecutionTimeMs   int64        `json:"executionTimeMs"`
}

// HealthStatus returns the runtime status of the AI agent module.
type HealthStatus struct {
	Status           string   `json:"status"`
	Engine           string   `json:"engine"`
	Region           string   `json:"region"`
	ModelID          string   `json:"modelId"`
	KnowledgeBaseID  string   `json:"knowledgeBaseId"`
	GuardrailActive  bool     `json:"guardrailActive"`
	AvailableTools   []string `json:"availableTools"`
	DatabaseEngine   string   `json:"databaseEngine"`
	Timestamp        time.Time `json:"timestamp"`
}
