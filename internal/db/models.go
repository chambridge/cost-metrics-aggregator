package db

import "time"

type Cluster struct {
	ID int
	Name string
}

type Node struct {
	  ID int
	  ClusterID int
	  Name string
	  Identifer string
	  Type string
}

type Metric struct {
	NodeID int
	Timestamp time.Time
	CoreCount int
}

type DailySummary struct {
	NodeID int
	Date time.Time
	CoreCount int
	TotalHours int
}
