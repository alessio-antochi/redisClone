package main

import (
	"bytes"
	"fmt"
	"strconv"
)

type simpleString struct {
	word string
	isError bool
}

type bulkString struct {
	word []byte
	isError bool
}

// all types here: https://redis.io/docs/latest/develop/reference/protocol-spec/#resp-protocol-description
// expect "<type(1 byte)><data>\r\n"
func parseRESP(n int, buf []byte) (any, int, error) {
	if len(buf) < 3 || !bytes.Contains(buf[1:], []byte("\r\n")) {
		return nil, 0, fmt.Errorf("Invalid RESP element: %q", buf);
	}

	switch buf[0] {
	case '+':
		return parseRespSimpleString(buf, false)
	case '-':
		return parseRespSimpleString(buf, true)
	case ':':
		return parseRespInt(buf)
	case '$':
		return parseRespBulkString(buf, false)
	case '*':
		return parseRespArray(n, buf)
	case '_':
		return parseRespNull(n, buf)
	case '#':
		return parseRespBoolean(n, buf)
	case ',':
		return parseRespDouble(n, buf)
	case '(':
		return parseRespBigNumber(n, buf)
	case '!':
		return parseRespBulkString(buf, true)
	case '=':
		return parseRespVerbatimString(n, buf)
	case '%':
		return parseRespMap(n, buf)
	case '|':
		return parseRespAttribute(n, buf)
	case '~':
		return parseRespSet(n, buf)
	case '>':
		return parseRespPush(n, buf)
	default:
		return nil, 0, fmt.Errorf("Not a valid type: %q", buf)
	}
}

// expect "+<string>\r\n" OR "-<string>\r\n"
// works for both simple strings and simple errors
func parseRespSimpleString(buf []byte, isError bool) (simpleString, int, error) {
	end := bytes.Index(buf[1:], []byte("\r\n")) + 3
	return simpleString{ word: string(buf[1:end - 2]), isError: isError}, end, nil
}

// expect ":<int>\r\n"
func parseRespInt(buf []byte) (int, int, error) {
	end := bytes.Index(buf[1:], []byte("\r\n")) + 3
	num, err := strconv.Atoi(string(buf[1:end - 2]))
	if err != nil {
		return 0, 0, fmt.Errorf("Error: %s when parsing int: %q", err, buf)
	}
	return num, end, nil
}

// expect "$<length>\r\n<data>\r\n" OR "!<length>\r\n<data>\r\n"
func parseRespBulkString(buf []byte, isError bool) (bulkString, int, error) {
	crlf := []byte("\r\n")

	right := bytes.Index(buf[1:], crlf) + 1
	len, err := strconv.Atoi(string(buf[1:right]))
	if err != nil {
		return bulkString{}, 0, fmt.Errorf("NAN length of bulk string: %q", buf)
	}

	if len == -1 {
		return bulkString{ word: nil, isError: isError }, right + 2, nil
	} else if len == 0 {
		return bulkString{ word: []byte{}, isError: isError }, right + 2, nil
	} else if len < 0 {
		return bulkString{}, 0, fmt.Errorf("Invalid lenght of bulk string: %d", len)
	}

	left := right + 2
	right = bytes.Index(buf[left:], crlf) + left

	if right - left != len {
		return bulkString{}, 0, fmt.Errorf("String length does not match the given length: %q", buf)
	}

	return bulkString{ word: buf[left:right], isError: isError }, right + 2, nil
}

// expect "*<number-of-elements>\r\n<element-1>...<element-n>
func parseRespArray(n int, buf []byte) ([]any, int, error) {
	// we'll use two variables left and right to get the slices of the buffer that contain the "actual" values
	crlf := []byte("\r\n")
	
	right := bytes.Index(buf[1:], crlf) + 1
	nElems, err := strconv.Atoi(string(buf[1:right]))
	if err != nil {
		return nil, 0, fmt.Errorf("Invalid number of elements in array: %q", buf)
	}

	arr := make([]any, nElems)
	left := right + 2
	var word any

	if nElems < 0 {
		return nil, 0, fmt.Errorf("Negative number of elements in array: %q", buf)
	}

	for i := range nElems {
		oldLeft := left

		word, left, err = parseRESP(n - left, buf[left:])
		if err != nil {
			return nil, 0, fmt.Errorf("Invalid elemnt in array: %q, got error: %s", buf[left:], err)
		}
		left += oldLeft
		
		arr[i] = word
	}

	return arr, left, nil
}

// expect "<type(1 byte)><data>\r\n"
func parseRespNull(n int, buf []byte) (any, int, error) {
	return nil, 0, nil
}

// expect "<type(1 byte)><data>\r\n"
func parseRespBoolean(n int, buf []byte) (any, int, error) {
	return nil, 0, nil
}

// expect "<type(1 byte)><data>\r\n"
func parseRespDouble(n int, buf []byte) (any, int, error) {
	return nil, 0, nil
}

// expect "<type(1 byte)><data>\r\n"
func parseRespBigNumber(n int, buf []byte) (any, int, error) {
	return nil, 0, nil
}

// expect "<type(1 byte)><data>\r\n"
func parseRespVerbatimString(n int, buf []byte) (any, int, error) {
	return nil, 0, nil
}

// expect "<type(1 byte)><data>\r\n"
func parseRespMap(n int, buf []byte) (any, int, error) {
	return nil, 0, nil
}

// expect "<type(1 byte)><data>\r\n"
func parseRespAttribute(n int, buf []byte) (any, int, error) {
	return nil, 0, nil
}

// expect "<type(1 byte)><data>\r\n"
func parseRespSet(n int, buf []byte) (any, int, error) {
	return nil, 0, nil
}

// expect "<type(1 byte)><data>\r\n"
func parseRespPush(n int, buf []byte) (any, int, error) {
	return nil, 0, nil
}


// ============ ENCODING FUNCTIONS =============================================

func encodeInt(nr int) []byte {
	ans := []byte(":")
	return append(ans, []byte(strconv.Itoa(nr) + "\r\n")...)
}

func encodeBulkString(str []byte) []byte {
	ans := []byte{'$'}
	ans = append(ans, []byte(strconv.Itoa(len(str)) + "\r\n")...)
	ans = append(ans, str...)
	return append(ans, []byte("\r\n")...)
}

// !!! takes as parameter an array of strings, NOT  bulk strings; automatically covnerts them to bulk strings
func encodeArrayOfBulkStrings(strs [][]byte) []byte {
	ans := []byte("*" + strconv.Itoa(len(strs)) + "\r\n")
	for _, str := range strs {
		ans = append(ans, encodeBulkString(str)...)
	}
	return ans
}

// msgType and channel are strings whcich will be converted into bulk strings, while msg is aleardy a valid RESP type(ex: bulkString, integer)
func encodePushedMessage(msgType string, channel string, msg []byte) []byte {
	ans := []byte("*3\r\n")
	ans = append(ans, encodeBulkString([]byte(msgType))...)
	ans = append(ans, encodeBulkString([]byte(channel))...)
	return append(ans, msg...)
}
