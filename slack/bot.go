package slack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const TestAuthUrl = "https://slack.com/api/auth.test?token=%s"

type Bot struct {
	UserId      string
	DisplayName string
	commands    []*Command
	ws          *WebSocket
}

func NewBot(token string) (*Bot, error) {
	ws, err := NewWebSocket(token)

	if err != nil {
		return nil, err
	}

	botInfo, err := getBotInfo(token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		UserId:      botInfo.UserId,
		DisplayName: botInfo.User,
		commands:    make([]*Command, 0),
		ws:          ws,
	}

	return bot, nil
}

func (b *Bot) Start() {
	go b.ws.Start()

	for event := range b.ws.ReadChannel() {
		switch v := event.(type) {
		case *IncomingMessage:
			b.dispatch(v)
		}
	}
}

func (b *Bot) Stop() {
	b.ws.Stop()
}

func (b *Bot) Say(channel string, text string) {
	b.ws.WriteChannel() <- NewOutgoingMessage(channel, text)
}

func (b *Bot) Command(pattern string, handler MessageHandler) {
	cmd, err := NewCommand(pattern, handler)

	if err != nil {
		panic(err)
	}

	b.commands = append(b.commands, cmd)
}

func (b *Bot) dispatch(m *IncomingMessage) {
	userId := fmt.Sprintf("<@%s>", b.UserId)

	if !strings.Contains(m.Text, userId) {
		return
	}

	for _, cmd := range b.commands {
		matches := cmd.Matches(m)

		if len(matches) == 0 {
			continue
		}

		go cmd.Handler(b, m, matches)
	}
}

type botInfo struct {
	Ok     bool   `json:"ok"`
	User   string `json:"user"`
	UserId string `json:"user_id"`
	Error  string `json:"error"`
}

func getBotInfo(token string) (*botInfo, error) {
	res, err := http.Get(fmt.Sprintf(TestAuthUrl, token))

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var info botInfo
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}

// MessageHandler is a function that will handle an IncomingMessage
// that is matched by a Command. "matches" contains the array of sub
// matches, if any, from the message text, determined by the pattern
// used to create the Command.
type MessageHandler func(bot *Bot, message *IncomingMessage, matches []string)

type Command struct {
	Pattern *regexp.Regexp
	Handler MessageHandler
}

func NewCommand(pattern string, handler MessageHandler) (*Command, error) {
	regx, err := regexp.Compile(pattern)

	if err != nil {
		return nil, err
	}

	return &Command{regx, handler}, nil
}

func (c *Command) Matches(m *IncomingMessage) []string {
	return c.Pattern.FindStringSubmatch(m.Text)
}
