package cli

import "context"

type InputMsg struct {
	Input string
}

type QuitMsg struct{}

func Quit(_ context.Context) Msg {
	return QuitMsg{}
}

type BatchMsg struct {
	cmds []Cmd
}

func BatchCmd(cmds ...Cmd) Cmd {
	return func(context.Context) Msg {
		return BatchMsg{
			cmds: cmds,
		}
	}
}
