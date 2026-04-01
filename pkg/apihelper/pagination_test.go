package apihelper

import (
	"context"
	"errors"
	"testing"
)

func TestPaginate_SinglePage(t *testing.T) {
	fetch := func(ctx context.Context, page int) ([]string, bool, error) {
		return []string{"a", "b"}, false, nil
	}
	var items []string
	for item := range Paginate(context.Background(), fetch) {
		items = append(items, item)
	}
	if len(items) != 2 {
		t.Errorf("items = %v, want [a b]", items)
	}
}

func TestPaginate_MultiplePages(t *testing.T) {
	fetch := func(ctx context.Context, page int) ([]int, bool, error) {
		switch page {
		case 1:
			return []int{1, 2}, true, nil
		case 2:
			return []int{3, 4}, true, nil
		case 3:
			return []int{5}, false, nil
		default:
			return nil, false, nil
		}
	}
	var items []int
	for item := range Paginate(context.Background(), fetch) {
		items = append(items, item)
	}
	if len(items) != 5 {
		t.Errorf("len = %d, want 5; items = %v", len(items), items)
	}
}

func TestPaginate_EmptyResult(t *testing.T) {
	fetch := func(ctx context.Context, page int) ([]string, bool, error) {
		return nil, false, nil
	}
	var count int
	for range Paginate(context.Background(), fetch) {
		count++
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestPaginate_ErrorStops(t *testing.T) {
	fetch := func(ctx context.Context, page int) ([]string, bool, error) {
		if page == 2 {
			return nil, false, errors.New("api error")
		}
		return []string{"item"}, true, nil
	}
	var count int
	for range Paginate(context.Background(), fetch) {
		count++
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestPaginate_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fetch := func(ctx context.Context, page int) ([]string, bool, error) {
		if page == 2 {
			cancel()
		}
		return []string{"item"}, true, nil
	}
	var count int
	for range Paginate(ctx, fetch) {
		count++
	}
	if count > 2 {
		t.Errorf("count = %d, should stop after cancellation", count)
	}
}
