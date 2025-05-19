package WebUI

// frontend's FlowChargingRecord
type RatingGroupDataUsage struct {
	Supi      string `bson:"Supi" json:"supi"`
	Filter    string `bson:"Filter" json:"filter,omitempty"`
	Snssai    string `bson:"Snssai" json:"snssai"`
	Dnn       string `bson:"Dnn" json:"dnn"`
	TotalVol  int64  `bson:"TotalVol" json:"totalVol"`
	UlVol     int64  `bson:"UlVol" json:"ulVol"`
	DlVol     int64  `bson:"DlVol" json:"dlVol"`
	QuotaLeft int64  `bson:"QuotaLeft" json:"quotaLeft,omitempty"`
	Usage     int64  `bson:"Usage" json:"usage"`
}

