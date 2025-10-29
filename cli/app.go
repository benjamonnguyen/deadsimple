package main

import (
	"bufio"
	"context"
	"os"
	"os/signal"
	"syscall"
)

type App struct {
	m      Model
	msgs   chan Msg
	cmds   chan Cmd
	errs   chan error
	ctx    context.Context
	cancel context.CancelFunc
}

type Model interface {
	Update(Msg) (Model, Cmd)
}

type Msg any

type Cmd func() Msg

func NewApp(m Model) *App {
	return &App{
		m:    m,
		msgs: make(chan Msg, 10),
		cmds: make(chan Cmd),
		errs: make(chan error),
	}
}

func (a *App) Run() error {
	a.ctx, a.cancel = context.WithCancel(context.Background())

	go a.handleSignals()
	go a.userInputLoop()
	go a.commandLoop()

	err := a.eventLoop()
	// TODO teardown
	return err
}

func (a *App) userInputLoop() {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		a.msgs <- InputMsg{
			Input: s.Text(),
		}
	}
	if err := s.Err(); err != nil {
		a.errs <- err
	}
}

func (a *App) handleSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-sigCh:
			a.msgs <- QuitMsg{}
		}
	}
}

func (a *App) commandLoop() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case cmd := <-a.cmds:
			if cmd == nil {
				continue
			}
			go func() {
				a.msgs <- cmd()
			}()
		}
	}
}

func (a *App) eventLoop() error {
	m := a.m
	var cmd Cmd
	for {
		select {
		case <-a.ctx.Done():
			return nil
		case msg := <-a.msgs:
			switch msg.(type) {
			case QuitMsg:
				a.cancel()
			default:
				m, cmd = m.Update(msg)
				a.cmds <- cmd
			}
		}
	}
}

type InputMsg struct {
	Input string
}

type QuitMsg struct{}
