package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
)

func commands(command []byte, words []any, buf []byte, s *Store, hub *Hub, c *Client) ([]byte, error) {
	if bytes.Equal(command, []byte("PING")) {
		ans, err := ping(words)
		if err != nil {
			return nil, fmt.Errorf("Recieved invalid PING: %q", buf)
		}
		return ans, err
	}
	if bytes.Equal(command, []byte("ECHO")) {
		ans, err := echo(words)
		if err != nil {
			return nil, fmt.Errorf("Recieved invalid ECHO: %q", buf)
		}
		return ans, err
	}
	if bytes.Equal(command, []byte("SET")) {
		ans, err := set(words, s)
		if err != nil {
			return nil, fmt.Errorf("Recieved invalid SET: %q", buf)
		}
		return ans, err
	}
	if bytes.Equal(command, []byte("GET")) {
		ans, err := get(words, s)
		if err != nil {
			return nil, fmt.Errorf("Recieved invalid GET: %q", buf)
		}
		return ans, err
	}
	if bytes.Equal(command, []byte("DEL")) {
		ans, err := del(words, s)
		if err != nil {
			return nil, fmt.Errorf("Recieved invalid DEL: %q", buf)
		}
		return ans, err
	}

	if bytes.Equal(command, []byte("EXISTS")) {
		ans, err := exists(words, s)
		if err != nil {
			return nil, fmt.Errorf("Recieved invalid EXISTS: %q", buf)
		}
		return ans, err
	}

	if bytes.Equal(command, []byte("EXPIRE")) {
		ans, err := expire(words, s)
		if err != nil {
			return nil, fmt.Errorf("Recieved invalid EXPIRE: %q", buf)
		}
		return ans, err
	}

	if bytes.Equal(command, []byte("TTL")) {
		ans, err := ttl(words, s)
		if err != nil {
			return nil, fmt.Errorf("Recieved invalid TTL: %q", buf)
		}
		return ans, err
	}

	if bytes.Equal(command, []byte("SUBSCRIBE")) {
		ans, err := subscribe(words, hub, c)
		if err != nil {
			return nil, fmt.Errorf("Recieved invalid SUBSCRIBE: %q", buf)
		}
		return ans, err
	}

	if bytes.Equal(command, []byte("UNSUBSCRIBE")) {
		var ans []byte
		var err error
		if len(words) == 1 {
			ans, err = unsubscribeAll(hub, c)
		} else {
			ans, err = unsubscribe(words, hub, c)
		}
		if err != nil {
			return nil, fmt.Errorf("Recieved invalid UNSUBSCRIBE: %q", buf)
		}
		return ans, err
	}

	if bytes.Equal(command, []byte("PUBLISH")) {
		ans, err := publish(words, hub)
		if err != nil {
			return nil, fmt.Errorf("Recieved invalid PUBLISH: %q", buf)
		}
		return ans, err
	}

	return []byte("-ERR unknown command\r\n"), nil
}

func handleConn(conn net.Conn, s *Store, hub *Hub) {
	client := NewClient(conn)
	defer client.Close()
	defer hub.UnsubscribeAll(client)

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		parsed, _, err := parseRESP(n, buf)
		words, ok := parsed.([]any)
		if err != nil || !ok || len(words) == 0 {
			client.Send([]byte("-ERR protocol error\r\n"))
			continue
		}

		command, ok := words[0].(bulkString)
		if !ok || command.isError {
			client.Send([]byte("-ERR protocol error\r\n"))
			continue
		}

		ans, err := commands(command.word, words, buf, s, hub, client)
		if err != nil {
			log.Printf("Command error: %s", err)
			ans = []byte("-ERR invalid arguments\r\n")
		}

		if !client.Send(ans) {
			return
		}
	}
}

func main() {
	store := NewStore()
	hub := NewHub()

	const port = ":8080"
	ln, err := net.Listen("tcp", port)
	if err != nil {
		panic(err)
	}
	log.Printf("Listening on port: %s", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go handleConn(conn, &store, &hub)
	}
}
