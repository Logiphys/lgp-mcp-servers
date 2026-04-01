package apihelper

import (
	"context"
	"iter"
)

type PageFetcher[T any] func(ctx context.Context, page int) (items []T, hasMore bool, err error)

func Paginate[T any](ctx context.Context, fetch PageFetcher[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for page := 1; ; page++ {
			if ctx.Err() != nil {
				return
			}
			items, hasMore, err := fetch(ctx, page)
			if err != nil {
				return
			}
			for _, item := range items {
				if !yield(item) {
					return
				}
			}
			if !hasMore || len(items) == 0 {
				return
			}
		}
	}
}
