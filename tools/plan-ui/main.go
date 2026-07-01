package main

import (
	"fmt"
	"os"
)

const usage = `plan-ui — collaborate with agents on a plan in a rich browser UI

Usage:
  plan-ui open <file> [--no-open]          Serve a plan artifact and open it in the browser
  plan-ui poll <file> [options]            Wait (long-poll) for human feedback
  plan-ui end <file>                       End a session
  plan-ui stop                             Shut down the background server
  plan-ui playbook                         Print the plan authoring playbook
  plan-ui server [--port N] [--foreground] Run the server (used internally)

poll options:
  --agent-reply "<text>"   Show a message to the human before waiting again
  --timeout-ms <n>         Return after n ms if no feedback (default: wait forever)

All agent-facing commands print JSON with a "next_step" field describing what to do next.`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "open":
		err = cmdOpen(args)
	case "poll":
		err = cmdPoll(args)
	case "end":
		err = cmdEnd(args)
	case "stop":
		err = cmdStop(args)
	case "playbook":
		err = cmdPlaybook(args)
	case "server":
		err = cmdServer(args)
	case "-h", "--help", "help":
		fmt.Println(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s\n", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
