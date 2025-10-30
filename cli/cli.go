// Package cli provides a framework to create a dead simple text-based CLI app
package cli

import (
	"bufio"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type App struct {
	m       Model
	msgs    chan Msg
	cmds    chan Cmd
	errs    chan error
	ctx     context.Context
	cancel  context.CancelFunc
	options []option
}

type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	Teardown(context.Context) error
}

type Msg any

type Cmd func(context.Context) Msg

func NewApp(m Model, opts ...option) *App {
	return &App{
		m:       m,
		msgs:    make(chan Msg, 10),
		cmds:    make(chan Cmd),
		errs:    make(chan error),
		ctx:     context.Background(),
		options: opts,
	}
}

func (a *App) Run() error {
	for _, opt := range a.options {
		opt(a)
	}
	a.ctx, a.cancel = context.WithCancel(a.ctx)

	go a.handleSignals()
	go a.userInputLoop()
	go a.commandLoop()

	a.cmds <- a.m.Init()

	m := a.eventLoop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.Teardown(ctx)
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
				a.msgs <- cmd(a.ctx)
			}()
		}
	}
}

func (a *App) eventLoop() Model {
	m := a.m
	var cmd Cmd
	for {
		select {
		case <-a.ctx.Done():
			return m // final state
		case msg := <-a.msgs:
			if msg == nil {
				continue
			}
			switch msg := msg.(type) {
			case QuitMsg:
				a.cancel()
			case BatchMsg:
				a.executeBatch(msg)
			default:
				m, cmd = m.Update(msg)
				a.cmds <- cmd
			}
		}
	}
}

func (a *App) executeBatch(msg BatchMsg) {
	for _, cmd := range msg.cmds {
		if cmd == nil {
			continue
		}
		go func() {
			a.msgs <- cmd(a.ctx)
		}()
	}
}
