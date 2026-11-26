package autotaskcontracts

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/Logiphys/lgp-mcp-servers/pkg/autotask"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools exposes the monthly read on its own gateway server grant.
func RegisterTools(srv *server.MCPServer, client *autotask.Client) {
	srv.AddTool(mcp.NewTool("get_service_month",
		mcp.WithDescription("Read a contract's service and bundle quantities for a UTC calendar month. Bundle membership comes from the current catalog. Historical service totals can be unknown."),
		mcp.WithNumber("contractID", mcp.Required(), mcp.Min(1)),
		mcp.WithString("month", mcp.Required(), mcp.Description("UTC calendar month, YYYY-MM")),
		mcp.WithSchemaAdditionalProperties(false),
		mcp.WithReadOnlyHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		id, err := requiredInt(args, "contractID", 1, maxID)
		if err != nil {
			return mcputil.ErrorResult(err), nil
		}
		month, err := time.Parse("2006-01", req.GetString("month", ""))
		if err != nil || month.Year() < 1 || len(args) != 2 {
			return mcputil.ErrorResult(fmt.Errorf("provide only contractID and a valid YYYY-MM month")), nil
		}
		result, err := readMonth(ctx, client, id, month)
		if err != nil {
			return mcputil.ErrorResult(err), nil
		}
		return mcputil.JSONResult(result), nil
	})
}

type monthReport struct {
	ContractID    int             `json:"contractID"`
	Month         string          `json:"month"`
	Periods       []monthInterval `json:"periods"`
	CatalogReadAt string          `json:"catalogReadAt,omitempty"`
	CatalogNote   string          `json:"catalogNote,omitempty"`
}

type monthInterval struct {
	StartDate               string            `json:"startDate"`
	EndDate                 string            `json:"endDate"`
	ServiceTotalsComplete   bool              `json:"serviceTotalsComplete"`
	Services                []serviceTotal    `json:"services"`
	Bundles                 []bundleQuantity  `json:"bundles"`
	CurrentCatalogEstimates []catalogEstimate `json:"currentCatalogEstimates"`
}

type directLine struct {
	ContractServiceID int `json:"contractServiceID"`
	Units             int `json:"units"`
}

type serviceTotal struct {
	ServiceID   int          `json:"serviceID"`
	Name        string       `json:"name"`
	DirectLines []directLine `json:"directLines"`
	DirectUnits int64        `json:"directUnits"`
	TotalUnits  *int64       `json:"totalUnits"`
}

type bundleQuantity struct {
	ContractServiceBundleID int    `json:"contractServiceBundleID"`
	ServiceBundleID         int    `json:"serviceBundleID"`
	Name                    string `json:"name"`
	Units                   int    `json:"units"`
}

type catalogEstimate struct {
	ContractServiceBundleID int    `json:"contractServiceBundleID"`
	ServiceID               int    `json:"serviceID"`
	Name                    string `json:"name"`
	DerivedUnits            int    `json:"derivedUnits"`
}

func readMonth(ctx context.Context, c *autotask.Client, id int, month time.Time) (monthReport, error) {
	out := monthReport{ContractID: id, Month: month.Format("2006-01"), Periods: []monthInterval{}}
	contract, err := c.Get(ctx, "Contracts", id)
	if err != nil {
		return out, fmt.Errorf("reading Contracts: %w", err)
	}
	returnedID, err := requiredInt(contract, "id", 1, maxID)
	if err != nil || returnedID != id {
		return out, fmt.Errorf("reading Contracts: requested contract was not returned")
	}
	direct, err := readUnitDays(ctx, c, id, month, "ContractServiceUnits", "contractServiceID", "serviceID")
	if err != nil {
		return out, err
	}
	bundles, err := readUnitDays(ctx, c, id, month, "ContractServiceBundleUnits", "contractServiceBundleID", "serviceBundleID")
	if err != nil {
		return out, err
	}
	catalog, err := readCatalog(ctx, c, direct.itemIDs(), bundles.itemIDs())
	if err != nil {
		return out, err
	}
	if len(catalog.bundleNames) > 0 {
		out.CatalogReadAt = time.Now().UTC().Format(time.RFC3339)
		out.CatalogNote = "Estimates use membership at catalogReadAt. Membership in the reported period is unverified. Service totals are unknown for periods with a nonzero bundle quantity."
	}
	for day := 0; day < len(direct); {
		next := day + 1
		for next < len(direct) && slices.Equal(direct[day], direct[next]) && slices.Equal(bundles[day], bundles[next]) {
			next++
		}
		if len(direct[day])+len(bundles[day]) > 0 {
			row := catalog.interval(direct[day], bundles[day])
			row.StartDate = month.AddDate(0, 0, day).Format(time.DateOnly)
			row.EndDate = month.AddDate(0, 0, next-1).Format(time.DateOnly)
			out.Periods = append(out.Periods, row)
		}
		day = next
	}
	return out, nil
}

func (cat catalog) interval(direct, bundles []booking) monthInterval {
	row := monthInterval{
		ServiceTotalsComplete:   true,
		Services:                []serviceTotal{},
		Bundles:                 []bundleQuantity{},
		CurrentCatalogEstimates: []catalogEstimate{},
	}
	for _, b := range bundles {
		row.Bundles = append(row.Bundles, bundleQuantity{b.lineID, b.itemID, cat.bundleNames[b.itemID], b.units})
		if b.units != 0 {
			row.ServiceTotalsComplete = false
		}
		for _, serviceID := range cat.members[b.itemID] {
			row.CurrentCatalogEstimates = append(row.CurrentCatalogEstimates, catalogEstimate{b.lineID, serviceID, cat.serviceNames[serviceID], b.units})
		}
	}
	byID := make(map[int]*serviceTotal)
	for _, b := range direct {
		service := byID[b.itemID]
		if service == nil {
			service = &serviceTotal{ServiceID: b.itemID, Name: cat.serviceNames[b.itemID]}
			byID[b.itemID] = service
		}
		service.DirectLines = append(service.DirectLines, directLine{b.lineID, b.units})
		service.DirectUnits += int64(b.units)
	}
	for _, id := range slices.Sorted(maps.Keys(byID)) {
		service := byID[id]
		if row.ServiceTotalsComplete {
			service.TotalUnits = &service.DirectUnits
		}
		row.Services = append(row.Services, *service)
	}
	return row
}
