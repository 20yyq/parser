// @@
// @ Author       : Eacher
// @ Date         : 2023-03-07 09:53:53
// @ LastEditTime : 2023-03-07 14:58:20
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /parser/maps.go
// @@
package parser

import (
	"fmt"
	"strconv"
	"bytes"
	"reflect"
	"sync"
	"context"
)

type VALUE interface { string|bool|int64|uint64|float64|*Node }

type ContentType string

const (
	splitChar = '.'
	contentType = `json`

	ContentTypeJson       ContentType = `json`
	// ContentTypeJs         ContentType = `js`
	// ContentTypeXml        ContentType = `xml`
	// ContentTypeIni        ContentType = `ini`
	// ContentTypeYaml       ContentType = `yaml`
	// ContentTypeYml        ContentType = `yml`
	// ContentTypeToml       ContentType = `toml`
	// ContentTypeProperties ContentType = `properties`
)

type Config struct {
	SplitChar 	byte
	Type 		ContentType
	Sync 		bool
}

type Item struct {
	mutex 	sync.RWMutex
	value  	reflect.Value
}

type Node struct {
	key 	string
	value  	*Item
	prev 	*Node
	next 	*Node
}

func (n *Node) Key() string {
	return n.key
}

type Maps struct {
	ctx 	context.Context
	start 	*Node
	c 		*Config
}

type original struct {
	buf 	[]byte
	index 	int
	tail 	int
	Err 	error
}

// 读取JSON对象体
func (o *original) jsonObject() (*Item, error) {
	n := &Node{}
	if err := o.loadJson(n); err != nil {
		return nil, fmt.Errorf("unknown json loop error %s \n", err)
	}
	o.index++

	return &Item{value: reflect.ValueOf(n)}, nil
}

// 读取Array对象体
func (o *original) arrayObject() (*Item, error) {
	if array, err := o.loadArray(); err == nil {
		o.index++
		return &Item{value: reflect.ValueOf(array)}, nil
	}
	return nil, fmt.Errorf("unknown array loop error \n")
}

// 读取Number
func (o *original) numberAnalysis() (*Item, error) {
	b, item := o.readValue(), &Item{}
	if -1 != bytes.IndexByte(b, '.') || b[0] == 'N' {
		if f, err := strconv.ParseFloat(string(b[:len(b)]), 64); err == nil {
			item.value = reflect.ValueOf(f)
			return item, nil
		}
		return nil, fmt.Errorf("ParseFloat error \n")
	}
	base := 10
	if -1 != bytes.IndexByte(b, 'e') || -1 != bytes.IndexByte(b, 'E') {
		base = 16
	}
	if b[0] == '-' {
		if i, err := strconv.ParseInt(string(b[:len(b)]), base, 64); err == nil {
			item.value = reflect.ValueOf(i)
			return item, nil
		}
		return nil, fmt.Errorf("ParseInt error \n")
	}
	start := 0
	if b[0] == '+' {
		start = 1
	}
	if i, err := strconv.ParseUint(string(b[start:len(b)]), base, 64); err == nil {
		item.value = reflect.ValueOf(i)
		return item, nil
	}
	return nil, fmt.Errorf("ParseUint error \n")
}

// 跳过单个任意字节
func (o *original) skipByte(skip []byte) bool {
	i := o.index
	for _, v := range o.buf[i:] {
		if o.index++; -1 == bytes.IndexByte(skip, v) {
			o.index--
			return true
		}
	}
	if i != o.index { o.index-- }
	return false
}

// 选择对象、数组和其他
func (o *original) electFunc() func ()(*Item, error) {
	switch o.buf[o.index] {
	case '{':
		return o.jsonObject
	case '[':
		return o.arrayObject
	case '"':
		return func ()(*Item, error) {
			b := o.readString()
			o.index++
			return &Item{value: reflect.ValueOf(b)}, nil
		}
	case '1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '+', 'N':
		return o.numberAnalysis
	case 'n':
		return func ()(*Item, error) {
			o.readValue()
			return &Item{value: reflect.ValueOf(nil)}, nil
		}
	case 't', 'f':
		return func ()(*Item, error) {
			var b bool
			if o.buf[o.index] == 't' { b = true }
			o.readValue()
			return &Item{value: reflect.ValueOf(b)}, nil
		}
	}
	return func ()(*Item, error) {
		return nil, fmt.Errorf("unknown value type %s \n", string(o.buf[o.index]))
	}
}

// 读取字符串
func (o *original) readString() string {
	i, num := o.index, 0
	for _, v := range o.buf[i:] {
		if o.index++; '"' == v {
			if num++; num > 1 && '\\' != o.buf[o.index-2] {
				o.index--
				return string(o.buf[i+1:o.index])
			}
		}
	}
	if i != o.index { o.index-- }
	return ""
}

// 读取值
func (o *original) readValue() []byte {
	i := o.index
	for _, v := range o.buf[i:] {
		if o.index++; -1 == bytes.IndexByte(skipValue, v) {
			continue
		}
		o.index--
		return bytes.Clone(o.buf[i:o.index])
	}
	if o.index != i { o.index-- }
	return nil
}

// 加载json
func (o *original) loadJson(n *Node) error {
	if !o.skipByte(spaces) || o.buf[o.index] != '{' {
		return fmt.Errorf("last bytes %s \n", o.buf[o.index:])
	}
	err := fmt.Errorf("last bytes %s \n", o.buf[o.index:])
	if o.index++; o.skipByte(spaces) {
		err = nil
		for o.buf[o.index] != '}' {
			n.next = &Node{prev: n}
			if o.buf[o.index] != '"' {
				n.next, err = nil, fmt.Errorf("last bytes %s \n", o.buf[o.index:])
				break
			}
			if n.key = o.readString(); n.key == "" {
				n.next, err = nil, fmt.Errorf("last bytes %s \n", o.buf[o.index:])
				break
			}
			if o.index++; !o.skipByte(append(spaces, ':')) {
				n.next, err = nil, fmt.Errorf("last bytes %s \n", o.buf[o.index:])
				break
			}
			if n.value, err = o.electFunc()(); err != nil {
				n.next, err = nil, fmt.Errorf("error %s, last bytes %s \n", err, o.buf[o.index:])
				break
			}
			n = n.next
			if o.skipByte(spaces); o.buf[o.index] != ',' {
				continue
			}
			o.index++
			o.skipByte(spaces) 
		}
	}
	return err
}

// 加载array
func (o *original) loadArray() ([]*Item, error) {
	var value []*Item
	if !o.skipByte(spaces) || o.buf[o.index] != '[' {
		return nil, fmt.Errorf("loadArray start %s \n", o.buf[o.index:])
	}
	if o.index++; o.skipByte(spaces) {
		for o.buf[o.index] != ']' {
			if v, err := o.electFunc()(); err == nil {
				value = append(value, v)
				o.skipByte(spaces)
				if o.buf[o.index] != ',' {
					continue
				}
				if o.index++; !o.skipByte(spaces) {
					return nil, fmt.Errorf("loadArray end %s \n", o.buf[o.index:])
				}
				continue
			}
			return nil, fmt.Errorf("loadArray loading %s \n", o.buf[o.index:])
		}
	}
	return value, nil
}

var spaces = []byte{' ', '\n', '\t', '\r'}
var skipValue = []byte{' ', '\n', '\t', '\r', ',', '}', ']'}

func NewMaps(b []byte, c Config) *Maps {
	m, o := &Maps{start: &Node{}, c: &c}, &original{buf: b, tail: len(b), index: 0}
	var cause context.CancelCauseFunc
	m.ctx, cause = context.WithCancelCause(context.Background())
	go func() { cause(o.loadJson(m.start)) }()
	if c.Sync { m.Load() }
	if m.c.SplitChar == 0 {
		m.c.SplitChar = byte(splitChar)
	}
	if m.c.Type == "" {
		m.c.Type = contentType
	}
	return m
}

// Load
func (m *Maps) Load() (err error) {
	<-m.ctx.Done()
	if err = context.Cause(m.ctx); err == context.Canceled {
		err = nil
	}
	return err
}

func findItem(n *Node, key string) *Item {
	num, err := strconv.ParseInt(key, 10, 8)
	for n != nil {
		if n.key == key {
			return n.value
		}
		if n.value != nil && err == nil && reflect.Slice == n.value.value.Kind() {
			if n.value.value.Len() > int(num) {
				if v := n.value.value.Index(int(num)); v.Kind() == reflect.Pointer {
					i, _ := v.Interface().(*Item)
					return i
				}
			}
		}
		n = n.next
	}
	return nil
}

// Get
func Get[V VALUE](m *Maps, keys string) (v V, ok bool) {
	var item *Item
	node, list, index := m.start, bytes.Split([]byte(keys), []byte{m.c.SplitChar}), 0
	for node != nil {
		item = findItem(node, string(list[index]))
		if node = nil; item != nil && item.value.Kind() == reflect.Pointer {
			if n, o := item.value.Interface().(*Node); o {
				node = n
			}
		}
		if index++; index == len(list) {
			if item != nil && reflect.TypeOf(v).Kind() == item.value.Kind() {
				v, ok = itemValue[V](item), true
			}
			break
		}
	}
	return
}

// 
func itemValue[V VALUE](item *Item) (v V) {
	var a any
	switch reflect.TypeOf(v).Kind() {
	case reflect.Pointer:
		if n, o := item.value.Interface().(*Node); o {
			a = n
		}
	case reflect.String:
		a = item.value.String()
	case reflect.Int64:
		a = item.value.Int()
	case reflect.Uint64:
		a = item.value.Uint()
	case reflect.Float64:
		a = item.value.Float()
	case reflect.Bool:
		a = item.value.Bool()
	}
	v1, ok := a.(V)
	if ok { v = v1 }
	return 
}

// Set
func Set[V VALUE](m *Maps, key string, v V) bool {
	return false
}