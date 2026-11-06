package types

// EnvProvider generates private runtime values from a script's JSON object.
type EnvProvider struct {
	Command   string `json:"command"`
	Directory string `json:"directory"`
	Timeout   int    `json:"timeout"`
}
