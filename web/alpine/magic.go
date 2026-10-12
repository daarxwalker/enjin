package alpine

import (
	"encoding/json"
	"fmt"
)

type magic struct {
	Event string
	El    string
	Store string
	Root  string
	Data  string
}

var Magic = magic{
	El:    "$el",
	Event: "$event",
	Store: "$store",
	Root:  "$root",
	Data:  "$data",
}

func (m magic) Refs(name string) string {
	return "$refs." + name
}

func (m magic) ID(name string) string {
	return "$id(" + jsString(name) + ")"
}

func (m magic) Watch(name, script string) string {
	return "$watch(" + jsString(name) + ", () => " + script + ")"
}

func (m magic) Dispatch(name string, data any) string {
	switch d := data.(type) {
	case string:
		return "$dispatch(" + jsString(name) + ", " + d + ")"
	default:
		dataBytes, err := json.Marshal(data)
		if err != nil {
			panic(fmt.Errorf("alpine: create dispatch %s failed: %w", name, err))
		}
		return "$dispatch(" + jsString(name) + ", " + string(dataBytes) + ")"
	}
}

func (m magic) NextTick(script string, await ...bool) string {
	r := "$nextTick(" + script + ")"
	if len(await) > 0 && await[0] {
		r = "await " + r
	}
	return r
}
