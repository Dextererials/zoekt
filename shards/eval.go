package shards

import (
	"context"

	"github.com/xvandish/zoekt"
	"github.com/xvandish/zoekt/query"
	"github.com/xvandish/zoekt/trace"
)

// typeRepoSearcher evaluates all type:repo sub-queries before sending the query
// to the underlying searcher. We need to evaluate type:repo queries first
// since they need to do cross shard operations.
type typeRepoSearcher struct {
	zoekt.Streamer
}

func (s *typeRepoSearcher) Search(ctx context.Context, q query.Q, opts *zoekt.SearchOptions) (sr *zoekt.SearchResult, err error) {
	tr, ctx := trace.New(ctx, "typeRepoSearcher.Search", "")
	tr.LazyLog(q, true)
	tr.LazyPrintf("opts: %+v", opts)
	defer func() {
		if sr != nil {
			tr.LazyPrintf("num files: %d", len(sr.Files))
			tr.LazyPrintf("stats: %+v", sr.Stats)
		}
		if err != nil {
			tr.LazyPrintf("error: %v", err)
			tr.SetError(err)
		}
		tr.Finish()
	}()

	q, err = s.eval(ctx, q)
	if err != nil {
		return nil, err
	}

	return s.Streamer.Search(ctx, q, opts)
}

func (s *typeRepoSearcher) StreamSearch(ctx context.Context, q query.Q, opts *zoekt.SearchOptions, sender zoekt.Sender) (err error) {
	tr, ctx := trace.New(ctx, "typeRepoSearcher.StreamSearch", "")
	tr.LazyLog(q, true)
	tr.LazyPrintf("opts: %+v", opts)
	defer func() {
		if err != nil {
			tr.LazyPrintf("error: %v", err)
			tr.SetError(err)
		}
		tr.Finish()
	}()

	q, err = s.eval(ctx, q)
	if err != nil {
		return err
	}

	return s.Streamer.StreamSearch(ctx, q, opts, sender)
}

func (s *typeRepoSearcher) List(ctx context.Context, r query.Q, opts *zoekt.ListOptions) (rl *zoekt.RepoList, err error) {
	tr, ctx := trace.New(ctx, "typeRepoSearcher.List", "")
	tr.LazyLog(r, true)
	tr.LazyPrintf("opts: %s", opts)
	defer func() {
		if rl != nil {
			tr.LazyPrintf("repos size: %d", len(rl.Repos))
			tr.LazyPrintf("crashes: %d", rl.Crashes)
			tr.LazyPrintf("minimal size : %d", len(rl.Minimal))
		}
		if err != nil {
			tr.LazyPrintf("error: %v", err)
			tr.SetError(err)
		}
		tr.Finish()
	}()

	r, err = s.eval(ctx, r)
	if err != nil {
		return nil, err
	}

	return s.Streamer.List(ctx, r, opts)
}

func (s *typeRepoSearcher) eval(ctx context.Context, q query.Q) (query.Q, error) {
	var err error
	q = query.Map(q, func(q query.Q) query.Q {
		if err != nil {
			return nil
		}

		rq, ok := q.(*query.Type)
		if !ok || rq.Type != query.TypeRepo {
			return q
		}

		var rl *zoekt.RepoList
		rl, err = s.Streamer.List(ctx, rq.Child, nil)
		if err != nil {
			return nil
		}

		rs := &query.RepoSet{Set: make(map[string]bool, len(rl.Repos))}
		for _, r := range rl.Repos {
			rs.Set[r.Repository.Name] = true
		}
		return rs
	})
	return q, err
}
