package main

import (
	"github.com/alecthomas/kong"
)

// CLI is the Kong-parsed command tree for plan-ui.
type CLI struct {
	Open     OpenCmd     `cmd:"" help:"Serve a plan artifact and open it in the browser."`
	Poll     PollCmd     `cmd:"" help:"Wait (long-poll) for human feedback."`
	End      EndCmd      `cmd:"" help:"End a session."`
	Stop     StopCmd     `cmd:"" help:"Shut down the background server."`
	Playbook PlaybookCmd `cmd:"" help:"Print the plan authoring playbook."`
	Server   ServerCmd   `cmd:"" hidden:"" help:"Run the HTTP server (used internally)."`
}

// OpenCmd serves a plan file and opens the browser.
type OpenCmd struct {
	File   string `arg:"" type:"path" help:"Plan HTML file to serve."`
	NoOpen bool   `help:"Create the session without launching a browser."`
}

func (c *OpenCmd) Run() error { return runOpen(c.File, c.NoOpen) }

// PollCmd long-polls for human feedback.
type PollCmd struct {
	File       string `arg:"" type:"path" help:"Plan HTML file whose session to poll."`
	AgentReply string `help:"Show this message to the human before waiting again." placeholder:"TEXT"`
	TimeoutMs  int    `help:"Return after N ms if no feedback (0 = wait forever)." placeholder:"N"`
}

func (c *PollCmd) Run() error { return runPoll(c.File, c.AgentReply, c.TimeoutMs) }

// EndCmd ends a session.
type EndCmd struct {
	File string `arg:"" type:"path" help:"Plan HTML file whose session to end."`
}

func (c *EndCmd) Run() error { return runEnd(c.File) }

// StopCmd shuts down the background server.
type StopCmd struct{}

func (c *StopCmd) Run() error { return runStop() }

// PlaybookCmd prints the embedded plan authoring playbook.
type PlaybookCmd struct{}

func (c *PlaybookCmd) Run() error { return runPlaybook() }

// ServerCmd runs the HTTP server in the foreground. Spawned detached by the
// client commands; not intended to be invoked by agents directly.
type ServerCmd struct {
	Port int `help:"Port to bind (0 = OS-assigned)." default:"0"`
}

func (c *ServerCmd) Run() error { return runServer(c.Port) }

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("plan-ui"),
		kong.Description("Collaborate with agents on a plan in a rich browser UI.\n\n"+
			"Agent-facing commands print JSON with a \"next_step\" field describing what to do next."),
		kong.UsageOnError(),
	)
	ctx.FatalIfErrorf(ctx.Run())
}
