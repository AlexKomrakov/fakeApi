package parser

import (
	"math/rand"
	"reflect"
	"regexp"
	"strings"
	"strconv"
	"fmt"
	"encoding/binary"
	cryptorand "crypto/rand"
)

func getFunctionsMap() map[string]interface{} {
	return map[string]interface{}{
		"{% bool %}": randomBool,
		"{% int:[\\d]+ %}": randomInt,
	}
}

func seedRandom() {
	var seed int64
	binary.Read(cryptorand.Reader, binary.LittleEndian, &seed)
	rand.Seed(seed)
}

func randomBool() bool {
	seedRandom()
	return rand.Intn(2) == 0
}

func randomInt(max interface {}) int {
	string, _ := max.(string) //TODO обработка ошибок
	max_int, _ := strconv.ParseInt(string, 10, 0)

	seedRandom()
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

//TODO Избавиться от дублирования с callStringFunc
func ParseString(str string) interface {} {
	functions := getFunctionsMap()
	for pattern, fnc := range functions {
		exact := "^" + pattern + "$"
		matched, _ := regexp.Match(exact, []byte(str))
		if matched {
			f := reflect.ValueOf(fnc)
			params := getParameters(str)
			return f.Call(params)[0].Interface()
		}

		str = regexp.MustCompile(pattern).ReplaceAllStringFunc(str, callStringFunc)
	}

	return str
}

func callStringFunc(str string) string {
	functions := getFunctionsMap()
	for pattern, fnc := range functions {
		exact := "^" + pattern + "$"
		matched, _ := regexp.Match(exact, []byte(str))
		if matched {
			f := reflect.ValueOf(fnc)
			params := getParameters(str)
			return fmt.Sprint(f.Call(params)[0].Interface())
		}
	}

	return str
}
