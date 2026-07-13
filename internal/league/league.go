// Package league defines the league data model and the Provider seam that
// command handlers read from.
//
// Today the only implementation is StaticProvider, which reads a mounted YAML
// file. Future implementations (YahooProvider, SleeperProvider) can satisfy the
// same Provider interface to serve live data without changing command wiring.
package league

// League is the annually-updated data that drives the bot's responses. It maps
// 1:1 to the mounted league.yaml file so it can be edited without a code change.
type League struct {
	Season      string   `yaml:"season"`
	Draft       Draft    `yaml:"draft"`
	DraftOrder  []string `yaml:"draft_order"`
	Punishments []string `yaml:"punishments"`
}

// Draft describes when and where the draft happens.
type Draft struct {
	Date      string `yaml:"date"`
	Time      string `yaml:"time"`
	Location  string `yaml:"location"`
	RemoteURL string `yaml:"remote_url"`
}
