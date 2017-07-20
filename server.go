package main

import (
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"

	"golang.org/x/net/websocket"
)

type Client struct {
	id   int
	conn *websocket.Conn
	path interface{}
}

var (
	connections = make(map[*websocket.Conn]Client)
)

func addConnection(ws *websocket.Conn) {
	id := len(connections) + 1

	fmt.Println(ws)
	u := ws.Request().URL
	u.Host = ws.Request().Host
	u.Scheme = "http"
	connections[ws] = Client{id: id, path: u.Path, conn: ws}
	fmt.Println("Connection added!")
}

func removeConnection(conn *websocket.Conn) {
	delete(connections, conn)
	fmt.Println("Connection removed.")
}

func broadcast(msg string, conn *websocket.Conn) {
	for conn := range connections {
		err := websocket.Message.Send(conn, msg)
		if err != nil {
			fmt.Println("Error: ", err.Error())
		} else {
			fmt.Println("Broadcast: " + msg)
		}
	}
}

func broadcastToChannel(msg string, conn *websocket.Conn, channel string) {
	for conn := range connections {
		if conn.Request().URL.Path == channel {
			err := websocket.Message.Send(conn, msg)
			if err != nil {
				fmt.Println("Error: ", err.Error())
			} else {
				fmt.Println("Broadcast: " + msg)
			}
		}
	}
}

// Echo incoming messages to other clients
func Echo(ws *websocket.Conn) {
	addConnection(ws)
	var incoming string
	for {
		receiveErr := websocket.Message.Receive(ws, &incoming)
		if receiveErr != nil {
			if receiveErr == io.EOF {
				removeConnection(ws)
				return
			}
			fmt.Println("Can't receive")
			continue
		} else {
			fmt.Println("Received back from client: " + incoming)
			fmt.Println(ws.Request().URL.Path)
			channel := ws.Request().URL.Path
			incoming = incoming + fmt.Sprintf("%04d\n", connections[ws].id)
			broadcastToChannel(incoming, ws, channel)
		}
	}
}

func main() {
	http.Handle("/channel/", websocket.Handler(Echo))
	// var handlerErr = http.ListenAndServe(os.Getenv("OPENSHIFT_GO_IP")+":"+os.Getenv("OPENSHIFT_GO_PORT"), nil)
	var handlerErr = http.ListenAndServe(":8000", nil)
	checkError(handlerErr)
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
