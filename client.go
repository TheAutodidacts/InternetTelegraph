package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/stianeikeland/go-rpio"
	"golang.org/x/net/websocket"
)

var (
	keyPinBCM     = 07
	keyPinNumber  = 26
	spkrPinBCM    = 10
	spkrPinNumber = 19
	state         = "idle"
)

type Config struct {
	Room   string
	Server string
	Port   string
}

type socketClient struct {
	ip, port, room string
	conn           *websocket.Conn
}

type morseKey struct {
	lastState, lastDur, lastStart, lastEnd int64
	keyPin                                 rpio.Pin
}

type tone struct {
	state   string
	spkrPin rpio.Pin
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
	}
}

func microseconds() int64 {
	t := time.Now().UnixNano()
	ms := t / int64(time.Microsecond)
	fmt.Println(ms)
	return ms
}

func (sc *socketClient) onMessage(m string, t tone) {

	value := m[:1]
	ts := m[1:]
	fmt.Println("message:")
	fmt.Println(m)
	fmt.Println(value)
	fmt.Println(ts)

	//receiveTs := microseconds()

	fmt.Println("message: ")
	fmt.Println(m)
	fmt.Println("value: ")
	fmt.Println(value)
	fmt.Println("message timestamp: ")
	fmt.Println(ts)

	//ts64, err := strconv.ParseInt(string(ts), 10, 64)
	//if err != nil {
	//	panic(err)
	//}

	// Doesn't work...
	//for (microseconds() - receiveTs) < ts64 {

	if value == "1" {
		fmt.Println("start audio")
		t.start()
	}
	if value == "0" {
		fmt.Println("stop audio")
		t.stop()
	}

}

func (sc *socketClient) listen(t tone) {
	fmt.Println("client listening...")
	var msg string
	for {
		err := websocket.Message.Receive(sc.conn, &msg)
		if err != nil {
			fmt.Println("Couldn’t receive message: " + err.Error())
			break
		} else {
			state = "receiving"
			sc.onMessage(msg, t)
		}
	}
}

func (sc *socketClient) send(data string) {
	err := websocket.Message.Send(sc.conn, data)
	if err != nil {
		fmt.Println("Sending failed:")
		fmt.Println(err.Error())
	}
}

func (t *tone) start() {
	t.spkrPin.Write(rpio.High)
	t.state = "ON"
}

func (t *tone) stop() {
	t.spkrPin.Write(rpio.Low)
	t.state = "OFF"
}

func (t *tone) set(value int) {
	if value == 0 {
		t.spkrPin.Write(rpio.Low)
		t.state = "OFF"

	}
	if value == 1 {
		t.spkrPin.Write(rpio.High)
		t.state = "ON"
	} else {
		fmt.Println("Err! Couldn’t set tone")
	}
}

func (key *morseKey) listen(sc socketClient) {
	//fmt.Println("listeningtokey")
	var lastVal rpio.State = 2
	// This is not ideal. This should be replaced with an edge detect.
	// Waiting on https://github.com/stianeikeland/go-rpio/issues/8.
	for {
		val := key.keyPin.Read()
		if val != lastVal && lastVal != 2 {
			fmt.Println(lastVal)
			key.keyEvent(val, sc)
		}
		lastVal = val
		time.Sleep(200 * time.Microsecond)
	}
}

func (key *morseKey) keyEvent(val rpio.State, sc socketClient) {
	fmt.Println("keyevent")
	fmt.Println(val)
	if state == "idle" {
		state = "sending"
	}

	if val == rpio.Low {
		key.lastEnd = microseconds()
		key.lastDur = key.lastEnd - key.lastStart
		value := strconv.FormatInt(key.lastState, 10)
		timestamp := strconv.FormatInt(key.lastDur, 10)
		msg := value + timestamp
		sc.send(msg)
		key.lastState = 0
		key.lastStart = microseconds()
	}

	if val == rpio.High {
		key.lastEnd = microseconds()
		key.lastDur = key.lastEnd - key.lastStart
		value := strconv.FormatInt(key.lastState, 10)
		timestamp := strconv.FormatInt(key.lastDur, 10)
		msg := value + timestamp
		sc.send(msg)
		key.lastState = 1
		key.lastStart = microseconds()
	}
}

func main() {

	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	config := Config{}
	err := decoder.Decode(&config)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println(config.Room)

	sc := socketClient{ip: config.Server, port: config.Port, room: config.Room} // For internet use
	//sc := socketClient{ip: "192.168.1.134", port: "8000"} // For intranet/LAN use
	//sc := socketClient{ip: "127.0.0.1", port: "8000"} // For testing

	conn, err := websocket.Dial("ws://"+sc.ip+":"+sc.port+"/"+sc.room, "", "http://localhost")
	sc.conn = conn
	checkError(err)

	key := morseKey{lastState: 1, lastDur: 0, lastStart: 0, lastEnd: 0}

	tone := tone{state: "OFF"}

	openErr := rpio.Open()
	checkError(openErr)

	keyPn := rpio.Pin(keyPinBCM)
	keyPn.Input()
	spkrPn := rpio.Pin(spkrPinBCM)
	spkrPn.Output()

	tone.spkrPin = spkrPn

	key.keyPin = keyPn
	defer rpio.Close()

	state = "receiving"

	go key.listen(sc)

	sc.listen(tone)

}
