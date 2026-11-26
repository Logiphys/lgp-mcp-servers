package autotaskcontracts

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/Logiphys/lgp-mcp-servers/pkg/autotask"
	"github.com/Logiphys/lgp-mcp-servers/pkg/mcputil"
)

const maxID = int(min(int64(math.MaxInt), 1<<53-1))

func requiredInt(row map[string]any, field string, minimum, maximum int) (int, error) {
	if _, ok := row[field]; !ok {
		return 0, fmt.Errorf("missing %s", field)
	}
	return mcputil.IntArgument(row, field, 0, minimum, maximum)
}

type booking struct {
	lineID, itemID, units int
}

type unitPeriod struct {
	booking
	start, end time.Time
}

// unitDays contains at most 31 days, each ordered by contract position.
type unitDays [][]booking

func (days unitDays) itemIDs() map[int]bool {
	ids := map[int]bool{}
	for _, day := range days {
		for _, b := range day {
			ids[b.itemID] = true
		}
	}
	return ids
}

func readUnitDays(ctx context.Context, c *autotask.Client, id int, month time.Time, entity, lineField, itemField string) (unitDays, error) {
	nextMonth := month.AddDate(0, 1, 0)
	filters := []autotask.Filter{
		{Op: "eq", Field: "contractID", Value: id},
		{Op: "gte", Field: "endDate", Value: month.Format(time.RFC3339)},
	}
	// Autotask datetimes cannot represent January of the year 10000.
	if nextMonth.Year() <= 9999 {
		filters = append(filters, autotask.Filter{Op: "lt", Field: "startDate", Value: nextMonth.Format(time.RFC3339)})
	}
	rows, err := queryAll(ctx, c, entity, filters)
	if err != nil {
		return nil, err
	}
	var periods []unitPeriod
	for _, row := range rows {
		p, err := parsePeriod(row, id, lineField, itemField)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entity, err)
		}
		if p.start.Before(nextMonth) && !p.end.Before(month) {
			periods = append(periods, p)
		}
	}
	days, err := expandPeriods(month, periods)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", entity, err)
	}
	return days, nil
}

func parsePeriod(row map[string]any, contractID int, lineField, itemField string) (unitPeriod, error) {
	var p unitPeriod
	id, err := requiredInt(row, "contractID", 1, maxID)
	if err != nil || id != contractID {
		return p, fmt.Errorf("period is outside the requested contract")
	}
	if p.lineID, err = requiredInt(row, lineField, 1, maxID); err != nil {
		return p, err
	}
	if p.itemID, err = requiredInt(row, itemField, 1, maxID); err != nil {
		return p, err
	}
	if p.units, err = requiredInt(row, "units", 0, math.MaxInt32); err != nil {
		return p, err
	}
	start, err := autotask.ParseContractTime(row["startDate"])
	if err != nil {
		return p, err
	}
	end, err := autotask.ParseContractTime(row["endDate"])
	if err != nil {
		return p, err
	}
	if end.Before(start) {
		return p, fmt.Errorf("period ends before it starts")
	}
	p.start = start.UTC().Truncate(24 * time.Hour)
	p.end = end.UTC().Truncate(24 * time.Hour)
	return p, nil
}

func expandPeriods(month time.Time, periods []unitPeriod) (unitDays, error) {
	slices.SortFunc(periods, func(a, b unitPeriod) int {
		if n := cmp.Compare(a.lineID, b.lineID); n != 0 {
			return n
		}
		return a.start.Compare(b.start)
	})
	periods = slices.Compact(periods)
	last := month.AddDate(0, 1, -1)
	days := make(unitDays, last.Day())
	for i, p := range periods {
		end := p.end
		if i+1 < len(periods) && periods[i+1].lineID == p.lineID {
			next := periods[i+1]
			if p.itemID != next.itemID || p.start.Equal(next.start) || end.After(next.start) {
				return nil, fmt.Errorf("conflicting periods for contract position %d", p.lineID)
			}
			// On a shared effective day the newer period owns that day.
			if end.Equal(next.start) {
				end = end.AddDate(0, 0, -1)
			}
		}
		start := p.start
		if start.Before(month) {
			start = month
		}
		if end.After(last) {
			end = last
		}
		for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
			index := day.Day() - 1
			days[index] = append(days[index], p.booking)
		}
	}
	return days, nil
}
