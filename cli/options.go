package main

import "context"

type option func(*App)

func WithContext(ctx context.Context) option {
	return func(a *App) {
		a.ctx = ctx
	}
}
