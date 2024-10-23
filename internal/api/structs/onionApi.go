package structs

type OnionApi struct {
	To   string `json:"t"`
	From string `json:"f"`
	//ReceiveTimestampsPerLayer []int64 `json:"ts"` // unixMilliseconds
	OriginallySentTimestamp int64  `json:"s"` // unixMilliseconds
	LastSentTimestamp       int64  `json:"l"` // unixMilliseconds
	Onion                   string `json:"o"`
}

//func (oa OnionApi) AddReceiveTimestamp(t int64) OnionApi {
//	oa.ReceiveTimestampsPerLayer = append(oa.ReceiveTimestampsPerLayer, t)
//	return oa
//}
