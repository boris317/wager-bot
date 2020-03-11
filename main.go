package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/boris317/wager-bot/slack"
)

const TOKEN = "*****"

type JSONStringer struct{}

func (j *JSONStringer) String() string {
	b, err := json.Marshal(j)
	if err != nil {
		return ""
	}
	return string(b)
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	bot, err := slack.NewBot(TOKEN)

	if err != nil {
		log.Fatalln(err)
	}

	bot.Command(`echo(.*)`, func(b *slack.Bot, m *slack.IncomingMessage, matches []string) {
		b.Say(m.Channel, matches[1])
	})

	bot.Command(`answer to life`, func(b *slack.Bot, m *slack.IncomingMessage, _ []string) {
		b.Say(m.Channel, "42")
	})

	bot.Command(`stock ([a-z]+)\s?`, func(b *slack.Bot, m *slack.IncomingMessage, matches []string) {
		reply := fmt.Sprintf("Oh boy! I hope you didn't have your life savings in %s.", matches[1])
		b.Say(m.Channel, reply)
	})

	go bot.Start()
	defer bot.Stop()

	bot.Say("DDYNV59BP", "Hello!")

	<-interrupt
	return
}
