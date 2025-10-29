package main

type InputMsg struct {
	Input string
}

type QuitMsg struct{}

func Quit() Msg {
	return QuitMsg{}
}
