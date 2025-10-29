package cli

import "context"

type InputMsg struct {
	Input string
}

type QuitMsg struct{}

func Quit(_ context.Context) Msg {
	return QuitMsg{}
}
