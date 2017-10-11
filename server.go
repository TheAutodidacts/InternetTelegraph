package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"golang.org/x/net/websocket"
)

type Client struct {
	id   int
	conn *websocket.Conn
	path interface{}
}

var (
	connections = make(map[*websocket.Conn]Client)
	idCounter   = 0
)

func addConnection(ws *websocket.Conn) {
	var id int
	if len(connections) == 0 {
		idCounter = 0
	}
	if len(connections)+1 > idCounter {
		idCounter = len(connections) + 1
		id = len(connections) + 1
	} else {
		idCounter = idCounter + 1
		id = idCounter
	}

	fmt.Println(ws)
	u := ws.Request().URL
	u.Host = ws.Request().Host
	u.Scheme = "http"
	connections[ws] = Client{id: id, path: u.Path, conn: ws}
	fmt.Println("Connection added!")
}

func removeConnection(conn *websocket.Conn) {
	fmt.Print("Removing ")
	fmt.Print(conn)
	fmt.Print(" from ")
	fmt.Println(connections)

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
			msgSenderStr := msg[len(msg)-4:]
			msgSender, _ := strconv.Atoi(msgSenderStr)
			if msgSender == connections[conn].id && msg[len(msg)-6:len(msg)-4] == "v2" {
				fmt.Print("Suppressing broadcast of ")
				fmt.Print(msg)
				fmt.Print(" to client #")
				fmt.Println(fmt.Sprintf("%04d", connections[conn].id))
			} else {
				// It’s from a different telegraph, or the sender is a v1 client,
				// and needs it echoed for backward compatability.
				outMsg := msg[:len(msg)-6] + msgSenderStr
				err := websocket.Message.Send(conn, outMsg)
				if err != nil {
					fmt.Println("Error: ", err.Error())
				} else {
					fmt.Println("Broadcast: " + outMsg)
				}
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
		} else if(incoming == "ping"){ // Reply to client pings

			err := websocket.Message.Send(ws, "pong")
			if err != nil {
				fmt.Println("Error: ", err.Error())
			} else {
				fmt.Println("Pong sent")
			}

		} else {
			fmt.Println("Received from client: " + incoming)
			fmt.Println(ws.Request().URL.Path)
			channel := ws.Request().URL.Path
			incoming = incoming + fmt.Sprintf("%04d", connections[ws].id)
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
