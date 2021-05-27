package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/interpreter"
	"go.uber.org/fx"
)

const (
	applicationName = "eventsGenerator"
)

type Config struct {
	MessageContents []Message
	FilePath        string
}

type Message struct {
	Event           interpreter.Event
	Payload         map[string]string
	BootTimeOffset  time.Duration
	BirthdateOffset time.Duration
}

func generateEvents(config Config) []interpreter.Event {
	now := time.Now()
	var events []interpreter.Event
	for i, msg := range config.MessageContents {
		if len(msg.Event.TransactionUUID) == 0 {
			msg.Event.TransactionUUID = strconv.Itoa(i)
		}
		events = append(events, createEvent(now, msg))
	}

	sort.Slice(events, func(a int, b int) bool {
		return events[a].Birthdate > events[b].Birthdate
	})
	return events
}

func createEvent(current time.Time, msg Message) interpreter.Event {
	event := msg.Event
	event.MsgType = 4
	event.Metadata = make(map[string]string)

	for k, v := range msg.Event.Metadata {
		event.Metadata[k] = v
	}

	payload := make(map[string]string)
	for k, v := range msg.Payload {
		payload[k] = v
	}

	event.Metadata["/boot-time"] = fmt.Sprint(current.Add(msg.BootTimeOffset).Unix())
	birthdate := current.Add(msg.BirthdateOffset)
	payload["ts"] = current.Add(msg.BirthdateOffset).Format(time.RFC3339Nano)
	if j, err := json.Marshal(payload); err == nil {
		event.Payload = string(j)
	} else {
		event.Payload = fmt.Sprintf(`{"ts":"%s"}`, birthdate)
	}
	event.Birthdate = birthdate.UnixNano()
	return event
}

func writeEvents(filePath string, events []interpreter.Event) {
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write events to file: %v", err)
		return
	}
	defer file.Close()
	if a, err := json.Marshal(events); err == nil {
		file.WriteString(string(a))
	}
}

func main() {
	v := viper.New()
	v.AddConfigPath(".")
	v.SetConfigName(applicationName)
	err := v.ReadInConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read in viper config: %v\n", err.Error())
		os.Exit(1)
	}

	app := fx.New(
		arrange.ForViper(v),
		arrange.Provide(Config{}),
		fx.Provide(
			generateEvents,
		),
		fx.Invoke(
			func(config Config, events []interpreter.Event) {
				var filePath string
				if len(os.Args) > 1 {
					filePath = os.Args[1]
				} else {
					filePath = config.FilePath
				}

				writeEvents(filePath, events)
				os.Exit(0)
			},
		),
	)

	if err := app.Err(); err == nil {
		app.Run()
	} else if errors.Is(err, pflag.ErrHelp) {
		return
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
