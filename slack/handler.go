package function

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/slack-go/slack"

	handler "github.com/openfaas/templates-sdk/go-http"

	log "github.com/sirupsen/logrus"
)

type Message struct {
	Title string `json:"title"`
	Body  struct {
		Text string `json:"text"`
	} `json:"body"`
}

// Handle a function invocation
func Handle(req handler.Request) (handler.Response, error) {

	if loglevel, ok := os.LookupEnv("SLACK_LOGLEVEL"); ok {
		switch loglevel {
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		case "fatal":
			log.SetLevel(log.FatalLevel)
		default:
			log.SetLevel(log.InfoLevel)
		}
	}

	var err error

	debugEnv, _ := os.LookupEnv("SLACK_DEBUG")
	debug, err := strconv.ParseBool(debugEnv)
	if err != nil {
		log.Fatal(err)
	}

	var secretBytes []byte
	secretBytes, err = ioutil.ReadFile("/var/openfaas/secrets/slack-api-token")
	if err != nil {
		log.Fatal(err)
	}

	slackAPIToken := strings.TrimSpace(string(secretBytes))
	if len(slackAPIToken) == 0 {
		log.Fatal("Missing Slack API token")
	}

	var slackChannel string
	if value, ok := os.LookupEnv("SLACK_CHANNEL"); ok {
		slackChannel = value
		log.Debug(fmt.Sprintf("Slack channel: '%s'", slackChannel))
	} else {
		log.Fatal("Missing Slack channel")
	}

	api := slack.New(slackAPIToken, slack.OptionDebug(debug))

	var message Message

	err = json.Unmarshal(req.Body, &message)
	if err != nil {
		log.Fatal(err)
	}

	msgTitle := slack.NewTextBlockObject("plain_text", message.Title, false, false)
	msgHeader := slack.NewHeaderBlock(msgTitle, slack.HeaderBlockOptionBlockID("header_block"))
	msgText := slack.NewTextBlockObject("mrkdwn", message.Body.Text, false, false)
	msgSection := slack.NewSectionBlock(msgText, nil, nil)
	msg := slack.MsgOptionBlocks(
		msgHeader,
		msgSection,
	)

	_, _, _, err = api.SendMessage(slackChannel, msg)
	if err != nil {
		return handler.Response{
			Body:       []byte(fmt.Sprintf("Failed to send message to Slack. Error: %s", err.Error())),
			StatusCode: http.StatusInternalServerError,
		}, err
	}

	return handler.Response{
		Body:       []byte(req.Body),
		StatusCode: http.StatusOK,
	}, err
}
