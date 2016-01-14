package main

import (
	"github.com/alexkomrakov/fakeapi/lib"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/unrolled/render"
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
    "flag"
	"strconv"
)

var (
	address = flag.String("a", ":8888", "Server address: host:port")

	upgrader  = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	startTime  = time.Now()
	logCont    bytes.Buffer
	logger     = log.New(&logCont, "Logger: ", log.Lshortfile)
	requests   = 0
	routes     = make(map[string]string)

	writeWait = 10 * time.Second
	pongWait = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10

	h = hub{
		broadcast:   make(chan []byte),
		register:    make(chan *connection),
		unregister:  make(chan *connection),
		connections: make(map[*connection]bool),
	}

	r *render.Render
)

func init() {
	r = render.New(render.Options{
		Extensions: []string{".tmpl", ".html"}, // Specify extensions to load for templates.
		IsDevelopment: true, // Render will now recompile the templates on every HTML response.
	})
}

func readDir(path string) ([]os.FileInfo, error) {
    dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

    return dir.Readdir(-1)
}

func defaultHandler(w http.ResponseWriter, req *http.Request) {
	var v = struct {
			Host string
			Routes map[string]string
			Requests string
			Log string
		}{
		req.Host,
		routes,
		fmt.Sprint(requests),
		fmt.Sprint(&logCont),
	}
	r.HTML(w, http.StatusOK, "index", v)
}

func exitHandler(w http.ResponseWriter, req *http.Request) {
	os.Exit(0)
}

func restartHandler(w http.ResponseWriter, req *http.Request) {
	command := exec.Command(os.Args[0])
	fmt.Print(os.Args)
	fmt.Print(command.Start())
	os.Exit(0)
}

func processData(data interface{}) interface {} {
	switch dataType := data.(type) {
		case string:
			data = lib.ParseString(dataType)
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

	h.broadcast <- []byte(strconv.Itoa(requests))

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

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	c := &connection{send: make(chan []byte, 256), ws: ws}
	h.register <- c

	go c.writer()
	c.reader()
}

type connection struct {
	// The websocket connection.
	ws *websocket.Conn
	// Buffered channel of outbound messages.
	send chan []byte
}
func (c *connection) reader() {
	defer func() {
		h.unregister <- c
		c.ws.Close()
	}()
	c.ws.SetReadLimit(512)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
	}
}
// write writes a message with the given message type and payload.
func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}
func (c *connection) writer() {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		h.unregister <- c
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-pingTicker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func main() {
	flag.Parse()

	logger.Println("Server started at " + startTime.String())
	router := mux.NewRouter()
	router.HandleFunc("/", defaultHandler)
	router.HandleFunc("/exit", exitHandler)
	router.HandleFunc("/restart", restartHandler)
	router.HandleFunc("/ws", serveWs)

	go h.run()

    dir := "./public/"
	files, _ := readDir(dir)
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		fileContent, err := ioutil.ReadFile(dir + file.Name())
		if err != nil {
            panic(err)
			fmt.Println("Error on reading file " + file.Name() + "\n")
		}
		var dat map[string]interface{}
		err = json.Unmarshal([]byte(fileContent), &dat)
        if err != nil {
            fmt.Print(fileContent)
            panic(err)
		}

		route, _        := dat["route"].(string)
		methods, result := dat["methods"].(string)
		routes[methods + " "  + route] = route

		if (result == true) {
			methods_list := strings.Split(methods, ",")
			router.HandleFunc(route, func(w http.ResponseWriter, req *http.Request) {
				jsonHandler(w, req, []byte(fileContent))
			}).Methods(methods_list...)
		} else {
			router.HandleFunc(route, func(w http.ResponseWriter, req *http.Request) {
				jsonHandler(w, req, []byte(fileContent))
			})
		}

	}

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))
	http.Handle("/", router)


    fmt.Println("Starting server on address: " + *address)

	err := http.ListenAndServe(*address, nil)
	if err != nil {
		fmt.Println(err.Error())
	}
}

type hub struct {
	// Registered connections.
	connections map[*connection]bool
	// Inbound messages from the connections.
	broadcast chan []byte
	// Register requests from the connections.
	register chan *connection
	// Unregister requests from connections.
	unregister chan *connection
}
func (h *hub) run() {
	for {
		select {
			case c := <-h.register:
				h.connections[c] = true
			case c := <-h.unregister:
				if _, ok := h.connections[c]; ok {
					delete(h.connections, c)
					close(c.send)
				}
			case m := <-h.broadcast:
			for c := range h.connections {
				select {
				case c.send <- m:
				default:
					close(c.send)
					delete(h.connections, c)
				}
			}
		}
	}
}
