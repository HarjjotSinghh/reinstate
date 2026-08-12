package capsule

// SidecarRef identifies a conversation event preserved outside the projection
// body. Excluded history is always referenced this way — never silently dropped.
type SidecarRef struct {
	EventID     string      `json:"event_id"`
	ContentHash string      `json:"content_hash,omitempty"`
	Bytes       int64       `json:"bytes,omitempty"`
	Portability Portability `json:"portability"`
	Reason      string      `json:"reason,omitempty"`
}
