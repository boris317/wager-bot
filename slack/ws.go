package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

const HandshakeUrl = "https://slack.com/api/rtm.connect?token=%s"

type WebSocket struct {
	in        chan interface{}
	out       chan IDer
	conn      *websocket.Conn
	isStopped bool
}

// NewWebSocket returns a WebSocket instance
func NewWebSocket(token string) (*WebSocket, error) {
	hs, err := doConnectHandshake(token)

	if err != nil {
		return nil, err
	}

	conn, _, err := websocket.DefaultDialer.Dial(hs.Url, nil)

	if err != nil {
		return nil, err
	}

	ws := &WebSocket{
		in:   make(chan interface{}, 256),
		out:  make(chan IDer, 256),
		conn: conn,
	}

	return ws, nil
}

// ReadChannel returns a read only channel of slack events
func (ws *WebSocket) ReadChannel() <-chan interface{} {
	return ws.in
}

// WriteChannel returns a channel used to send slack messages
func (ws *WebSocket) WriteChannel() chan<- IDer {
	return ws.out
}

// Start the web socket session and blocks. It starts two goroutines that act as a
// read pump and write pump
func (ws *WebSocket) Start() {
	defer ws.Stop()

	var wg sync.WaitGroup
	wg.Add(2)

	go ws.readPump(&wg)
	go ws.writePump(&wg)

	wg.Wait()
	return
}

// readPump reads messages from the web socket connection and sends the unmarshaled
// event to the in channel. When the the underlying web socket connection is closed
// a ReadMessage error will cause the pump to stop.
func (ws *WebSocket) readPump(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		_, data, err := ws.conn.ReadMessage()
		if err != nil {
			log.Println("read error: ", err)
			return
		}

		event, err := unmarshalEvent(data)

		if err != nil {
			return
		}

		ws.in <- event
	}
}

// writePump reads messages from the out channel and writes them to the
// web socket connection. Closing the in channel will stop this pump.
func (ws *WebSocket) writePump(wg *sync.WaitGroup) {
	ticker := time.NewTicker(1 * time.Minute)

	defer func() {
		ticker.Stop()
		wg.Done()
	}()

	messageId := uint64(0)

	for {
		select {
		case v, ok := <-ws.out:
			if !ok {
				return
			}

			messageId++
			v.SetId(messageId)

			if err := ws.conn.WriteJSON(v); err != nil {
				return
			}
		case <-ticker.C:
			ws.out <- NewPing()
		}
	}
}

// Stop shuts down the web socket connection with slack and closes ReadChannel and WriteChannel
func (ws *WebSocket) Stop() error {
	if ws.isStopped {
		return nil
	}

	defer ws.conn.Close()

	ws.isStopped = true
	close(ws.out)

	err := ws.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	time.Sleep(time.Second)
	close(ws.in)

	return err
}

type handshake struct {
	Ok  bool
	Url string
}

func doConnectHandshake(token string) (*handshake, error) {
	res, err := http.Get(fmt.Sprintf(HandshakeUrl, token))

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errorFromResponse(res)
	}

	var hs handshake
	if err := json.NewDecoder(res.Body).Decode(&hs); err != nil {
		return nil, err
	}

	return &hs, nil
}

func errorFromResponse(res *http.Response) error {
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)

	return errors.New(buf.String())
}
