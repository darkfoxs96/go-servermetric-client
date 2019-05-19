package main

type ConnectReq struct {
	Name string `json:"name"`
	Host string `json:"host"`
}

type ConnectResp struct {
	Status string `json:"status"`
	ID     int64  `json:"id"`
}

type PushMetrics struct {
	ServerID int64                      `json:"serverId"`
	Name     string                     `json:"name"`
	Metrics  map[string][][]interface{} `json:"metrics"`
}

type EventPush struct {
	Ping  int64
	Error error
}
