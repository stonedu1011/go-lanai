package internal

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/utils"
	"encoding"
	"fmt"
	"github.com/go-kit/kit/log/level"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

var (
	TmplFuncMap = template.FuncMap{
		"cap": Capped,
		"pad": Padding,
		"lvl": MakeLevelFunc(true),
	}
	TmplFuncMapNonTerm = template.FuncMap{
		"cap": Capped,
		"pad": Padding,
		"lvl": MakeLevelFunc(false),
	}
)

type levelFuncs struct {
	text  func(int) string
	color func(interface{}) string
}

var (
	levelFuncsMap = map[string]levelFuncs{
		"debug": {
			text: MakeLevelPaddingFunc("DEBUG"),
			color: MakeQuickColorFunc(Gray),
		},
		"info": {
			text: MakeLevelPaddingFunc("INFO"),
			color: MakeQuickColorFunc(Cyan),
		},
		"warn": {
			text: MakeLevelPaddingFunc("WARN"),
			color: MakeQuickColorFunc(BoldYellow),
		},
		"error": {
			text: MakeLevelPaddingFunc("ERROR"),
			color: MakeQuickColorFunc(BoldRed),
		},
	}
)

func MakeKVFunc(ignored utils.StringSet) func(Fields) string {
	return func(kvs Fields) string {
		kvStrs := make([]string, 0, len(kvs))
		for k, v := range kvs {
			if v == nil || ignored.Has(k) || reflect.ValueOf(v).IsZero() {
				continue
			}
			kvStrs = append(kvStrs, fmt.Sprintf(`%s="%v"`, k, v))
		}

		if len(kvStrs) == 0 {
			return ""
		}
		return "{" + strings.Join(kvStrs, ", ") + "}"
	}
}

func MakeLevelFunc(term bool) func(kvs Fields, padding int) string {
	if term {
		return func(kvs Fields, padding int) string {
			lv, _ := kvs[level.Key().(string)]
			lvStr := Sprint(lv)
			if funcs, ok := levelFuncsMap[lvStr]; ok {
				return funcs.color(funcs.text(padding))
			}
			return lvStr
		}
	} else {
		return func(kvs Fields, padding int) string {
			lv, _ := kvs[level.Key().(string)]
			lvStr := Sprint(lv)
			if funcs, ok := levelFuncsMap[lvStr]; ok {
				return funcs.text(padding)
			}
			return lvStr
		}
	}
}

func MakeLevelPaddingFunc(v interface{}) func(int) string {
	return func(p int) string {
		return Padding(v, p)
	}
}

// Padding example: `{{padding value -6}}` "{{padding value 10}}"
func Padding(v interface{}, padding int) string {
	tag := "%" + strconv.Itoa(padding) + "v"
	return fmt.Sprintf(tag, v)
}

// Capped truncate given value to specified length, with tailing "..." if truncated
func Capped(v interface{}, cap int) string {
	s := Sprint(v)
	if len(s) <= cap {
		return s
	}
	return fmt.Sprintf("%." + strconv.Itoa(cap - 3) + "s...", s)
}


func Sprint(v interface{}) string {
	switch v.(type) {
	case string:
		return v.(string)
	case []byte:
		return string(v.([]byte))
	case fmt.Stringer:
		return v.(fmt.Stringer).String()
	case encoding.TextMarshaler:
		if s, e := v.(encoding.TextMarshaler).MarshalText(); e == nil {
			return string(s)
		}
	}
	return fmt.Sprintf("%v", v)
}
