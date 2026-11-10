package entity

import "time"

type RegionStatus string

const (
	RegionStatusOnline      RegionStatus = "ONLINE"
	RegionStatusOffline     RegionStatus = "OFFLINE"
	RegionStatusMaintenance RegionStatus = "MAINTENANCE"
)

type Region struct {
	ID       string
	Name     string
	Location string
	Status   RegionStatus
	Metadata map[string]any
}

type ResourceType struct {
	ID          string
	Name        string
	Unit        string
	IsDivisible bool
	MinAmount   int64
}

type RegionCapacity struct {
	RegionID          string
	ResourceTypeID    string
	TotalCapacity     int64
	AllocatedCapacity int64
	ReservedCapacity  int64
	UpdatedAt         time.Time
}

func (c RegionCapacity) AvailableCapacity() int64 {
	return c.TotalCapacity - c.AllocatedCapacity - c.ReservedCapacity
}
