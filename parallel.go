package zod

import (
	"context"
	"runtime"
	"sync"
)

// ParallelOpts configures ParseParallelSlice. Zero values are normalized to
// sensible defaults (Workers = GOMAXPROCS, MinChunk = 64).
type ParallelOpts struct {
	// Workers is the size of the goroutine pool. Default: runtime.GOMAXPROCS(0).
	// Values <= 1 force sequential validation.
	Workers int
	// MinChunk is the minimum slice length that opts into parallelism.
	// Below this threshold, validation runs sequentially. Default: 64.
	MinChunk int
}

// normalized applies defaults for zero-valued option fields.
func (opts ParallelOpts) normalized() ParallelOpts {
	if opts.Workers <= 0 {
		opts.Workers = runtime.GOMAXPROCS(0)
	}
	if opts.MinChunk <= 0 {
		opts.MinChunk = 64
	}
	return opts
}

// ParseParallelSlice validates each element of data with elemSchema, optionally
// in parallel chunks. Returns the validated values and a combined *ZodError
// whose issue paths are prefixed with the absolute element index.
//
// Issue order is deterministic: chunk order, then within-chunk index order.
// When len(data) < MinChunk or Workers <= 1, validation is sequential.
// Context cancellation is respected; on cancel, (nil, ctx.Err()) is returned.
func ParseParallelSlice(ctx context.Context, elemSchema AnySchemaLike, data []any, opts ParallelOpts) ([]any, error) {
	opts = opts.normalized()
	if data == nil {
		data = []any{}
	}
	if elemSchema == nil {
		out := make([]any, len(data))
		copy(out, data)
		return out, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	child := elemSchema.Internals()
	n := len(data)
	if n == 0 {
		return []any{}, nil
	}

	if n < opts.MinChunk || opts.Workers <= 1 {
		return parseSliceSequential(ctx, child, data)
	}
	return parseSliceParallel(ctx, child, data, opts)
}

func parseSliceSequential(ctx context.Context, child *Internals, data []any) ([]any, error) {
	out := make([]any, len(data))
	var issues []Issue
	for i, item := range data {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		p := AcquirePayload(item)
		child.Run(p, nil)
		out[i] = p.Value
		if len(p.Issues) > 0 {
			issues = appendIssuesWithIndex(issues, p.Issues, i)
		}
		ReleasePayload(p)
	}
	if len(issues) > 0 {
		return out, newZodError(issues, nil)
	}
	return out, nil
}

type parallelJob struct {
	id    int
	start int
	end   int
}

type parallelChunk struct {
	start  int
	values []any
	issues []Issue
	err    error
}

func parseSliceParallel(ctx context.Context, child *Internals, data []any, opts ParallelOpts) ([]any, error) {
	n := len(data)
	workers := opts.Workers
	if workers > n {
		workers = n
	}

	chunkSize := (n + workers - 1) / workers
	if chunkSize < 1 {
		chunkSize = 1
	}

	jobs := make([]parallelJob, 0, workers)
	for start := 0; start < n; start += chunkSize {
		end := start + chunkSize
		if end > n {
			end = n
		}
		jobs = append(jobs, parallelJob{id: len(jobs), start: start, end: end})
	}

	results := make([]parallelChunk, len(jobs))
	jobsCh := make(chan parallelJob, len(jobs))
	for _, j := range jobs {
		jobsCh <- j
	}
	close(jobsCh)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobsCh {
				results[j.id] = runParallelChunk(ctx, child, data, j)
			}
		}()
	}
	wg.Wait()

	out := make([]any, n)
	var issues []Issue
	for i := range results {
		r := &results[i]
		if r.err != nil {
			return nil, r.err
		}
		copy(out[r.start:], r.values)
		issues = append(issues, r.issues...)
	}
	if len(issues) > 0 {
		return out, newZodError(issues, nil)
	}
	return out, nil
}

func runParallelChunk(ctx context.Context, child *Internals, data []any, j parallelJob) parallelChunk {
	values := make([]any, j.end-j.start)
	var issues []Issue
	for i := j.start; i < j.end; i++ {
		if err := ctx.Err(); err != nil {
			return parallelChunk{start: j.start, values: values, issues: issues, err: err}
		}
		p := AcquirePayload(data[i])
		child.Run(p, nil)
		values[i-j.start] = p.Value
		if len(p.Issues) > 0 {
			issues = appendIssuesWithIndex(issues, p.Issues, i)
		}
		ReleasePayload(p)
	}
	return parallelChunk{start: j.start, values: values, issues: issues}
}

// appendIssuesWithIndex copies src issues, prefixing each path with index.
// Path slices are freshly allocated so pooled payloads can be released safely.
func appendIssuesWithIndex(dst []Issue, src []Issue, index int) []Issue {
	for i := range src {
		iss := src[i]
		path := make([]any, 0, len(iss.Path)+1)
		path = append(path, index)
		path = append(path, iss.Path...)
		iss.Path = path
		dst = append(dst, iss)
	}
	return dst
}
