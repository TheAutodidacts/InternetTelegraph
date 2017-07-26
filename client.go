package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	term "github.com/nsf/termbox-go"
	"github.com/stianeikeland/go-rpio"
	"golang.org/x/net/websocket"
)

var (
	keyPinBCM           = 07
	keyPinNumber        = 26
	spkrPinBCM          = 10
	spkrPinNumber       = 19
	state               = "idle"
	queue               []string
	outQueue            []string
	bufferReferenceTime int64
	bufferDelay         int64  = 500000 // what should this be?
	lastKeyId           string          // identifier for the telegraph that the current queue came from
	lastKeyVal          string
	t                   tone
)

// var stopPWM = make(chan bool)

type Config struct {
	Channel string
	Server  string
	Port    string
	Gpio    bool
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

func (sc *socketClient) dial(firstDial bool) {
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
			playMorse(".-. . .- -.. -.--")
			sc.status = "connected"
			fmt.Println("sc.status = " + sc.status)
			fmt.Print("sc.conn = ")
			fmt.Println(sc.conn)
			if firstDial != true {
				// time to restart the listener goroutines with the new websocket
				// fmt.Println("Sending message to stopKeyListener channel.")
				// stopKeyListener <- true
				// fmt.Println("Sending message to startListeners channel.")
				// startListeners <- true
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
	if gpio == true {
		if value == 0 {
			t.spkrPin.Write(rpio.Low)
			t.state = "OFF"

		} else if value == 1 {
			t.spkrPin.Write(rpio.High)
			t.state = "ON"
		} else {
			fmt.Print("Err! Couldn’t set tone to: ")
			fmt.Println(value)
		}
	} else {
		if value == 0 {
			// need to figure out the best way to generate + play a tone
			t.state = "OFF"
		} else if value == 1 {
			// need to figure out the best way to generate + play a tone
			t.state = "ON"
		} else {
			fmt.Println("Err! Couldn’t set tone")
		}
	}
}

func (t *tone) start() {
	if gpio == true {
		t.spkrPin.Write(rpio.High)
	} else {
		// TODO: cross-platform generate and play tone
	}
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
	if gpio == true {
		t.spkrPin.Write(rpio.Low)
	} else {
		// TODO: cross-platform generate and play tone
	}
	// stopPWM <- true
	t.state = "OFF"
}

func playMorse(message string) {
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
		case 32: // == " "
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

func (sc *socketClient) onMessage(m string) {
	// value := m[:1]            // whether the key went up or down
	ts := m[1 : len(m)-4] // timestamp (in microseconds)
	keyId := m[len(m)-4:] // last 4 digits of message is the key id

	if keyId != lastKeyId { // if its a different telegraph sending
		if len(queue) > 0 {
			// ...and there's already a queue from a different telegraph, do nothing.
		} else {
			fmt.Print("New telegraph detected. Setting bufferReferenceTime:")
			lastKeyId = keyId
			tsint64, err := strconv.ParseInt(ts, 10, 64)
			if err != nil {
				fmt.Println(err)
			}
			// Set the time offset between local and remote clients, plus bufferDelay
			bufferReferenceTime = (microseconds() + bufferDelay) - tsint64

			fmt.Println(bufferReferenceTime)

			queue = append(queue, m)
		}

		fmt.Print("Received message ")
		fmt.Print(m)
		fmt.Print(" from ")
		fmt.Print(keyId)
		fmt.Print(" at ")
		fmt.Println(time.Now())
	} else {
		queue = append(queue, m)
	}
}

func (sc *socketClient) listen() {
	fmt.Println("Client listening…")
	var msg string
	for {
		err := websocket.Message.Receive(sc.conn, &msg)
		if err != nil {
			fmt.Println("Websocket error on Message.Receive(): " + err.Error())
			sc.status = "disconnected"
			sc.dial(false)

		} else {
			sc.onMessage(msg)
		}
	}
}

func (sc *socketClient) outputListen() {
	for {
		if len(outQueue) > 0 {

			fmt.Println("Out queue detected in outputListen()")

			fmt.Println("Sending message: " + outQueue[0])
			sendErr := websocket.Message.Send(sc.conn, outQueue[0])
			if sendErr != nil {
				sc.status = "disconnected"
				fmt.Print("sc.conn in send function = ")
				fmt.Println(sc.conn)
				fmt.Println("Could not send message:")
				fmt.Println(sendErr.Error())
				if outQueue[0][:1] != "1" { // Error beep only on keyup, to prevent confusion.
					playMorse("........")
					fmt.Println("Redialling websocket server…")
					fmt.Println("Current status: " + sc.status)
					sc.dial(false)
				}
			} else {
				fmt.Print("Sent: ")
				fmt.Println(outQueue[0])
				outQueue = append(outQueue[:0], outQueue[0+1:]...) //
			}
		}
	}
}

func main() {
	if len(os.Getenv("TELEGRAPH_CONFIG_PATH")) == 0 {
		os.Setenv("TELEGRAPH_CONFIG_PATH", "config.json")
	}

	file, _ := os.Open(os.Getenv("TELEGRAPH_CONFIG_PATH"))
	decoder := json.NewDecoder(file)
	config := Config{Gpio: true}
	err := decoder.Decode(&config)
	if err != nil {
		fmt.Println("Error reading config.json: ", err)
		fmt.Println("Falling back on default config…")
		config.Channel = "lobby"
		config.Server = "morse.autodidacts.io"
		config.Port = "8000"
	}
	fmt.Println(config.Channel)

	gpio = config.Gpio

	// Initialize morse key
	key := morseKey{lastState: 1, lastDur: 0, lastStart: 0, lastEnd: 0}

	t = tone{state: "OFF"}

	if gpio == true {
		// Setup GPIO
		openErr := rpio.Open()
		if openErr != nil {
			fmt.Println("Error initializing GPIO: " + err.Error())
		}
		keyPn := rpio.Pin(keyPinBCM)
		keyPn.Input()
		spkrPn := rpio.Pin(spkrPinBCM)
		spkrPn.Output()
		t.spkrPin = spkrPn
		key.keyPin = keyPn

		defer rpio.Close()

	} else {
		// Setup for keypress detection
		err := term.Init()
		if err != nil {
			panic(err)
		}

		defer term.Close()
	}

	// Init socketClient & dial websocket
	sc := socketClient{ip: config.Server, port: config.Port, channel: config.Channel}

	sc.dial(true)

	// Start the listener for incoming messages
	go sc.listen()

	// Start the listener that monitors the output queue and sends messages
	go sc.outputListen()

	keyVal := rpio.Low

	fmt.Println(keyVal)

	// Adding a simplified version of things...
	for {
		if sc.status != "connected" && sc.status != "dialling" {
			sc.dial(true) // Connect if broken
		}
		var val string
		if gpio == true {
			keyVal = key.keyPin.Read()
		} else {

		keyPressLoop:
			for {
				switch ev := term.PollEvent(); ev.Type {
				case term.EventKey:
					switch ev.Key {
					case term.KeyEsc:
						os.Exit(1)
					case term.KeySpace:
						keyVal = rpio.High
						break keyPressLoop
					case term.KeyEnter:
						keyVal = rpio.Low
						break keyPressLoop
					default:
						break keyPressLoop
					}
				case term.EventError:
					panic(ev.Err)
				}
			}
		}

		if keyVal == rpio.High {
			val = "1"
		} else {
			val = "0"
		}

		if keyVal != lastKeyValue {
			fmt.Print("key change: ")
			fmt.Print(lastKeyValue)
			fmt.Print(" → ")
			fmt.Println(keyVal)
			toneVal, _ := strconv.Atoi(val)
			t.set(toneVal)
			timestamp := strconv.FormatInt(microseconds(), 10)
			msg := val + timestamp
			outQueue = append(outQueue, msg)
			//go sc.send(msg)
			lastKeyValue = keyVal
		}

		if len(queue) > 0 {
			m := queue[0]

			if err != nil {
				fmt.Println(err)
			}
			ts := m[1 : len(queue[0])-4]
			ts64, _ := strconv.ParseInt(ts, 10, 64)

			if ts64 > bufferReferenceTime+microseconds() {
				msgValue, _ := strconv.Atoi(m[:1])

				queue = append(queue[:0], queue[0+1:]...) // pop message out of queue
				t.set(msgValue)
			}
		}
	}
}
