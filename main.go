package main

import (
	"github.com/gorilla/mux"
	"encoding/json"
	"net/http"
	"strings"
	"io/ioutil"
	"io"
	"fmt"
	"os"
	"time"
	"log"
	"bytes"
	"os/exec"
	"path"
	"runtime"
)

var (
	startTime  = time.Now()
	logCont    bytes.Buffer
	logger     = log.New(&logCont, "<br/>logger: ", log.Lshortfile)
	requests   = 0
	routes     = make(map[string]int)
)

func readDir(path string) ([]os.FileInfo, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	return dir.Readdir(-1)
}

func defaultHandler(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "<a href='/exit' target='_blank'>Shut down server</a><br/>")
	io.WriteString(w, "<a href='/restart' target='_blank'>Restart server</a><br/>")

	io.WriteString(w, "<br/>Routes:<br/>")
	for route := range routes {
		io.WriteString(w, "<a href='"+route+"'>"+route+"</a><br/>")
	}

	io.WriteString(w, "<br/>Requests total: " + fmt.Sprint(requests) + "<br/>")

	io.WriteString(w, fmt.Sprint(&logCont))
}

func exitHandler(w http.ResponseWriter, req *http.Request) {
	os.Exit(0)
}

func restartHandler(w http.ResponseWriter, req *http.Request) {
	_, filename, _, _ := runtime.Caller(1)
	command := exec.Command(path.Join(path.Dir(filename), filename))
	fmt.Print(command.Start())
	os.Exit(0)
}

func processData(data interface{}) interface {} {
		switch dataType := data.(type) {
	case string:
		data = ParseString(dataType)
	case []interface{}:
		for key, value := range dataType {
			dataType[key] = processData(value)
		}
	case map[string]interface{}:
		for key, value := range dataType {
			dataType[key] = processData(value)
		}
	}
	return data
}

func jsonHandler(w http.ResponseWriter, req *http.Request, fileContent []byte) {
	requests++

	var content map[string]interface{}
	if err := json.Unmarshal(fileContent, &content); err != nil {
		panic(err)
	}
	processed := processData(content["data"])
	byteData, err := json.Marshal(processed)
	if err != nil {
		io.WriteString(w, "not a string")
	}
	io.WriteString(w, string(byteData))
}

func main() {
	logger.Println("Server started at " + startTime.String())
	router := mux.NewRouter()
	router.HandleFunc("/", defaultHandler)
	router.HandleFunc("/exit", exitHandler)
	router.HandleFunc("/restart", restartHandler)

	files, _ := readDir(".")
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		fileContent, err := ioutil.ReadFile(file.Name())
		if err != nil {
			fmt.Println("Error on reading file" + file.Name() + "\n")
		}
		var dat map[string]interface{}
		if err := json.Unmarshal([]byte(fileContent), &dat); err != nil {
			panic(err)
		}

		route, _ := dat["route"].(string)
		routes[route] = 1
		router.HandleFunc(route, func(w http.ResponseWriter, req *http.Request) {
			jsonHandler(w, req, []byte(fileContent))
		})
	}

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))
	http.Handle("/", router)
	http.ListenAndServe(":8888", nil)
}
