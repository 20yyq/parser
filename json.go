// @@
// @ Author       : Eacher
// @ Date         : 2023-03-07 09:53:53
// @ LastEditTime : 2023-03-11 15:10:35
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /parser/json.go
// @@
package parser

import (
	"fmt"
	"strconv"
	"bytes"
)

type json struct {
	buf 	[]byte
	index 	int
	tail 	int
}

// 读取JSON对象体
func (j *json) jsonObject() (any, error) {
	n := &Node{}
	if err := j.loadJson(n); err != nil {
		return nil, fmt.Errorf("unknown json loop error %s \n", err)
	}
	j.index++

	return n, nil
}

// 读取Array对象体
func (j *json) arrayObject() (any, error) {
	if array, err := j.loadArray(); err == nil {
		j.index++
		return array, nil
	}
	return nil, fmt.Errorf("unknown array loop error \n")
}

// 读取Number
func (j *json) numberAnalysis() (any, error) {
	b := j.readValue()
	if -1 != bytes.IndexByte(b, '.') || b[0] == 'N' {
		if f, err := strconv.ParseFloat(string(b[:len(b)]), 64); err == nil {
			return Item{value: f}, nil
		}
		return nil, fmt.Errorf("ParseFloat error \n")
	}
	base := 10
	if -1 != bytes.IndexByte(b, 'e') || -1 != bytes.IndexByte(b, 'E') {
		base = 16
	}
	if b[0] == '-' {
		if i, err := strconv.ParseInt(string(b[:len(b)]), base, 64); err == nil {
			return Item{value: i}, nil
		}
		return nil, fmt.Errorf("ParseInt error \n")
	}
	start := 0
	if b[0] == '+' {
		start = 1
	}
	if i, err := strconv.ParseUint(string(b[start:len(b)]), base, 64); err == nil {
		return Item{value: i}, nil
	}
	return nil, fmt.Errorf("ParseUint error \n")
}

// 跳过单个任意字节
func (j *json) skipByte(skip []byte) bool {
	i := j.index
	for _, v := range j.buf[i:] {
		if j.index++; -1 == bytes.IndexByte(skip, v) {
			j.index--
			return true
		}
	}
	if i != j.index { j.index-- }
	return false
}

// 选择对象、数组和其他
func (j *json) electFunc() func ()(any, error) {
	switch j.buf[j.index] {
	case '{':
		return j.jsonObject
	case '[':
		return j.arrayObject
	case '"':
		return func ()(any, error) {
			b := j.readString()
			j.index++
			return Item{value: b}, nil
		}
	case '1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '+', 'N':
		return j.numberAnalysis
	case 'n':
		return func ()(any, error) {
			j.readValue()
			var n *Node
			return n, nil
		}
	case 't', 'f':
		return func ()(any, error) {
			var b bool
			if j.buf[j.index] == 't' { b = true }
			j.readValue()
			return Item{value: b}, nil
		}
	}
	return func ()(any, error) {
		return nil, fmt.Errorf("unknown value type %s \n", string(j.buf[j.index]))
	}
}

// 读取字符串
func (j *json) readString() string {
	i, num := j.index, 0
	for _, v := range j.buf[i:] {
		if j.index++; '"' == v {
			if num++; num > 1 && '\\' != j.buf[j.index-2] {
				j.index--
				return string(j.buf[i+1:j.index])
			}
		}
	}
	if i != j.index { j.index-- }
	return ""
}

// 读取值
func (j *json) readValue() []byte {
	i := j.index
	for _, v := range j.buf[i:] {
		if j.index++; -1 == bytes.IndexByte(skipValue, v) {
			continue
		}
		j.index--
		return bytes.Clone(j.buf[i:j.index])
	}
	if j.index != i { j.index-- }
	return nil
}

// 加载json
func (j *json) loadJson(n *Node) error {
	if !j.skipByte(spaces) || j.buf[j.index] != '{' {
		return fmt.Errorf("last bytes %s \n", j.buf[j.index:])
	}
	err := fmt.Errorf("last bytes %s \n", j.buf[j.index:])
	if j.index++; j.skipByte(spaces) {
		err = nil
		for j.buf[j.index] != '}' {
			n.next = &Node{prev: n}
			if j.buf[j.index] != '"' {
				n.next, err = nil, fmt.Errorf("last bytes %s \n", j.buf[j.index:])
				break
			}
			if n.key = j.readString(); n.key == "" {
				n.next, err = nil, fmt.Errorf("last bytes %s \n", j.buf[j.index:])
				break
			}
			if j.index++; !j.skipByte(append(spaces, ':')) {
				n.next, err = nil, fmt.Errorf("last bytes %s \n", j.buf[j.index:])
				break
			}
			if n.value, err = j.electFunc()(); err != nil {
				n.next, err = nil, fmt.Errorf("error %s, last bytes %s \n", err, j.buf[j.index:])
				break
			}
			n = n.next
			if j.skipByte(spaces); j.buf[j.index] != ',' {
				continue
			}
			j.index++
			j.skipByte(spaces) 
		}
	}
	return err
}

// 加载array
func (j *json) loadArray() ([]any, error) {
	var value []any
	if !j.skipByte(spaces) || j.buf[j.index] != '[' {
		return nil, fmt.Errorf("loadArray start %s \n", j.buf[j.index:])
	}
	if j.index++; j.skipByte(spaces) {
		for j.buf[j.index] != ']' {
			if v, err := j.electFunc()(); err == nil {
				value = append(value, v)
				j.skipByte(spaces)
				if j.buf[j.index] != ',' {
					continue
				}
				if j.index++; !j.skipByte(spaces) {
					return nil, fmt.Errorf("loadArray end %s \n", j.buf[j.index:])
				}
				continue
			}
			return nil, fmt.Errorf("loadArray loading %s \n", j.buf[j.index:])
		}
	}
	return value, nil
}

