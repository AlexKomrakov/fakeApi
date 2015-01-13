package parser

import (
	"math/rand"
	"reflect"
	"time"
	"regexp"
)

func randomBool() bool {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(2) == 0
}

func randomInt() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(100)
}

func ParseString(str string) interface {} {
	functions := map[string]interface{}{
		"{% randomBool %}": randomBool,
		"{% randomInt(:[\\d]+)? %}": randomInt,
	}
	for pattern, fnc := range functions {
		matched, _ := regexp.Match(pattern, []byte(str))
		if matched {
			f := reflect.ValueOf(fnc)
			params := make([]reflect.Value,0)
			return f.Call(params)[0].Interface()
		}
	}

	return str
}
