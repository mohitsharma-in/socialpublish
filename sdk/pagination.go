package socialpublish

import "context"

// Pager is implemented by all paginated response types.
type Pager[T any] interface {
	GetItems() []T
	GetNextCursor() string
	GetHasMore() bool
}

// FetchFn fetches one page given a cursor.
type FetchFn[T any] func(ctx context.Context, cursor string) (Pager[T], error)

// Iter walks all pages lazily.
type Iter[T any] struct {
	ctx    context.Context
	fetch  FetchFn[T]
	cursor string
	items  []T
	pos    int
	done   bool
	err    error
}

// NewIter creates an Iter.
func NewIter[T any](ctx context.Context, fn FetchFn[T]) *Iter[T] {
	return &Iter[T]{ctx: ctx, fetch: fn}
}

// Next advances to the next item.
func (it *Iter[T]) Next() bool {
	if it.err != nil {
		return false
	}
	if it.pos < len(it.items) {
		it.pos++
		return true
	}
	if it.done {
		return false
	}
	page, err := it.fetch(it.ctx, it.cursor)
	if err != nil {
		it.err = err
		return false
	}
	it.items = page.GetItems()
	it.cursor = page.GetNextCursor()
	it.done = !page.GetHasMore()
	it.pos = 0
	if len(it.items) == 0 {
		return false
	}
	it.pos = 1
	return true
}

// Item returns the current item.
func (it *Iter[T]) Item() T { return it.items[it.pos-1] }

// Err returns the first error encountered during iteration.
func (it *Iter[T]) Err() error { return it.err }
