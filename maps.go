// @@
// @ Author       : Eacher
// @ Date         : 2023-03-07 09:53:53
// @ LastEditTime : 2023-03-11 15:10:35
// @ LastEditors  : Eacher
// @ --------------------------------------------------------------------------------<
// @ Description  : 
// @ --------------------------------------------------------------------------------<
// @ FilePath     : /parser/maps.go
// @@
package parser

import (
	"fmt"
	"bytes"
	"sync"
	"context"
	"strconv"
)

type VALUE interface { string|bool|int64|uint64|float64|Node|[]any }

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
	value  	any
}

type Node struct {
	key 	string
	value  	any
	prev 	*Node
	next 	*Node
}

type Maps struct {
	ctx 	context.Context
	start 	*Node
	c 		*Config
}

var spaces = []byte{' ', '\n', '\t', '\r'}
var skipValue = []byte{' ', '\n', '\t', '\r', ',', '}', ']'}

func NewMaps(b []byte, c Config) *Maps {
	m, o := &Maps{start: &Node{}, c: &c}, &json{buf: b, tail: len(b), index: 0}
	var cause context.CancelCauseFunc
	m.ctx, cause = context.WithCancelCause(context.Background())
	go func(o *json) {
		err := o.loadJson(m.start)
		if err == nil {
			o.index++
			if o.skipByte(spaces); o.index < o.tail && -1 == bytes.IndexByte(spaces, o.buf[o.index]) {
				err = fmt.Errorf("eof %s \n", o.buf[o.index:])
			}
		}
		cause(err)
	}(o)
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

// Get
func Get[V VALUE](m *Maps, keys string) (v V, ok bool) {
	var a any
	var array []any
	node, list, index := m.start, bytes.Split([]byte(keys), []byte{m.c.SplitChar}), 0
	for node != nil {
		if node.key != string(list[index]) {
			node = node.next
			continue
		}
		a, node, array = node.value, nil, nil
		index++
	loop:
		switch a.(type) {
		case *Node:
			node = a.(*Node)
			a = *node
		case Item:
			item := a.(Item)
			a = item.value
		case []any:
			array = a.([]any)
			anyList := make([]any, len(array))
			copy(anyList, array)
			a = anyList
		}
		if index == len(list) {
			v, ok = a.(V)
			ok, node = true, nil
			break
		}
		if array != nil {
			num, err := strconv.ParseInt(string(list[index]), 10, 8)
			if err == nil && len(array) > int(num) {
				a, array = array[num], nil
				index++
				goto loop
			}
		}
		a = nil
	}
	return
}

// Set
func Set[V VALUE](m *Maps, keys string, v V) bool {
	return false
}