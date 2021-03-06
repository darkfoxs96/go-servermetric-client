package gometric

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/darkfoxs96/go-servermetric-client/tools"
)

// Errors
var (
	ErrBadKey         = fmt.Errorf("Servermetric-client: bad key")
	ErrAlreadyConnect = fmt.Errorf("Servermetric-client: already connect")
	ErrDontConnected  = fmt.Errorf("Servermetric-client: don't connected")
)

const (
	// API
	PING       = "/api/ping?key="
	METRIC     = "/api/metric?key="
	CONNECT    = "/api/connect?key="
	DISCONNECT = "/api/disconnect?key="
	// PARAMS
	CONTENT_TYPE_JSON = "application/json; charset=utf-8"
)

type Client struct {
	Name             string
	Host             string
	PushEvery        time.Duration
	servermetricHost string
	key              string
	alreadyConnect   bool
	id               int64
	metrics          map[string]*MetricData
	metricsMutex     sync.RWMutex
	ctx              context.Context
	stop             context.CancelFunc
}

func (c *Client) IsConnected() bool {
	return c.alreadyConnect
}

func (c *Client) RunPusher(fn func(ev EventPush)) {
	for {
		select {
		case <-c.ctx.Done():
			err := c.Disconnect()
			if fn != nil {
				if err != nil {
					fn(EventPush{0, err})
				} else {
					fn(EventPush{0, c.ctx.Err()})
				}
			}
			return
		case <-time.After(c.PushEvery):
			ping, err := c.PushMetrics()
			if fn != nil {
				fn(EventPush{ping, err})
			}
		}
	}
}

func (c *Client) Stop() {
	if c.stop != nil {
		c.stop()
	}
}

func (c *Client) AppendMetric(name string, fields string, types string, data ...interface{}) (err error) {
	defer c.metricsMutex.Unlock()
	c.metricsMutex.Lock()

	metric := c.metrics[name]
	if metric == nil {
		newV := strings.Replace(types, " ", "", -1)
		metric = &MetricData{
			Fields: fields,
			Types:  strings.Split(newV, ","),
			Data:   [][]interface{}{data},
		}
		c.metrics[name] = metric
		return
	}

	metric.Data = append(metric.Data, data)
	return
}

func (c *Client) PushMetrics() (ping int64, err error) {
	defer c.metricsMutex.RUnlock()
	c.metricsMutex.RLock()

	t1 := time.Now().UnixNano()
	err = c.pushMetrics()
	ping = time.Now().UnixNano() - t1
	if err != nil {
		return
	}

	c.ClearMetrics()
	return
}

func (c *Client) pushMetrics() (err error) {
	if !c.alreadyConnect || c.id == 0 {
		return ErrDontConnected
	}

	metrics := &PushMetrics{
		ServerID: c.id,
		Name:     c.Name,
		Metrics:  c.metrics,
	}

	b, err := json.Marshal(metrics)
	if err != nil {
		return
	}

	buf := bytes.NewReader(b)
	resp, err := http.DefaultClient.Post(c.servermetricHost+METRIC+c.key, CONTENT_TYPE_JSON, buf)
	if err != nil {
		time.Sleep(time.Second)

		resp, err = http.DefaultClient.Post(c.servermetricHost+METRIC+c.key, CONTENT_TYPE_JSON, buf)
		if err != nil {
			return
		}
	}

	if resp.StatusCode == http.StatusForbidden {
		return ErrBadKey
	}
	if resp.StatusCode != http.StatusOK {
		e := &Error{}
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(e)
		_ = resp.Body.Close()

		return fmt.Errorf("Servermetric-client: pushMetrics() status response %v, body: ", resp.StatusCode, e.Msg)
	}

	return
}

func (c *Client) ClearMetrics() {
	c.metrics = make(map[string]*MetricData)
}

func (c *Client) Ping() (ping int64, err error) {
	return c.ping(c.key)
}

func (c *Client) ping(key string) (ping int64, err error) {
	t1 := time.Now().UnixNano()

	resp, err := http.DefaultClient.Get(c.servermetricHost + PING + key)
	if err != nil {
		time.Sleep(time.Second)

		resp, err = http.DefaultClient.Get(c.servermetricHost + PING + key)
		if err != nil {
			return
		}
	}

	if resp.StatusCode == http.StatusForbidden {
		err = ErrBadKey
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Servermetric-client: ping() status response %v", resp.StatusCode)
		return
	}

	return time.Now().UnixNano() - t1, nil
}

func (c *Client) UpdateKey(newKey string) (err error) {
	_, err = c.ping(newKey)
	if err != nil {
		return
	}

	c.key = newKey
	return
}

func (c *Client) Connect() (err error) {
	if c.alreadyConnect {
		return ErrAlreadyConnect
	}

	connReq := &ConnectReq{
		Name: c.Name,
		Host: c.Host,
	}

	b, err := json.Marshal(connReq)
	if err != nil {
		return
	}

	buf := bytes.NewReader(b)
	resp, err := http.DefaultClient.Post(c.servermetricHost+CONNECT+c.key, CONTENT_TYPE_JSON, buf)
	if err != nil {
		time.Sleep(time.Second)

		resp, err = http.DefaultClient.Post(c.servermetricHost+CONNECT+c.key, CONTENT_TYPE_JSON, buf)
		if err != nil {
			return
		}
	}

	if resp.StatusCode == http.StatusForbidden {
		return ErrBadKey
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Servermetric-client: Client() status response %v", resp.StatusCode)
	}

	connResp := &ConnectResp{}
	err = tools.ParseJson(resp, connResp)
	if err != nil {
		return
	}

	c.id = connResp.ID
	c.alreadyConnect = true
	return
}

func (c *Client) Disconnect() (err error) {
	if !c.alreadyConnect || c.id == 0 {
		return ErrDontConnected
	}

	resp, err := http.DefaultClient.Get(c.servermetricHost + DISCONNECT + c.key + "&id=" + strconv.Itoa(int(c.id)))
	if err != nil {
		time.Sleep(time.Second)

		resp, err = http.DefaultClient.Get(c.servermetricHost + DISCONNECT + c.key + "&id=" + strconv.Itoa(int(c.id)))
		if err != nil {
			return
		}
	}

	if resp.StatusCode == http.StatusForbidden {
		return ErrBadKey
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Servermetric-client: Disconnect() status response %v", resp.StatusCode)
	}

	c.id = 0
	c.alreadyConnect = false
	return
}
