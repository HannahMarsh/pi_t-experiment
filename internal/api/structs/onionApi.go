package structs

type OnionApi struct {
	To                        string  `json:"t"`
	From                      string  `json:"f"`
	ReceiveTimestampsPerLayer []int64 `json:"ts"` // unixMilliseconds
	Onion                     string  `json:"o"`
}

func (oa OnionApi) AddReceiveTimestamp(t int64) OnionApi {
	oa.ReceiveTimestampsPerLayer = append(oa.ReceiveTimestampsPerLayer, t)
	return oa
}
