package z

import (
	"context"
	"sync"
)

// Payload mirrors ParsePayload: the value being parsed plus accumulated
// issues. It is threaded through the whole pipeline; parsing never throws.
type Payload struct {
	Value  any
	Issues []Issue
	// Aborted marks the whole payload as aborted (uses this in pipes).
	Aborted bool

	// parseCtx is set for the duration of a top-level/child Run so checks and
	// SuperRefine can read Context / error maps without changing CheckFn.
	parseCtx *ParseCtx
}

// AddIssue appends a raw issue.
func (p *Payload) AddIssue(iss Issue) {
	p.Issues = append(p.Issues, iss)
}

// aborted reports whether validation is aborted considering issues from
// startIndex on. Mirrors util.aborted: any issue whose continue flag is
// not explicitly true aborts.
func (p *Payload) aborted(startIndex int) bool {
	if p.Aborted {
		return true
	}
	for i := startIndex; i < len(p.Issues); i++ {
		if p.Issues[i].cont != continueYes {
			return true
		}
	}
	return false
}

// explicitlyAborted mirrors util.explicitlyAborted: only issues with
// continue == false abort `when`-gated checks.
func (p *Payload) explicitlyAborted(startIndex int) bool {
	if p.Aborted {
		return true
	}
	for i := startIndex; i < len(p.Issues); i++ {
		if p.Issues[i].cont == continueNo {
			return true
		}
	}
	return false
}

// PrependPath prefixes seg to the path of every issue added at or after
// startIndex. Containers (object/array/record/...) use this to scope child
// issues, exactly like `path.unshift(key)`.
func (p *Payload) PrependPath(startIndex int, seg any) {
	for i := startIndex; i < len(p.Issues); i++ {
		iss := &p.Issues[i]
		path := make([]any, 0, len(iss.Path)+1)
		path = append(path, seg)
		path = append(path, iss.Path...)
		iss.Path = path
	}
}

// ParseCtx carries per-parse options (ParseContext).
type ParseCtx struct {
	// Context is the request/operation context (cancellations, deadlines,
	// values). SuperRefine reads it via RefinementCtx.Context().
	Context context.Context
	// Error is a per-parse error map (second link in the resolution chain).
	Error ErrorMap
	// ReportInput includes the offending input in finalized issues.
	ReportInput bool
	// Direction selects decode (forward) or encode (backward) parsing.
	Direction ParseDirection

	// skipChecks is internal (uses it for backward/codec passes).
	skipChecks bool
}

var payloadPool = sync.Pool{
	New: func() any {
		return &Payload{Issues: make([]Issue, 0, 4)}
	},
}

// AcquirePayload fetches a pooled payload initialized with value.
func AcquirePayload(value any) *Payload {
	p := payloadPool.Get().(*Payload)
	p.Value = value
	p.Issues = p.Issues[:0]
	p.Aborted = false
	p.parseCtx = nil
	return p
}

// ReleasePayload returns a payload to the pool. Callers must not retain
// references to p or its issue slice afterwards; on failure paths, finalized
// issues are copied out before release.
func ReleasePayload(p *Payload) {
	if cap(p.Issues) > 64 {
		p.Issues = make([]Issue, 0, 4) // don't pin huge error slices
	}
	p.Value = nil
	p.Aborted = false
	p.parseCtx = nil
	payloadPool.Put(p)
}
