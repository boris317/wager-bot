package slack

import (
	"encoding/json"
	"log"
)

type IDer interface {
	SetId(id uint64)
}

type Error struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type Ping struct {
	Id   uint64 `json:"id"`
	Type string `json:"type"`
}

func (p *Ping) SetId(id uint64) {
	p.Id = id
}

func NewPing() *Ping {
	return &Ping{Type: "ping"}
}

type OutgoingMessage struct {
	Id      uint64 `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func (m *OutgoingMessage) SetId(id uint64) {
	m.Id = id
}

func NewOutgoingMessage(channel string, text string) *OutgoingMessage {
	return &OutgoingMessage{
		Type:    "message",
		Channel: channel,
		Text:    text,
	}
}

type OutgoingMessageReply struct {
	Ok        bool   `json:"ok"`
	ReplyTo   uint64 `json:"reply_to"`
	Timestamp string `json:"ts"`
	Text      string `json:"text"`
	Error     *Error `json:"error"`
}

type IncomingMessage struct {
	User      string `json:"user"`
	Channel   string `json:"channel"`
	Text      string `json:"text"`
	Timestamp string `json:"ts"`
}

type EventType struct {
	Type    string `json:"type"`
	ReplyTo int64  `json:"reply_to"`
}

func (e *EventType) GetType() string {
	if e.Type != "" {
		return e.Type
	}
	if e.ReplyTo > 0 {
		return "reply"
	}
	return ""
}

func unmarshalEvent(data []byte) (interface{}, error) {
	var event EventType

	if _, err := jsonUnmarshal(data, &event); err != nil {
		return nil, err
	}

	switch event.GetType() {
	case "message":
		return jsonUnmarshal(data, &IncomingMessage{})
	case "reply":
		return jsonUnmarshal(data, &OutgoingMessageReply{})
	default:
		return string(data), nil
	}
}

func jsonUnmarshal(data []byte, v interface{}) (interface{}, error) {
	if err := json.Unmarshal(data, v); err != nil {
		log.Println("unmarshal error:", err)
		return nil, err
	}
	return v, nil
}
