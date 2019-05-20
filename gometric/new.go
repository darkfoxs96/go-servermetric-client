package gometric

import (
	"context"
	"time"
)

func NewClientCtx(servermetricHost, key, yourName, yourHost string, pushEvery time.Duration, ctx context.Context, stop context.CancelFunc) (client *Client, err error) {
	return newClient(servermetricHost, key, yourName, yourHost, pushEvery, ctx, stop)
}

func NewClient(servermetricHost, key, yourName, yourHost string, pushEvery time.Duration) (client *Client, err error) {
	ctx, stop := context.WithCancel(context.Background())
	return newClient(servermetricHost, key, yourName, yourHost, pushEvery, ctx, stop)
}

func newClient(servermetricHost, key, yourName, yourHost string, pushEvery time.Duration, ctx context.Context, stop context.CancelFunc) (client *Client, err error) {
	client = &Client{
		servermetricHost: servermetricHost,
		key:              key,
		Name:             yourName,
		Host:             yourHost,
		PushEvery:        pushEvery,
		metrics:          make(map[string]*MetricData),
		ctx:              ctx,
		stop:             stop,
	}

	_, err = client.Ping()
	if err != nil {
		return
	}

	err = client.Connect()
	if err != nil {
		return
	}

	return
}
