package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// SlashCommand represents the incoming Slash Command payload from Slack
type SlashCommand struct {
	Token               string `json:"token"`
	TeamID              string `json:"team_id"`
	TeamDomain          string `json:"team_domain"`
	ChannelID           string `json:"channel_id"`
	ChannelName         string `json:"channel_name"`
	UserID              string `json:"user_id"`
	UserName            string `json:"user_name"`
	Command             string `json:"command"`
	Text                string `json:"text"`
	APIAppID            string `json:"api_app_id"`
	IsEnterpriseInstall string `json:"is_enterprise_install"`
	ResponseURL         string `json:"response_url"`
	TriggerID           string `json:"trigger_id"`
}

func parseSlashCommand(values url.Values) *SlashCommand {
	return &SlashCommand{
		Token:       values.Get("token"),
		TeamID:      values.Get("team_id"),
		TeamDomain:  values.Get("team_domain"),
		ChannelID:   values.Get("channel_id"),
		ChannelName: values.Get("channel_name"),
		UserID:      values.Get("user_id"),
		UserName:    values.Get("user_name"),
		Command:     values.Get("command"),
		Text:        values.Get("text"),
		APIAppID:    values.Get("api_app_id"),
		ResponseURL: values.Get("response_url"),
		TriggerID:   values.Get("trigger_id"),
	}
}

type SlackMessage struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func sendToSlack(channelID, message string) error {
	secrets, err := getSecrets("prod/GoDSOTPBot")
	if err != nil {
		return err
	}

	slackBotToken := secrets.SlackBotToken
	fmt.Println("Creating Slack message")
	slackMessage := SlackMessage{
		Channel: channelID,
		Text:    message,
	}
	fmt.Println("Message created is: " + slackMessage.Channel + " " + slackMessage.Text)

	slackMessageJSON, err := json.Marshal(slackMessage)
	if err != nil {
		return err
	}

	fmt.Println("Attempting to send slack message")

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(slackMessageJSON)) //+"/"+channelID, bytes.NewBuffer(slackMessageJSON))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", "Bearer "+slackBotToken)
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	fmt.Println("Request sent to Slack API successfully")

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API request failed with status: %d", resp.StatusCode)
	}

	return nil
}

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	payload, err := url.ParseQuery(request.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Failed to parse request",
		}, nil
	}

	slashCommand := parseSlashCommand(payload)

	secrets, err := getSecrets("prod/GoDSOTPBot")
	if err != nil {
		fmt.Println(secrets.GoBotVerificationToken + " " + secrets.SlackBotToken)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Bad Request",
		}, nil
	}

	// Check if the request is valid (e.g., token verification)
	if slashCommand.Token != secrets.GoBotVerificationToken {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Body:       "Unauthorized",
		}, nil
	}

	// Process the slash command
	responseMessage := fmt.Sprintf("Received command: %s with text: %s", slashCommand.Command, slashCommand.Text)
	if slashCommand.Command == "/draftorder" {
		message := "1. Bill\n2. Eric\n3. CP\n4. Tony\n5. Scolson\n6. Dan\n7. Hagel\n8. Andrew\n9. TL\n10. Jared"
		responseMessage = fmt.Sprintf(
			message,
		)
		sendToSlack(slashCommand.ChannelID, message)
	} else if slashCommand.Command == "/punishment" {
		message := "1. First Place names the Last Place Team\n2. License Plate holder must be displayed from draft to end of regular season\n3. Last place purchases the first round of drinks at the draft"
		responseMessage = fmt.Sprintf(message)
		sendToSlack(slashCommand.ChannelID, message)
	} else if slashCommand.Command == "/draftinfo" {
		message := "Draft is usually Labor Day weekend.  Please provide me with any conflicts as soon as possible so we can schedule around it. In person drafting will depend on a variety of things."
		responseMessage = fmt.Sprintf(message)
		sendToSlack(slashCommand.ChannelID, message)
	} else if slashCommand.Command == "/champion" {
		message := "Jared is your champion"
		responseMessage = fmt.Sprintf(message)
		sendToSlack(slashCommand.ChannelID, message)
	} else if slashCommand.Command == "/remotedraft" {
		message := "If unable to attend in person please go here: https://sleeper.com/draft/nfl/1051530318366318592?ftue=commish"
		responseMessage = fmt.Sprintf(message)
		sendToSlack(slashCommand.ChannelID, message)
	} else {
		responseMessage = fmt.Sprintf("Something went wrong, command %s is not supported", slashCommand.Command)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       responseMessage,
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
