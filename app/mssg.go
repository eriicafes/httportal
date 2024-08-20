package app

import (
	"fmt"
)

type Mssg struct {
	Event string
	Data  string
}

func (m Mssg) String() string {
	var res string
	if m.Event != "" {
		res = appendMssg(res, fmt.Sprintf("event: %s", m.Event))
	}
	res = appendMssg(res, fmt.Sprintf("data: %s", m.Data))
	if res == "" {
		return ":\n\n"
	}
	return res + "\n\n"
}

func appendMssg(src, s string) string {
	if src == "" {
		return s
	}
	return src + "\n" + s
}
