package autotaskcontracts

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Logiphys/lgp-mcp-servers/pkg/autotask"
)

func queryAll(ctx context.Context, c *autotask.Client, entity string, filters []autotask.Filter) ([]map[string]any, error) {
	var rows []map[string]any
	seen := make(map[string]bool)
	cursor := ""
	for {
		page, err := c.QueryPage(ctx, entity, filters, autotask.QueryOpts{PageSize: 500}, cursor)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entity, err)
		}
		rows = append(rows, page.Items...)
		if page.NextPageURL == nil {
			return rows, nil
		}
		cursor = *page.NextPageURL
		if seen[cursor] {
			return nil, fmt.Errorf("%s: repeated query continuation", entity)
		}
		seen[cursor] = true
	}
}

type catalog struct {
	serviceNames, bundleNames map[int]string
	members                   map[int][]int
}

func readCatalog(ctx context.Context, c *autotask.Client, serviceIDs, bundleIDs map[int]bool) (catalog, error) {
	cat := catalog{members: map[int][]int{}}
	rows, err := catalogRows(ctx, c, "ServiceBundleServices", "serviceBundleID", bundleIDs)
	if err != nil {
		return cat, err
	}
	for _, row := range rows {
		bundleID, err := requiredInt(row, "serviceBundleID", 1, maxID)
		if err != nil || !bundleIDs[bundleID] {
			return cat, fmt.Errorf("ServiceBundleServices: unexpected bundle ID in catalog response")
		}
		serviceID, err := requiredInt(row, "serviceID", 1, maxID)
		if err != nil {
			return cat, fmt.Errorf("ServiceBundleServices: %w", err)
		}
		cat.members[bundleID] = append(cat.members[bundleID], serviceID)
		serviceIDs[serviceID] = true
	}
	for id, members := range cat.members {
		slices.Sort(members)
		cat.members[id] = slices.Compact(members)
	}
	cat.bundleNames, err = readNames(ctx, c, "ServiceBundles", bundleIDs)
	if err != nil {
		return cat, err
	}
	cat.serviceNames, err = readNames(ctx, c, "Services", serviceIDs)
	return cat, err
}

func readNames(ctx context.Context, c *autotask.Client, entity string, ids map[int]bool) (map[int]string, error) {
	rows, err := catalogRows(ctx, c, entity, "id", ids)
	if err != nil {
		return nil, err
	}
	names := make(map[int]string, len(ids))
	for _, row := range rows {
		id, err := requiredInt(row, "id", 1, maxID)
		if err != nil || !ids[id] {
			return nil, fmt.Errorf("%s: unexpected ID in catalog response", entity)
		}
		name, _ := row["name"].(string)
		if strings.TrimSpace(name) == "" || (names[id] != "" && names[id] != name) {
			return nil, fmt.Errorf("%s: missing or conflicting name for ID %d", entity, id)
		}
		names[id] = name
	}
	if len(names) != len(ids) {
		return nil, fmt.Errorf("%s: missing referenced catalog record", entity)
	}
	return names, nil
}

func catalogRows(ctx context.Context, c *autotask.Client, entity, field string, ids map[int]bool) ([]map[string]any, error) {
	var all []map[string]any
	for batch := range slices.Chunk(slices.Sorted(maps.Keys(ids)), 500) {
		rows, err := queryAll(ctx, c, entity, []autotask.Filter{{Op: "in", Field: field, Value: batch}})
		if err != nil {
			return nil, err
		}
		all = append(all, rows...)
	}
	return all, nil
}
