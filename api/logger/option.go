package logger

import "github.com/equinor/seismic-cloud/api/events"

type LogEventOption interface {
	apply(*events.Event)
}

type EmptyOption struct{}

func (EmptyOption) apply(*events.Event) { return }

type funcOption struct {
	f func(*events.Event)
}

func (fo *funcOption) apply(h *events.Event) {
	fo.f(h)
	return
}

func newFuncOption(f func(*events.Event)) *funcOption {
	return &funcOption{
		f: f,
	}
}
