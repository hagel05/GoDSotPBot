package slack

import "net/url"

// SlashCommand represents the subset of Slack's slash-command payload the bot
// uses. Slack sends this as application/x-www-form-urlencoded.
type SlashCommand struct {
	Command     string
	Text        string
	UserName    string
	ChannelID   string
	ChannelName string
	ResponseURL string
	TeamID      string
}

// ParseSlashCommand decodes a form-encoded slash-command body.
func ParseSlashCommand(values url.Values) SlashCommand {
	return SlashCommand{
		Command:     values.Get("command"),
		Text:        values.Get("text"),
		UserName:    values.Get("user_name"),
		ChannelID:   values.Get("channel_id"),
		ChannelName: values.Get("channel_name"),
		ResponseURL: values.Get("response_url"),
		TeamID:      values.Get("team_id"),
	}
}

// Response types control who sees a slash-command reply.
const (
	// ResponseInChannel makes the reply visible to everyone in the channel.
	ResponseInChannel = "in_channel"
	// ResponseEphemeral makes the reply visible only to the invoking user.
	ResponseEphemeral = "ephemeral"
)

// Response is the JSON body returned directly to a slash command.
type Response struct {
	ResponseType string `json:"response_type"`
	Text         string `json:"text"`
}

// InChannel builds a channel-visible response.
func InChannel(text string) Response {
	return Response{ResponseType: ResponseInChannel, Text: text}
}

// Ephemeral builds a response visible only to the invoking user.
func Ephemeral(text string) Response {
	return Response{ResponseType: ResponseEphemeral, Text: text}
}
