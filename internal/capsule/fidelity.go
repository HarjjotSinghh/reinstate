package capsule

import "sort"

// Components is the set of fidelity component classifications included when
// computing Overall. Callers typically pass task-field components here; event
// classifications are derived and merged by AggregateFidelity.
type Components []Component

// AggregateFidelity derives the fidelity report from classified events and the
// included component set. Overall is the worst portability among all resulting
// components. Mode is always structured_handoff.
//
// There is no "lossless" value. Portability is exact, normalized, summarized,
// referenced, or omitted.
func AggregateFidelity(events []Event, included Components) Fidelity {
	type agg struct {
		port   Portability
		count  int
		bytes  int64
		reason string
	}
	byName := make(map[string]*agg, len(included)+len(events))

	merge := func(name string, p Portability, count int, bytes int64, reason string) {
		if name == "" || !ValidPortability(p) {
			return
		}
		cur, ok := byName[name]
		if !ok {
			byName[name] = &agg{port: p, count: count, bytes: bytes, reason: reason}
			return
		}
		if worsePortability(p, cur.port) {
			cur.port = p
			if reason != "" {
				cur.reason = reason
			}
		}
		cur.count += count
		cur.bytes += bytes
	}

	for _, c := range included {
		merge(c.Name, c.Portability, c.Count, c.Bytes, c.Reason)
	}
	for _, e := range events {
		merge(componentNameForEvent(e), e.Portability, 1, eventBytes(e), e.Reason)
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]Component, 0, len(names))
	var overall Portability
	have := false
	for _, name := range names {
		a := byName[name]
		out = append(out, Component{
			Name:        name,
			Portability: a.port,
			Count:       a.count,
			Bytes:       a.bytes,
			Reason:      a.reason,
		})
		if !have || worsePortability(a.port, overall) {
			overall = a.port
			have = true
		}
	}
	if !have {
		overall = PortabilityExact
	}
	return Fidelity{
		Overall:    overall,
		Mode:       FidelityModeStructuredHandoff,
		Components: out,
	}
}

// ValidPortability reports whether p is one of the five defined portability
// values. There is no "lossless" value.
func ValidPortability(p Portability) bool {
	switch p {
	case PortabilityExact, PortabilityNormalized, PortabilitySummarized,
		PortabilityReferenced, PortabilityOmitted:
		return true
	default:
		return false
	}
}

// WorstPortability returns the worse of a and b. Omitted is worst; exact is best.
func WorstPortability(a, b Portability) Portability {
	if worsePortability(a, b) {
		return a
	}
	return b
}

func worsePortability(a, b Portability) bool {
	return portabilityRank(a) > portabilityRank(b)
}

func portabilityRank(p Portability) int {
	switch p {
	case PortabilityExact:
		return 0
	case PortabilityNormalized:
		return 1
	case PortabilitySummarized:
		return 2
	case PortabilityReferenced:
		return 3
	case PortabilityOmitted:
		return 4
	default:
		return -1
	}
}

func componentNameForEvent(e Event) string {
	switch e.Kind {
	case KindMessage:
		switch e.Actor {
		case ActorUser:
			return "user_messages"
		case ActorAssistant:
			return "assistant_messages"
		case ActorHarness:
			return "harness_messages"
		default:
			return "messages"
		}
	case KindToolCall:
		return "tool_calls"
	case KindToolResult:
		return "tool_results"
	case KindAttachment:
		return "attachments"
	case KindSummary:
		return "summaries"
	case KindCheckpoint:
		return "checkpoints"
	case KindMetadata:
		return "metadata"
	case KindUnknown:
		return "unknown"
	default:
		if e.Kind == "" {
			return ""
		}
		return string(e.Kind)
	}
}

func eventBytes(e Event) int64 {
	var n int64
	for _, b := range e.Blocks {
		n += int64(len(b.Text))
	}
	return n
}

func worstOfComponents(components []Component) Portability {
	if len(components) == 0 {
		return PortabilityExact
	}
	worst := components[0].Portability
	for _, c := range components[1:] {
		if worsePortability(c.Portability, worst) {
			worst = c.Portability
		}
	}
	return worst
}
