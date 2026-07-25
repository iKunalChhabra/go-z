package z

// CheckFn implements a check's logic: inspect p.Value, append issues, or
// mutate p.Value (overwrite checks).
type CheckFn func(p *Payload)

// Check is a composable unit of validation attached
// to a schema. Checks run after the base type parse; failed checks append
// issues and, depending on abort semantics, stop later checks.
type Check struct {
	// Name identifies the check kind (e.g. "min_length", "greater_than").
	Name string
	// Fn performs the check.
	Fn CheckFn
	// Error is the check-level error map (first link of the chain for issues
	// this check produces). Attach it to issues via Check.Issue.
	Error ErrorMap
	// Abort stops subsequent checks when this check fails ({abort:true};
	// issues produced by non-aborting checks carry continue:true).
	Abort bool
	// When gates the check (`when`): if set, the check runs iff it
	// returns true — even when earlier issues would normally abort (only an
	// explicit abort suppresses it).
	When func(p *Payload) bool
	// OnAttach hooks run when the check is attached to a schema; used to
	// stash metadata in the schema bag (e.g. minLength for JSON Schema).
	OnAttach []func(in *Internals)
}

// Issue prepares a raw issue produced by this check: it wires the check's
// error map and continue semantics (continue:true unless Abort).
func (c *Check) Issue(iss Issue) Issue {
	iss.errMap = c.Error
	if c.Abort {
		iss.cont = continueUnset
	} else {
		iss.cont = continueYes
	}
	return iss
}

// runChecks executes a check chain over the payload, porting runChecks
// loop (core/schemas.ts): tracks abort state incrementally, honors `when`
// gates, and lets overwrite checks mutate the value.
func runChecks(p *Payload, checks []*Check) {
	isAborted := p.aborted(0)
	for _, ch := range checks {
		if ch.When != nil {
			if p.explicitlyAborted(0) {
				continue
			}
			if !ch.When(p) {
				continue
			}
		} else if isAborted {
			continue
		}
		currLen := len(p.Issues)
		ch.Fn(p)
		if len(p.Issues) == currLen {
			continue
		}
		if !isAborted {
			isAborted = p.aborted(currLen)
		}
	}
}
