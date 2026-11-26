package autotask

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const contractServiceTool = "search_contract_services"

func registerContractServiceTool(srv *server.MCPServer, c *Client) {
	// Keep the opt-in read on this instance, including meta-dispatch.
	srv.AddTool(mcp.NewTool("search_contract_services",
		mcp.WithDescription("Read recurring contract service lines and their quantity and period price on a UTC date. Scope by contractID, contractIDs or exact line id."),
		mcp.WithNumber("contractID", mcp.Min(1)),
		mcp.WithArray("contractIDs", mcp.WithNumberItems(mcp.Min(1)), mcp.MinItems(1), mcp.MaxItems(500)),
		mcp.WithNumber("serviceID", mcp.Min(1)),
		mcp.WithArray("serviceIDs", mcp.WithNumberItems(mcp.Min(1)), mcp.MinItems(1), mcp.MaxItems(500)),
		mcp.WithNumber("id", mcp.Description("Exact ContractServices line ID."), mcp.Min(1)),
		mcp.WithString("asOf", mcp.Description("UTC calendar date YYYY-MM-DD. Defaults to today. Retain it when paging.")),
		mcp.WithNumber("page", mcp.Min(1)),
		mcp.WithNumber("pageSize", mcp.Min(1), mcp.Max(500)),
		mcp.WithString("nextPageUrl", mcp.Description("Continuation from the preceding page, with the same filters, date and page size.")),
		mcp.WithReadOnlyHintAnnotation(true),
	), c.searchContractServices)
}

func contractFilters(args map[string]any) ([]Filter, error) {
	maxID := int(math.Min(float64(math.MaxInt), (1<<53)-1))
	var filters []Filter
	scoped := false
	for _, field := range []string{"contractID", "id", "serviceID"} {
		_, single := args[field]
		multiple, plural := args[field+"s"]
		if plural && (single || field == "id") {
			return nil, fmt.Errorf("invalid combination of %s filters", field)
		}
		if single {
			id, err := mcputil.IntArgument(args, field, 0, 1, maxID)
			if err != nil {
				return nil, err
			}
			filters = append(filters, Filter{Op: "eq", Field: field, Value: id})
		}
		if plural {
			values, ok := multiple.([]any)
			if !ok || len(values) == 0 || len(values) > 500 {
				return nil, fmt.Errorf("%ss must contain 1 to 500 IDs", field)
			}
			for _, v := range values {
				n, ok := v.(float64)
				if !ok || math.Trunc(n) != n || !(n >= 1 && n <= float64(maxID)) {
					return nil, fmt.Errorf("invalid %s in list", field)
				}
			}
			filters = append(filters, Filter{Op: "in", Field: field, Value: values})
		}
		if field != "serviceID" && (single || plural) {
			scoped = true
		}
	}
	if !scoped {
		return nil, fmt.Errorf("contractID, contractIDs or an exact line id is required")
	}
	return filters, nil
}

func (c *Client) searchContractServices(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	filters, err := contractFilters(args)
	if err != nil {
		return mcputil.ErrorResult(err), nil
	}
	date := time.Now().UTC().Format(time.DateOnly)
	if _, ok := args["asOf"]; ok {
		date, _ = args["asOf"].(string)
	}
	day, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return mcputil.ErrorResult(fmt.Errorf("asOf must be a valid YYYY-MM-DD date")), nil
	}
	page, err := mcputil.IntArgument(args, "page", 1, 1, math.MaxInt32)
	if err != nil {
		return mcputil.ErrorResult(err), nil
	}
	size, err := mcputil.IntArgument(args, "pageSize", 25, 1, 500)
	if err != nil {
		return mcputil.ErrorResult(err), nil
	}
	var cursor string
	if _, ok := args["nextPageUrl"]; ok {
		cursor, err = req.RequireString("nextPageUrl")
		if err != nil || cursor == "" {
			return mcputil.ErrorResult(fmt.Errorf("nextPageUrl must be a nonempty string")), nil
		}
	}
	lines, err := c.numberedPage(ctx, "ContractServices", filters, QueryOpts{PageSize: size}, page, cursor)
	if err != nil {
		return mcputil.ErrorResult(err), nil
	}
	items, err := c.contractServicePeriods(ctx, lines.Items, day, filters)
	if err != nil {
		return mcputil.ErrorResult(err), nil
	}
	return mcputil.JSONResult(map[string]any{
		"asOf": date, "items": items,
		"summary": map[string]any{"page": page, "pageSize": size, "returned": len(items), "hasMore": lines.NextPageURL != nil, "nextPageUrl": lines.NextPageURL},
	}), nil
}

// ParseContractTime accepts date-only and timestamp fields. Autotask timestamps
// without an offset are UTC.
func ParseContractTime(raw any) (time.Time, error) {
	value, _ := raw.(string)
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", time.DateOnly} {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date in contract service period")
}

func validLineID(raw any) (float64, bool) {
	n, ok := raw.(float64)
	return n, ok && n >= 1 && n <= (1<<53)-1 && math.Trunc(n) == n
}

func matchesContractFilters(line map[string]any, filters []Filter) bool {
	for _, f := range filters {
		value, ok := validLineID(line[f.Field])
		if !ok {
			return false
		}
		switch ids := f.Value.(type) {
		case int:
			if value != float64(ids) {
				return false
			}
		case []any:
			if !slices.Contains(ids, any(value)) {
				return false
			}
		}
	}
	return true
}

func (c *Client) contractServicePeriods(ctx context.Context, lines []map[string]any, day time.Time, filters []Filter) ([]map[string]any, error) {
	output := make([]map[string]any, 0, len(lines))
	byID := make(map[float64]map[string]any, len(lines))
	ids := make([]float64, 0, len(lines))
	for _, line := range lines {
		// An opaque continuation belongs to the original filters. Reject a
		// mismatched page before reading its units or returning financial data.
		if !matchesContractFilters(line, filters) {
			return nil, fmt.Errorf("contract service page does not match the requested filters")
		}
		id, lineOK := validLineID(line["id"])
		contractID, contractOK := validLineID(line["contractID"])
		serviceID, serviceOK := validLineID(line["serviceID"])
		if !lineOK || !contractOK || !serviceOK || byID[id] != nil {
			return nil, fmt.Errorf("invalid contract or service ID, or duplicate line")
		}
		item := map[string]any{
			"id": id, "contractID": contractID, "serviceID": serviceID,
			"unitPrice": line["unitPrice"], "internalCurrencyUnitPrice": line["internalCurrencyUnitPrice"],
			"quantity": nil, "period": nil,
		}
		ids = append(ids, id)
		byID[id] = item
		output = append(output, item)
	}
	if len(ids) == 0 {
		return output, nil
	}
	tomorrow := day.AddDate(0, 0, 1)
	periodFilters := []Filter{
		{Op: "in", Field: "contractServiceID", Value: ids},
		{Op: "lt", Field: "startDate", Value: tomorrow.Format(time.RFC3339)},
		{Op: "gte", Field: "endDate", Value: day.Format(time.RFC3339)},
	}
	starts := make(map[float64]time.Time)
	ambiguous := make(map[float64]bool)
	seen := make(map[string]bool)
	cursor := ""
	for {
		if seen[cursor] {
			return nil, fmt.Errorf("repeated contract period continuation")
		}
		seen[cursor] = true
		periods, err := c.QueryPage(ctx, "ContractServiceUnits", periodFilters, QueryOpts{PageSize: 500}, cursor)
		if err != nil {
			return nil, err
		}
		for _, period := range periods.Items {
			id, _ := period["contractServiceID"].(float64)
			item := byID[id]
			if item == nil || period["contractID"] != item["contractID"] || period["serviceID"] != item["serviceID"] {
				return nil, fmt.Errorf("period does not match a requested contract service")
			}
			start, err := ParseContractTime(period["startDate"])
			if err != nil {
				return nil, err
			}
			end, err := ParseContractTime(period["endDate"])
			if err != nil {
				return nil, err
			}
			if end.Before(start) {
				return nil, fmt.Errorf("contract service period ends before it starts")
			}
			if !start.Before(tomorrow) || end.Before(day) || start.Before(starts[id]) {
				continue
			}
			if start.Equal(starts[id]) {
				ambiguous[id] = true
				continue
			}
			delete(ambiguous, id)
			quantity, ok := period["units"].(float64)
			if !ok || math.Trunc(quantity) != quantity {
				return nil, fmt.Errorf("invalid quantity for contract service %.0f", id)
			}
			starts[id] = start
			item["quantity"] = quantity
			item["period"] = map[string]any{
				"id": period["id"], "startDate": period["startDate"], "endDate": period["endDate"],
				"price": period["price"], "internalCurrencyPrice": period["internalCurrencyPrice"],
			}
		}
		if periods.NextPageURL == nil {
			if len(ambiguous) > 0 {
				return nil, fmt.Errorf("multiple contract service periods share the latest effective date")
			}
			return output, nil
		}
		cursor = *periods.NextPageURL
	}
}
