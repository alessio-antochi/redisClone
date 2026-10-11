package main

import (
	"bytes"
	"fmt"
	"strconv"
	"time"
)

func ping(strings []any) ([]byte, error)  {
	if len(strings) == 1 {
		return []byte("+PONG\r\n"), nil
	}

	return joinWords(strings[1:], " ")
}

func echo(strings []any) ([]byte, error)  {
	if len(strings) == 1 {
		return []byte("-Invalid echo: expecting a message\r\n"), nil
	}

	return joinWords(strings[1:], " ")
}

func set(strings []any, s *Store) ([]byte, error) {
	if len(strings) < 3 {
		return []byte("-SET command: not enough arguments\r\n"), nil
	}

	keyBulk, ok := strings[1].(bulkString)
	if !ok {
		return nil, fmt.Errorf("%q is not of type bulkString", strings[1])
	}
	key := string(keyBulk.word)

	valueBulk, ok := strings[2].(bulkString)
	if !ok {
		return nil, fmt.Errorf("%q is not of type bulkString", strings[2])
	}
	value := bytes.Clone(valueBulk.word)

	// TODO flags

	s.Set(key, value)

	return []byte("+OK\r\n"), nil
}

func get(strings []any, s *Store) ([]byte, error) {
	if len(strings) != 2 {
		return []byte("-GET command: not enough arguments\r\n"), nil
	}

	keyBulk, ok := strings[1].(bulkString)
	if !ok {
		return nil, fmt.Errorf("%q is not of type bulkString", strings[1])
	}
	key := string(keyBulk.word)

	value, ok := s.Get(key)
	if !ok {
		return []byte("_\r\n"), nil
	}

	return encodeBulkString(value), nil
}

func del(strings []any, s *Store) ([]byte, error) {
	if len(strings) < 2 {
		return []byte("-DEL command: not enough arguments\r\n"), nil
	}

	deletedKeys := 0
	for _, str := range strings[1:] {
		keyBulk, ok := str.(bulkString)
		if !ok {
			return nil, fmt.Errorf("%q is not of type bulkString", strings[1])
		}
		key := string(keyBulk.word)

		if s.Del(key) {
			deletedKeys++
		}
	}

	return encodeInt(deletedKeys), nil
}

func exists(strings []any, s *Store) ([]byte, error) {
	if len(strings) < 2 {
		return []byte("-EXISTS command: not enough arguments\r\n"), nil
	}

	existingKeys := 0
	for _, str := range strings[1:] {
		keyBulk, ok := str.(bulkString)
		if !ok {
			return nil, fmt.Errorf("%q is not of type bulkString", strings[1])
		}
		key := string(keyBulk.word)

		if s.Exists(key) {
			existingKeys++
		}
	}

	return encodeInt(existingKeys), nil
}

func expire(strings []any, s *Store) ([]byte, error) {
	if len(strings) < 3 {
		return []byte("-EXPIRE command: not enough arguments\r\n"), nil
	}

	// TODO flags

	keyBulk, ok := strings[1].(bulkString)
	if !ok {
		return nil, fmt.Errorf("%q is not of type bulkString", strings[1])
	}
	key := string(keyBulk.word)

	durationBulk, ok := strings[2].(bulkString)
	if !ok {
		return nil, fmt.Errorf("%q is not of type bulkString", strings[2])
	}
	duration, err := strconv.Atoi(string(durationBulk.word))
	if err != nil {
		return encodeInt(0), fmt.Errorf("Error: %s when getting expiration duration: %q", err, durationBulk.word)
	}

	if s.Expire(key, time.Duration(duration) * time.Second) {
		return encodeInt(1), nil
	}
	return encodeInt(0), nil
}

func ttl(strings []any, s *Store) ([]byte, error) {
	if len(strings) != 2 {
		return []byte("-TTL command: not enough arguments\r\n"), nil
	}

	keyBulk, ok := strings[1].(bulkString)
	if !ok {
		return nil, fmt.Errorf("%q is not of type bulkString", strings[1])
	}
	key := string(keyBulk.word)

	return encodeInt(s.Ttl(key)), nil
}

func subscribe(strings []any, hub *Hub, c *Client) ([]byte, error) {
	if len(strings) < 2 {
		return []byte("-SUBSCRIBE command: not enough arguments\r\n"), nil
	}

	var ans []byte

	for i, str := range strings[1:] {
		channelBulk, ok := str.(bulkString)
		if !ok {
			return nil, fmt.Errorf("%q is not of type bulkString", strings[1])
		}
		channel := string(channelBulk.word)

		subscribedChannels := hub.Subscribe(c, channel)

		ans = encodePushedMessage("subscribe", channel, encodeInt(subscribedChannels))
		if i + 2 < len(strings) {
			c.Send(ans)
		}
	}

	return ans, nil
}

func unsubscribe(strings []any, hub *Hub, c *Client) ([]byte, error) {
	var ans []byte
	for i, str := range strings[1:] {
		channelBulk, ok := str.(bulkString)
		if !ok {
			return nil, fmt.Errorf("%q is not of type bulkString", strings[1])
		}
		channel := string(channelBulk.word)

		subscribedChannels := hub.Unsubscribe(c, channel)

		ans = encodePushedMessage("unsubscribe", channel, encodeInt(subscribedChannels))
		if i + 2 < len(strings) {
			c.Send(ans)
		}
	}

	return ans, nil
}

func unsubscribeAll(hub *Hub, c *Client) ([]byte, error) {
	var ans []byte
	for channel := range c.subs {
		subscribedChannels := hub.Unsubscribe(c, channel)

		ans = encodePushedMessage("unsubscribe", channel, encodeInt(subscribedChannels))
		if subscribedChannels > 0 {
			c.Send(ans)
		}
	}

	return ans, nil
}

func publish(strings []any, hub *Hub) ([]byte, error) {
	if len(strings) != 3 {
		return []byte("-PUBLISH command: not enough arguments\r\n"), nil
	}

	channelBulk, ok := strings[1].(bulkString)
	if !ok {
		return nil, fmt.Errorf("%q is not of type bulkString", strings[1])
	}
	channel := string(channelBulk.word)

	message, ok := strings[2].(bulkString)
	if !ok {
		return nil, fmt.Errorf("%q is not of type bulkString", strings[1])
	}

	ans := hub.Publish(channel, bytes.Clone(message.word))

	return encodeInt(ans), nil
}
