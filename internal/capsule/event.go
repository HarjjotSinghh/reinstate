package capsule

import "time"

// Actor is the normalized speaker of a canonical event.
type Actor string

const (
	ActorUser      Actor = "user"
	ActorAssistant Actor = "assistant"
	ActorTool      Actor = "tool"
	ActorHarness   Actor = "harness"
	ActorUnknown   Actor = "unknown"
)

// Kind is the normalized event category.
type Kind string

const (
	KindMessage    Kind = "message"
	KindToolCall   Kind = "tool_call"
	KindToolResult Kind = "tool_result"
	KindAttachment Kind = "attachment"
	KindSummary    Kind = "summary"
	KindCheckpoint Kind = "checkpoint"
	KindMetadata   Kind = "metadata"
	KindUnknown    Kind = "unknown"
)

// Portability classifies how faithfully a component or event transfers.
// There is no "lossless" value.
type Portability string

const (
	PortabilityExact      Portability = "exact"
	PortabilityNormalized Portability = "normalized"
	PortabilitySummarized Portability = "summarized"
	PortabilityReferenced Portability = "referenced"
	PortabilityOmitted    Portability = "omitted"
)

// BlockType discriminates typed content blocks inside an event.
type BlockType string

const (
	BlockTypeText       BlockType = "text"
	BlockTypeJSON       BlockType = "json"
	BlockTypeToolInput  BlockType = "tool_input"
	BlockTypeToolOutput BlockType = "tool_output"
	BlockTypeAttachment BlockType = "attachment"
	BlockTypeRef        BlockType = "ref"
)

// Category names a redaction class. Values align with secretscan categories
// but live here so the capsule model does not import that package.
type Category string

const (
	CategoryAWSKey        Category = "aws_key"
	CategoryGCPKey        Category = "gcp_key"
	CategoryGitHubToken   Category = "github_token"
	CategoryOpenAIKey     Category = "openai_key"
	CategoryAnthropicKey  Category = "anthropic_key"
	CategoryPrivateKey    Category = "private_key"
	CategoryJWT           Category = "jwt"
	CategoryBearer        Category = "bearer"
	CategoryURLCredential Category = "url_credential"
	CategoryHighEntropy   Category = "high_entropy"
)

// SourcePointer locates the originating vendor record for a canonical event.
type SourcePointer struct {
	Agent      string `json:"agent"`
	SessionID  string `json:"session_id"`
	RecordKey  string `json:"record_key,omitempty"`
	ByteOffset int64  `json:"byte_offset"`
	Index      int    `json:"index"`
}

// Block is a typed content unit inside an Event.
//
// Path, when set, must be a portable pathmap token (${REPO:…} / ${HOME}…),
// never an absolute filesystem path.
type Block struct {
	Type    BlockType         `json:"type"`
	Text    string            `json:"text,omitempty"`
	MIME    string            `json:"mime,omitempty"`
	SHA256  string            `json:"sha256,omitempty"`
	Size    int64             `json:"size,omitempty"`
	Ref     string            `json:"ref,omitempty"`
	Path    string            `json:"path,omitempty"`
	IsError bool              `json:"is_error,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// Redaction records that a secret-shaped span was replaced. It never stores the
// matched value — only category and digest prefix.
type Redaction struct {
	Category Category `json:"category"`
	Digest   string   `json:"digest"`
}

// Event is one canonical conversation unit inside a continuity capsule.
type Event struct {
	ID           string        `json:"id"`
	Order        int           `json:"order"`
	Timestamp    time.Time     `json:"timestamp,omitzero"`
	Actor        Actor         `json:"actor"`
	Kind         Kind          `json:"kind"`
	NativeType   string        `json:"native_type,omitempty"`
	NativeName   string        `json:"native_name,omitempty"`
	Blocks       []Block       `json:"blocks,omitempty"`
	CallID       string        `json:"call_id,omitempty"`
	LinkedCallID string        `json:"linked_call_id,omitempty"`
	Portability  Portability   `json:"portability"`
	Reason       string        `json:"reason,omitempty"`
	Redactions   []Redaction   `json:"redactions,omitempty"`
	Truncated    bool          `json:"truncated,omitempty"`
	ContentHash  string        `json:"content_hash"`
	Source       SourcePointer `json:"source"`
}
