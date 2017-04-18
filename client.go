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
	stopKeyListener = make(chan bool)
	startListeners = make(chan bool)
)

// var stopPWM = make(chan bool)

type Config struct {
	Channel string
	Server  string
	Port    string
}

type socketClient struct {
	ip, port, channel, status string
	conn                      *websocket.Conn
}

type morseKey struct {
	lastState, lastDur, lastStart, lastEnd int64
	keyPin                                 rpio.Pin
}

type tone struct {
	state   string
	spkrPin rpio.Pin
}

func (sc *socketClient) dial(firstDial bool, t tone) {
	sc.status = "dialling"
	// start := time.Now()
	retryEnd := time.Now().Add(time.Duration(1800) * time.Second) // try to reconnect for thirty minutes before completely giving up

	for i := 1; i > 0; i++ {
		fmt.Print("Dialling websocket (try #")
		fmt.Print(i)
		fmt.Print(")\n")
		conn, err := websocket.Dial("ws://"+sc.ip+":"+sc.port+"/channel/"+sc.channel, "", "http://localhost")
		if err == nil {
			sc.conn = conn
			playMorse(".-. . .- -.. -.--", t)
			sc.status = "connected"
			fmt.Println("sc.status = "+sc.status)
			fmt.Print("sc.conn = ")
			fmt.Println(sc.conn)
			if firstDial != true {
				// time to restart the listener goroutines with the new websocket
				fmt.Println("Sending message to stopKeyListener channel.")
			  stopKeyListener <- true
				fmt.Println("Sending message to startListeners channel.")
				startListeners <- true
				}
			return
		}
		if err != nil {
			fmt.Println("Error connecting to 'ws://" + sc.ip + ":" + sc.port + "/channel/" + sc.channel + "': " + err.Error())
			// connect to server; retry until the retry period has expired
			if time.Now().After(retryEnd) {
				fmt.Println("Timed out; websocket reconnection failed too many times.")
				sc.status = "disconnected"
				return
			}
			time.Sleep(time.Duration(i) * time.Second)
			continue
		}
	}
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

func (t *tone) start() {
	t.spkrPin.Write(rpio.High)
	t.state = "ON"

	// This makes a very bad tone with a piezo buzzer.
	// go func() {
	// 	for {
	// 		select {
	// 		case <-stopPWM:
	// 			return
	// 		default:
	// 			t.spkrPin.Write(rpio.High)
	// 			time.Sleep(250 * time.Microsecond)
	// 			t.spkrPin.Write(rpio.Low)
	// 			time.Sleep(250 * time.Microsecond)
	// 		}
	// 	}
	// }()
}

func (t *tone) stop() {
	t.spkrPin.Write(rpio.Low)
	// stopPWM <- true
	t.state = "OFF"
}

func playMorse(message string, t tone) {
	speed := time.Duration(50)
	for i := 0; i < len(message); i++ {
		switch message[i] {
		case 46: // == "."
			t.start()
			time.Sleep(speed * time.Millisecond)
			t.stop()
			time.Sleep(speed * time.Millisecond)
		case 45: // == "-"
			t.start()
			time.Sleep(3 * speed * time.Millisecond)
			t.stop()
			time.Sleep(speed * time.Millisecond)
		case 32: // == "-"
			time.Sleep(3 * speed * time.Millisecond)
		default:
			// Do nothing...
		}
	}
}

func microseconds() int64 {
	t := time.Now().UnixNano()
	ms := t / int64(time.Microsecond)
	return ms
}

func (sc *socketClient) onMessage(m string, t tone) {

	value := m[:1]
	ts := m[1:]

	fmt.Print("message:   ")
	fmt.Println(m)
	fmt.Print("key value: ")
	fmt.Println(value)
	fmt.Print("timestamp: ")
	fmt.Println(ts)
	fmt.Println()

	if value == "1" {
		// fmt.Println("start audio")
		t.start()
	}
	if value == "0" {
		// fmt.Println("stop audio")
		t.stop()
	}

}

func (sc *socketClient) listen(t tone) {
	fmt.Println("Client listening…")
	var msg string
	for {
		err := websocket.Message.Receive(sc.conn, &msg)
		if err != nil {
			sc.status = "disconnected"
			fmt.Println("Couldn’t receive message: " + err.Error())

			if sc.status == "dialling" {
				fmt.Println("Currently redialling websocket server.")
			} else {
				fmt.Println("Attempting to reconnect to websocket server in 10 seconds…")
				time.Sleep(10 * time.Second)
				sc.status = "dialling"
				go sc.dial(false, t)
			}

			time.Sleep(10 * time.Second)

		} else {
			state = "receiving"
			sc.onMessage(msg, t)
		}
	}
}

func (sc *socketClient) send(data string, t tone) {
	err := websocket.Message.Send(sc.conn, data)
	if err != nil {
		sc.status = "disconnected"
		fmt.Print("sc.conn in send function = ")
		fmt.Println(sc.conn)
		fmt.Println("Could not send message:")
		fmt.Println(err.Error())
		fmt.Println("Please double check your internet connection and telegraph configuration.")
		if data[:1] != "1" { // Error beep on only on keyup, to prevent confusion.
			playMorse("........", t)
			if sc.status == "dialling" || sc.status == "connected" {
				fmt.Println("Currently redialling websocket server.")
				fmt.Println("Current status: " + sc.status)
			} else {
				fmt.Println("Redialling websocket server in 10 seconds…")
				fmt.Println("Current status: " + sc.status)
				go sc.dial(false, t)
			}
		}
	}
	return
}

func (key *morseKey) listen(sc socketClient, t tone) {
	var lastVal rpio.State = 2
	// This is not ideal. This should be replaced with an edge detect.
	// Waiting on https://github.com/stianeikeland/go-rpio/issues/8.
	for {
		val := key.keyPin.Read()
		if val != lastVal && lastVal != 2 {
			fmt.Print("keyEvent: ")
			fmt.Print(lastVal)
			fmt.Print(" → ")
			fmt.Println(val)
			key.keyEvent(val, sc, t)
		}
		lastVal = val
		select {
		case msg := <-stopKeyListener:
				if msg == true {
					fmt.Println("Websocket has been re-dialled! Stopping key listen goroutine.")
					msg = false
					return
				}
			default:
				continue
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func (key *morseKey) keyEvent(val rpio.State, sc socketClient, t tone) {
	if state == "idle" {
		state = "sending"
	}

	if val == rpio.Low {
		key.lastEnd = microseconds()
		key.lastDur = key.lastEnd - key.lastStart
		value := strconv.FormatInt(key.lastState, 10)
		timestamp := strconv.FormatInt(key.lastDur, 10)
		msg := value + timestamp
		go sc.send(msg, t)
		key.lastState = 0
		key.lastStart = microseconds()
	}

	if val == rpio.High {
		key.lastEnd = microseconds()
		key.lastDur = key.lastEnd - key.lastStart
		value := strconv.FormatInt(key.lastState, 10)
		timestamp := strconv.FormatInt(key.lastDur, 10)
		msg := value + timestamp
		go sc.send(msg, t)
		key.lastState = 1
		key.lastStart = microseconds()
	}

}

func main() {
	//for {
	if len(os.Getenv("TELEGRAPH_CONFIG_PATH")) == 0 {
		os.Setenv("TELEGRAPH_CONFIG_PATH", "config.json")
	}

	file, _ := os.Open(os.Getenv("TELEGRAPH_CONFIG_PATH"))
	decoder := json.NewDecoder(file)
	config := Config{}
	err := decoder.Decode(&config)
	if err != nil {
		fmt.Println("Error reading config.json: ", err)
		fmt.Println("Falling back on default config…")
		config.Channel = "lobby"
		config.Server = "morse.autodidacts.io"
		config.Port = "8000"
	}
	fmt.Println(config.Channel)

	// Initialize morse key
	key := morseKey{lastState: 1, lastDur: 0, lastStart: 0, lastEnd: 0}

	// Init tone
	tone := tone{state: "OFF"}

	// Setup GPIO
	openErr := rpio.Open()
	if openErr != nil {
		fmt.Println("Error initializing GPIO: " + err.Error())
	}
	keyPn := rpio.Pin(keyPinBCM)
	keyPn.Input()
	spkrPn := rpio.Pin(spkrPinBCM)
	spkrPn.Output()

	tone.spkrPin = spkrPn
	key.keyPin = keyPn

	defer rpio.Close()

	// Init socketClient & dial websocket
	sc := socketClient{ip: config.Server, port: config.Port, channel: config.Channel}

	if sc.status != "connected" && sc.status != "dialling" {
		sc.dial(true, tone) // Connect!
	}

	state = "receiving"

	go sc.listen(tone)

	go key.listen(sc, tone)

	for {
		select {
		case msg := <-startListeners:
			if msg == true {
				fmt.Println("New websocket connection! Starting new key listener goroutine.")
				go sc.listen(tone)
				go key.listen(sc, tone)
				msg = false
			}
	  }
	}
}
