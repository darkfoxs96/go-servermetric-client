package main

import (
	"context"
	"time"
)

func NewClientCtx(servermetricHost, key, yourName, yourHost string, pushEvery time.Duration, listener func(ev EventPush), ctx context.Context, stop context.CancelFunc) (client *Client, err error) {
	return newClient(servermetricHost, key, yourName, yourHost, pushEvery, listener, ctx, stop)
}

func NewClient(servermetricHost, key, yourName, yourHost string, pushEvery time.Duration, listener func(ev EventPush)) (client *Client, err error) {
	return newClient(servermetricHost, key, yourName, yourHost, pushEvery, listener, nil, nil)
}

func newClient(servermetricHost, key, yourName, yourHost string, pushEvery time.Duration, listener func(ev EventPush), ctx context.Context, stop context.CancelFunc) (client *Client, err error) {
	client = &Client{
		servermetricHost: servermetricHost,
		key:              key,
		Name:             yourName,
		Host:             yourHost,
		PushEvery:        pushEvery,
	}

	_, err = client.Ping()
	if err != nil {
		return
	}

	err = client.Connect()
	if err != nil {
		return
	}

	go client.RunPusher(listener, ctx, stop)
	return
}
