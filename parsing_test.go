package main

import (
	"reflect"
	"testing"
)

type wordRespPair struct {
	word []byte
	response any
}

func TestParseRESP(t *testing.T) {
	badWords := [][]byte{nil,
						 []byte(""),
						 []byte("\r\n"),
						 []byte("abcdefg"),
						 []byte("?\r\n")}
	goodWords := [][]byte{[]byte(":0\r\n")}

	for _, word := range badWords {
		if _, _, err := parseRESP(len(word), word); err == nil {
			t.Errorf("Expected an error when parsing this message: %q", word)
		}
	}

	for _, word := range goodWords {
		if _, _, err := parseRESP(len(word), word); err != nil {
			t.Errorf("Got an error: %s when parsing this correct message: %q", err, word)
		}
	}
}

func TestParseRespSimpleString(t *testing.T) {
	goodPairs:= []wordRespPair{{ word: []byte("+Simple message\r\n"), response: simpleString{ word: "Simple message", isError: false } },
						  	   { word: []byte("-Some error\r\n"), response:  simpleString{ word: "Some error", isError: true } }}
	for _, pair := range goodPairs {
		if resp, _, err := parseRESP(len(pair.word), pair.word); err != nil || reflect.DeepEqual(resp, pair.response) == false {
			t.Errorf("Got an error: %s when parsing this correct string: %q", err, pair.word)
		}
	}

	word := []byte("+Only this\r\nnot supposed to appear\r\n")
	if parsed, _, err := parseRESP(len(word), word); err != nil || parsed != (simpleString{word: "Only this", isError: false}) {
		t.Errorf("Got an error or didn't parse this string correctly: %q. Got this: %s", word, parsed)
	}
}

func TestParseRespInt(t *testing.T) {
	goodWords := [][]byte{[]byte(":1283787\r\n"),
						  []byte(":+981237\r\n"),
						  []byte(":+981237\r\n"),
						  []byte(":9223372036854775807\r\n") /* INT64_MAX */ }
	badWords := [][]byte{[]byte(":++12313421\r\n"),
						 []byte(":-1234a123\r\n"),
						 []byte(":+1234-123\r\n"),
						 []byte(":9223372036854775808\r\n")}


	for _, word := range goodWords {
		if _, _, err := parseRESP(len(word), word); err != nil {
			t.Errorf("Got an error: %s when parsing this correct message: %q", err, word)
		}
	}

	for _, word := range badWords {
		if _, _, err := parseRESP(len(word), word); err == nil {
			t.Errorf("Expected an error when parsing this message: %q", word)
		}
	}
}

func TestParseRespBulkString(t *testing.T) {
	goodPairs := []wordRespPair{{word: []byte("$5\r\nhello\r\n"),
								 response: bulkString{ word: []byte("hello"), isError: false }},
								{word: []byte("$0\r\n"),
								 response: bulkString{ word: []byte{}, isError: false }},
								{word: []byte("$-1\r\n"),
								 response: bulkString{nil, false}},
								{word: []byte("!5\r\nhello\r\n"),
								 response: bulkString{ word: []byte("hello"), isError: true }},
								{word: []byte("!0\r\n"),
								 response: bulkString{ word: []byte{}, isError: true }},
								{word: []byte("!-1\r\n"),
								 response: bulkString{nil, true}},
								}
	badWords := [][]byte{[]byte("$5\r\nabcd\r\n"),
						 []byte("$5\r\nabcdefr\r\n"),
						 []byte("$5\r\n")}
	
	for _, pair := range goodPairs {
		if resp, _, err := parseRESP(len(pair.word), pair.word); err != nil || reflect.DeepEqual(resp, pair.response) == false {
			t.Errorf("Got an error: %s when parsing this correct string: %q", err, pair.word)
		}
	}

	for _, word := range badWords {
		if _, _, err := parseRESP(len(word), word); err == nil {
			t.Errorf("Expected an error when parsing this message: %q", word)
		}
	}
}

func TestParseRespArray(t *testing.T) {
	goodPairs := []wordRespPair{{word: []byte("*0\r\n"),
								 response: []any{}},
								{word: []byte("*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n"),
								 response: []any{bulkString{ word: []byte("hello"), isError: false }, bulkString{ word: []byte("world"), isError: false }}},
								{word: []byte("*3\r\n:1\r\n:2\r\n:3\r\n"),
								 response: []any{1, 2, 3}},
								{word: []byte("*2\r\n*3\r\n:1\r\n:2\r\n:3\r\n*2\r\n+Hello\r\n-World\r\n"),
								 response: []any{[]any{1, 2, 3}, []any{simpleString{ word: "Hello", isError: false },
								 									   simpleString{ word: "World", isError: true }}}},
							   }
	badWords := [][]byte{[]byte("*3\r\n$5\r\nhello\r\n$5\r\nworld\r\n"),
						 []byte("*1\r\n"),
						}
	
	for _, pair := range goodPairs {
		resp, _, err := parseRESP(len(pair.word), pair.word)
		if err != nil {
			t.Errorf("Got an error: %s when parsing this correct string: %q", err, pair.word)
		} else if reflect.DeepEqual(resp, pair.response) == false {
			t.Errorf("Got a wrong response: %q, expected: %q", resp, pair.response)
		}
	}

	for _, word := range badWords {
		if _, _, err := parseRESP(len(word), word); err == nil {
			t.Errorf("Expected an error when parsing this message: %q", word)
		}
	}
}
