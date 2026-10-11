package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func TestPubSubBasic(t *testing.T) {
	hub := NewHub()

	server, user := net.Pipe()
	c := NewClient(server)
	defer user.Close()
	defer c.Close()

	if got := hub.Subscribe(c, "news"); got != 1 {
		t.Fatalf("Subscribe(news) = %d, want 1", got)
	}
	if got := hub.Subscribe(c, "sports"); got != 2 {
		t.Fatalf("Subscribe(sports) = %d, want 2", got)
	}

	if got := hub.Publish("nobody", []byte("x")); got != 0 {
		t.Fatalf("Publish to empty channel = %d, want 0", got)
	}
	if got := hub.Publish("news", []byte("hello")); got != 1 {
		t.Fatalf("Publish(news) = %d receivers, want 1", got)
	}

	// the subscriber must receive the exact RESP array
	want := "*3\r\n$7\r\nmessage\r\n$4\r\nnews\r\n$5\r\nhello\r\n"
	buf := make([]byte, len(want))
	user.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.ReadFull(user, buf); err != nil {
		t.Fatalf("reading pushed message: %v", err)
	}
	if string(buf) != want {
		t.Fatalf("got %q, want %q", buf, want)
	}

	// unsubscribing from one channel must not affect the other
	if got := hub.Unsubscribe(c, "news"); got != 1 {
		t.Fatalf("Unsubscribe(news) = %d, want 1", got)
	}
	if got := hub.Publish("news", []byte("again")); got != 0 {
		t.Fatalf("Publish(news) after unsubscribe = %d, want 0", got)
	}
	if got := hub.Publish("sports", []byte("goal")); got != 1 {
		t.Fatalf("Publish(sports) = %d, want 1 (still subscribed)", got)
	}

	hub.UnsubscribeAll(c)
	if got := hub.Publish("sports", []byte("x")); got != 0 {
		t.Fatalf("Publish after UnsubscribeAll = %d, want 0", got)
	}
}

func TestPubSubConcurrent(t *testing.T) {
	const (
		numClients    = 50
		numPublishers = 10
		numChannels   = 5
		publishes     = 200
	)

	hub := NewHub()
	var wg sync.WaitGroup

	// clients connect, subscribe, then disconnect
	// half close the connection before unsubscribing, so publishers hit clients that are already disconnected
	for i := range numClients {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			server, user := net.Pipe()
			go io.Copy(io.Discard, user) // drain so the writer never blocks
			defer user.Close()

			c := NewClient(server)
			for j := range numChannels {
				if (id + j) % 2 == 0 {
					hub.Subscribe(c, fmt.Sprintf("ch %d", j))
				}
			}
			time.Sleep(time.Duration(id % 5) * time.Millisecond)

			if id % 2 == 0 {
				c.Close()
				hub.UnsubscribeAll(c)
			} else {
				hub.UnsubscribeAll(c)
				c.Close()
			}
		}(i)
	}

	for range numPublishers {
		wg.Go(func() {
			for k := range publishes {
				hub.Publish(fmt.Sprintf("ch %d", k % numChannels), []byte("msg"))
			}
		})
	}

	// fail instead of hanging forever on a deadlock
	finished := make(chan struct{})
	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock: goroutines did not finish in 10 secs")
	}

	for j := range numChannels {
		if got := hub.Publish(fmt.Sprintf("ch %d", j), []byte("x")); got != 0 {
			t.Errorf("\"ch %d\" still has %d subscribers after all clients left", j, got)
		}
	}
}