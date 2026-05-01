package rstat

import "time"

const (
	// requests per 10 min
	DefaultReqLimit = 600
	// 10 min
	DefaultTimeWindow      = time.Second * 60 * 10
	DefaultTimeWindowDelta = time.Second * 20
)

var _ IClient = (*ClientLimiter)(nil)

type ClientLimiter struct {
	client *Client

	nextReq chan struct{}
}

func NewClientLimiter(client *Client) *ClientLimiter {
	cl := &ClientLimiter{
		client:  client,
		nextReq: make(chan struct{}),
	}
	go cl.run()
	return cl
}

func (cl *ClientLimiter) run() {
	// make ping to get request limit info
	res, _ := cl.client.ping()
	reqLimit := res.Header.Used + res.Header.Remaining
	if reqLimit == 0 {
		reqLimit = DefaultReqLimit
	}

	// delta need to make little bit less request than limit to escape errors
	nextReqDur := (DefaultTimeWindow + DefaultTimeWindowDelta) / time.Duration(reqLimit)
	reqTick := time.NewTicker(nextReqDur)

	// to unlock first request without delay set first signal
	cl.nextReq <- struct{}{}
	//
	for {
		<-reqTick.C
		cl.nextReq <- struct{}{}
	}
}

func (cl *ClientLimiter) SubredditNew(after string) (Resp, error) {
	<-cl.nextReq
	return cl.client.SubredditNew(after)
}

func (cl *ClientLimiter) GetTotalReqCnt() int {
	return cl.client.GetTotalReqCnt()
}
