package options

// Opts opts.
type Opts struct {
	Signal        *Signal
	IOSPushSound  string
	IOSBadgeCount bool
	Ex            string
}

// Signal message id.
type Signal struct {
	ClientMsgID string
}

type PushPayload struct {
	SessionType int    `json:"sessionType"`
	SourceID    string `json:"sourceId"`
}

type PushEx struct {
	Badge             string      `json:"badge"`
	Sound             string      `json:"sound"`
	ForceNotification bool        `json:"force_notification"`
	OpenUrl           string      `json:"open_url"`
	Payload           PushPayload `json:"payload"`
}
