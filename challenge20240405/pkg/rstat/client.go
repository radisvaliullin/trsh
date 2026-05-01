package rstat

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"
)

var _ IClient = (*Client)(nil)

type IClient interface {
	// Requests posts by 100 items
	// after is item id for request next 100 items after id
	SubredditNew(after string) (Resp, error)
	GetTotalReqCnt() int
}

type ClientConfig struct {
	Token     string
	Subreddit string
}

type Client struct {
	config ClientConfig

	cln *http.Client

	// requests total count, use atomic
	totalReqCnt uint64
}

func NewClient(config ClientConfig) *Client {
	cln := &Client{
		config: config,
	}
	cln.cln = &http.Client{
		Timeout: time.Second * 5,
	}
	return cln
}

func (c *Client) SubredditNew(after string) (Resp, error) {
	return c.subredditNew(after, "", "")
}

func (c *Client) subredditNew(after, before, limit string) (Resp, error) {

	resp := Resp{}

	// query params
	if limit == "" {
		limit = "100"
	}
	vals := url.Values{}
	if len(after) != 0 {
		vals.Set("after", after)
	}
	if len(before) != 0 {
		vals.Set("before", before)
	}
	if len(limit) != 0 {
		vals.Set("limit", limit)
	}

	// build url
	reqURL := url.URL{
		Scheme:   "https",
		Host:     "oauth.reddit.com",
		Path:     fmt.Sprintf("r/%s/new", c.config.Subreddit),
		RawQuery: vals.Encode(),
	}
	// build request
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		log.Fatalf("client: new request build error: %v", err)
	}
	req.Header.Set("User-Agent", "ChangeMeClient/0.1 by YourUsername")
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", c.config.Token))

	// request
	httpRes, err := c.cln.Do(req)
	if err != nil {
		log.Printf("client: request error, %v", err)
		return Resp{}, err
	}
	defer httpRes.Body.Close()
	atomic.AddUint64(&c.totalReqCnt, 1)

	// handle header
	used, _ := strconv.Atoi(httpRes.Header.Get("x-ratelimit-used"))
	remaining, _ := strconv.Atoi(httpRes.Header.Get("x-ratelimit-remaining"))
	reset, _ := strconv.Atoi(httpRes.Header.Get("x-ratelimit-reset"))
	resp.Header = Header{
		Used:      used,
		Remaining: remaining,
		Reset:     reset,
	}

	// error response
	if httpRes.StatusCode != 200 {
		errBody, err := io.ReadAll(httpRes.Body)
		if err != nil {
			log.Printf("client: response body read error, %v", err)
			return resp, err
		}
		if len(errBody) > 256 {
			errBody = errBody[:256]
		}
		err = fmt.Errorf("client: error response, code - %v, message - %v", httpRes.StatusCode, string(errBody))
		return resp, err
	}

	// decode response
	err = json.NewDecoder(httpRes.Body).Decode(&resp.Payload)
	if err != nil {
		log.Printf("client: response decode error: %v", err)
		return resp, err
	}

	return resp, nil
}

func (c *Client) ping() (Resp, error) {
	return c.subredditNew("", "", "1")
}

func (c *Client) GetTotalReqCnt() int {
	return int(atomic.LoadUint64(&c.totalReqCnt))
}
