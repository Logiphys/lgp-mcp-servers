package autotask

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// FieldInfo represents metadata about an Autotask entity field.
type FieldInfo struct {
	Name                string          `json:"name"`
	DataType            string          `json:"dataType"`
	Length              int             `json:"length,omitempty"`
	IsRequired          bool            `json:"isRequired"`
	IsReadOnly          bool            `json:"isReadOnly"`
	IsQueryable         bool            `json:"isQueryable"`
	IsReference         bool            `json:"isReference"`
	ReferenceEntityType string          `json:"referenceEntityType,omitempty"`
	IsPickList          bool            `json:"isPickList"`
	PicklistValues      []PicklistValue `json:"picklistValues,omitempty"`
	PicklistParentField string          `json:"picklistParentValueField,omitempty"`
}

// PicklistValue represents a single picklist option.
type PicklistValue struct {
	Value          string `json:"value"`
	Label          string `json:"label"`
	IsDefaultValue bool   `json:"isDefaultValue"`
	SortOrder      int    `json:"sortOrder"`
	IsActive       bool   `json:"isActive"`
	IsSystem       bool   `json:"isSystem"`
	ParentValue    string `json:"parentValue,omitempty"`
}

// PicklistCache provides lazy-loaded, cached access to Autotask field metadata.
type PicklistCache struct {
	mu      sync.RWMutex
	cache   map[string][]FieldInfo
	loading map[string]chan struct{} // signals when a load completes
	client  *Client
	logger  *slog.Logger
}

// NewPicklistCache creates a new picklist cache.
func NewPicklistCache(client *Client, logger *slog.Logger) *PicklistCache {
	return &PicklistCache{
		cache:   make(map[string][]FieldInfo),
		loading: make(map[string]chan struct{}),
		client:  client,
		logger:  logger,
	}
}

// GetFields returns field info for an entity type, loading and caching on first access.
func (p *PicklistCache) GetFields(ctx context.Context, entityType string) ([]FieldInfo, error) {
	normalized := NormalizeEntityType(entityType)

	// Check cache (read lock)
	p.mu.RLock()
	if fields, ok := p.cache[normalized]; ok {
		p.mu.RUnlock()
		return fields, nil
	}
	p.mu.RUnlock()

	// Check if another goroutine is already loading (write lock)
	p.mu.Lock()
	// Double-check after acquiring write lock
	if fields, ok := p.cache[normalized]; ok {
		p.mu.Unlock()
		return fields, nil
	}
	if ch, ok := p.loading[normalized]; ok {
		p.mu.Unlock()
		// Wait for the other goroutine to finish loading
		<-ch
		p.mu.RLock()
		fields := p.cache[normalized]
		p.mu.RUnlock()
		return fields, nil
	}
	// Start loading
	ch := make(chan struct{})
	p.loading[normalized] = ch
	p.mu.Unlock()

	// Fetch from API
	fields, err := p.loadFields(ctx, normalized)

	p.mu.Lock()
	if err == nil {
		p.cache[normalized] = fields
	}
	delete(p.loading, normalized)
	close(ch)
	p.mu.Unlock()

	return fields, err
}

// GetPicklistValues returns active picklist values for a specific field.
func (p *PicklistCache) GetPicklistValues(ctx context.Context, entityType, fieldName string) ([]PicklistValue, error) {
	fields, err := p.GetFields(ctx, entityType)
	if err != nil {
		return nil, err
	}
	for _, f := range fields {
		if strings.EqualFold(f.Name, fieldName) && f.IsPickList {
			var active []PicklistValue
			for _, v := range f.PicklistValues {
				if v.IsActive {
					active = append(active, v)
				}
			}
			return active, nil
		}
	}
	return nil, nil
}

// GetQueues returns ticket queue picklist values.
func (p *PicklistCache) GetQueues(ctx context.Context) ([]PicklistValue, error) {
	return p.GetPicklistValues(ctx, "Tickets", "queueID")
}

// GetTicketStatuses returns ticket status picklist values.
func (p *PicklistCache) GetTicketStatuses(ctx context.Context) ([]PicklistValue, error) {
	return p.GetPicklistValues(ctx, "Tickets", "status")
}

// GetTicketPriorities returns ticket priority picklist values.
func (p *PicklistCache) GetTicketPriorities(ctx context.Context) ([]PicklistValue, error) {
	return p.GetPicklistValues(ctx, "Tickets", "priority")
}

// ClearCache removes cached field info.
func (p *PicklistCache) ClearCache(entityType string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if entityType == "" {
		p.cache = make(map[string][]FieldInfo)
	} else {
		delete(p.cache, NormalizeEntityType(entityType))
	}
}

func (p *PicklistCache) loadFields(ctx context.Context, entityType string) ([]FieldInfo, error) {
	p.logger.Debug("loading field info", "entity", entityType)
	rawFields, err := p.client.GetFieldInfo(ctx, entityType)
	if err != nil {
		return nil, fmt.Errorf("loading field info for %s: %w", entityType, err)
	}

	fields := make([]FieldInfo, 0, len(rawFields))
	for _, raw := range rawFields {
		fi := FieldInfo{}
		if v, ok := raw["name"].(string); ok {
			fi.Name = v
		}
		if v, ok := raw["dataType"].(string); ok {
			fi.DataType = v
		}
		if v, ok := raw["length"].(float64); ok {
			fi.Length = int(v)
		}
		if v, ok := raw["isRequired"].(bool); ok {
			fi.IsRequired = v
		}
		if v, ok := raw["isReadOnly"].(bool); ok {
			fi.IsReadOnly = v
		}
		if v, ok := raw["isQueryable"].(bool); ok {
			fi.IsQueryable = v
		}
		if v, ok := raw["isReference"].(bool); ok {
			fi.IsReference = v
		}
		if v, ok := raw["referenceEntityType"].(string); ok {
			fi.ReferenceEntityType = v
		}
		if v, ok := raw["isPickList"].(bool); ok {
			fi.IsPickList = v
		}
		if v, ok := raw["picklistParentValueField"].(string); ok {
			fi.PicklistParentField = v
		}
		// Parse picklist values
		if plRaw, ok := raw["picklistValues"].([]any); ok {
			for _, pvRaw := range plRaw {
				if pv, ok := pvRaw.(map[string]any); ok {
					val := PicklistValue{IsActive: true} // default active
					if v, ok := pv["value"].(string); ok {
						val.Value = v
					}
					if v, ok := pv["label"].(string); ok {
						val.Label = v
					}
					if v, ok := pv["isDefaultValue"].(bool); ok {
						val.IsDefaultValue = v
					}
					if v, ok := pv["sortOrder"].(float64); ok {
						val.SortOrder = int(v)
					}
					if v, ok := pv["isActive"].(bool); ok {
						val.IsActive = v
					}
					if v, ok := pv["isSystem"].(bool); ok {
						val.IsSystem = v
					}
					if v, ok := pv["parentValue"].(string); ok {
						val.ParentValue = v
					}
					fi.PicklistValues = append(fi.PicklistValues, val)
				}
			}
		}
		fields = append(fields, fi)
	}

	p.logger.Debug("loaded fields", "entity", entityType, "count", len(fields))
	return fields, nil
}
