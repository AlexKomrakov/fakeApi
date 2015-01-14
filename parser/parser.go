package parser

import (
	"math/rand"
	"reflect"
	"time"
	"regexp"
	"strings"
	"strconv"
)

func randomBool() bool {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(2) == 0
}

func randomInt(max interface {}) int {
	string, _ := max.(string) //TODO обработка ошибок
	max_int, _ := strconv.ParseInt(string, 10, 0)

	rand.Seed(time.Now().UnixNano())
	return rand.Intn(int(max_int))
}

func getParameters(str string) []reflect.Value {
	str = strings.Trim(str, "}{% ")
	params := strings.SplitN(str, ":", 2) //TODO автоматически определять размер параметров

	result := make([]reflect.Value, len(params)-1)
	for key, value := range params[1:] {
		result[key] = reflect.ValueOf(value)
	}

	return result
}

func ParseString(str string) interface {} {
	functions := map[string]interface{}{
		"{% randomBool %}": randomBool,
		"{% randomInt:[\\d]+ %}": randomInt,
	}
	for pattern, fnc := range functions {
		matched, _ := regexp.Match(pattern, []byte(str))
		if matched {
			f := reflect.ValueOf(fnc)
			params := getParameters(str)
			return f.Call(params)[0].Interface()
		}
	}

	return str
}
