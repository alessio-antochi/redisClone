package main

import (
	"net"
	"sync"
)

type Client struct {
	conn net.Conn
	out chan []byte
	done chan struct{}
	closeOnce sync.Once
	subs map[string]struct{}
}

func NewClient(conn net.Conn) *Client {
	c := &Client{
		conn: conn,
		out: make(chan []byte, 64),
		done: make(chan struct{}),
		subs: make(map[string]struct{}),
	}
	go func() {
		for {
			select {
			case msg := <-c.out:
				if _, err := c.conn.Write(msg); err != nil {
					c.Close()
					return
				}
			case <-c.done:
				return
			}
		}
	}();
	return c
}

func (c *Client) Send(msg []byte) bool {
	select {
	case c.out <- msg:
		return true
	case <-c.done:
		return false
	}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.conn.Close()
	})
}

type Hub struct {
	mu sync.Mutex
	channels map[string]map[*Client]struct{}
}

func NewHub() Hub {
	return Hub{channels: make(map[string]map[*Client]struct{})}
}

func (h *Hub) Subscribe(c *Client, channel string) int {
	c.subs[channel] = struct{}{}

	h.mu.Lock()
	_, ok := h.channels[channel]
	if !ok {
		h.channels[channel] = make(map[*Client]struct{})
	}
	h.channels[channel][c] = struct{}{}
	h.mu.Unlock()

	return len(c.subs)
}

func (h *Hub) Unsubscribe(c *Client, channel string) int {
	delete(c.subs, channel)

	h.mu.Lock()
	delete(h.channels[channel], c)
	h.mu.Unlock()

	return len(c.subs)
}

func (h *Hub) UnsubscribeAll(c *Client) {
	if len(c.subs) == 0 {
		return
	}

	h.mu.Lock()
	for channel := range c.subs {
		delete(h.channels[channel], c)
	}
	h.mu.Unlock()

	clear(c.subs)
}

func (h *Hub) Publish(channel string, payload []byte) int {
	h.mu.Lock()
	subs := make([]*Client, 0, len(h.channels[channel]))
	for c := range h.channels[channel] {
		subs = append(subs, c)
	}
	h.mu.Unlock()

	msgArray := [][]byte{[]byte("message"), []byte(channel), payload}
	msg := encodeArrayOfBulkStrings(msgArray)

	deletedSubs := make([]*Client, 0)
	for _, c := range subs {
		select {
		case c.out <- msg:
		case <-c.done:
			deletedSubs = append(deletedSubs, c)
		default:
			// client's buffer is full so we disconnect and delete it
			c.Close()
			deletedSubs = append(deletedSubs, c)
		}
	}

	if len(deletedSubs) > 0 {
		h.mu.Lock()
		for _, c := range deletedSubs {
			delete(h.channels[channel], c)
		}
		h.mu.Unlock()
	}

	return len(subs) - len(deletedSubs)
}
