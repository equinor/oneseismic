package server

type EmptyOption struct{}

func (EmptyOption) apply(*HttpServer) (err error) { return }

type funcOption struct {
	f func(*HttpServer) error
}

func (fo *funcOption) apply(h *HttpServer) error {

	return fo.f(h)
}

func newFuncOption(f func(*HttpServer) error) *funcOption {
	return &funcOption{
		f: f,
	}
}
