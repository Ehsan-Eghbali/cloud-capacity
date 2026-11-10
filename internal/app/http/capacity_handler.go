package http

import (
	"net/http"

	"cloud-capacity-allocator/internal/core/port"
)

type regionDTO struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Location string         `json:"location"`
	Status   string         `json:"status"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type resourceTypeDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Unit        string `json:"unit"`
	IsDivisible bool   `json:"is_divisible"`
	MinAmount   int64  `json:"min_amount"`
}

type capacityDTO struct {
	ResourceTypeID    string `json:"resource_type_id"`
	TotalCapacity     int64  `json:"total_capacity"`
	AllocatedCapacity int64  `json:"allocated_capacity"`
	ReservedCapacity  int64  `json:"reserved_capacity"`
	AvailableCapacity int64  `json:"available_capacity"`
}

type regionCapacityDTO struct {
	RegionID     string        `json:"region_id"`
	RegionName   string        `json:"region_name"`
	RegionStatus string        `json:"region_status"`
	Capacities   []capacityDTO `json:"capacities"`
}

type CapacityHandler struct {
	service port.CapacityService
}

func NewCapacityHandler(service port.CapacityService) *CapacityHandler {
	return &CapacityHandler{service: service}
}

func (h *CapacityHandler) ListRegions(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.ListRegions(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	items := make([]regionDTO, 0, len(resp.Regions))
	for _, region := range resp.Regions {
		items = append(items, regionDTO{
			ID:       region.ID,
			Name:     region.Name,
			Location: region.Location,
			Status:   string(region.Status),
			Metadata: region.Metadata,
		})
	}
	writeData(w, http.StatusOK, items)
}

func (h *CapacityHandler) ListResourceTypes(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.ListResourceTypes(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	items := make([]resourceTypeDTO, 0, len(resp.ResourceTypes))
	for _, t := range resp.ResourceTypes {
		items = append(items, resourceTypeDTO{
			ID:          t.ID,
			Name:        t.Name,
			Unit:        t.Unit,
			IsDivisible: t.IsDivisible,
			MinAmount:   t.MinAmount,
		})
	}
	writeData(w, http.StatusOK, items)
}

func (h *CapacityHandler) GetRegionCapacity(w http.ResponseWriter, r *http.Request) {
	resp, err := h.service.GetRegionCapacity(r.Context(), port.GetRegionCapacityRequest{RegionID: r.PathValue("region_id")})
	if err != nil {
		writeError(w, r, err)
		return
	}

	items := make([]capacityDTO, 0, len(resp.Capacities))
	for _, c := range resp.Capacities {
		items = append(items, capacityDTO{
			ResourceTypeID:    c.ResourceTypeID,
			TotalCapacity:     c.TotalCapacity,
			AllocatedCapacity: c.AllocatedCapacity,
			ReservedCapacity:  c.ReservedCapacity,
			AvailableCapacity: c.AvailableCapacity(),
		})
	}
	writeData(w, http.StatusOK, regionCapacityDTO{
		RegionID:     resp.Region.ID,
		RegionName:   resp.Region.Name,
		RegionStatus: string(resp.Region.Status),
		Capacities:   items,
	})
}
