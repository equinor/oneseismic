package server

type EmptyOption struct{}

func (EmptyOption) apply(*HTTPServer) (err error) { return }

type funcOption struct {
	f func(*HTTPServer) error
}

func (fo *funcOption) apply(h *HTTPServer) error {

	return fo.f(h)
}

func newFuncOption(f func(*HTTPServer) error) *funcOption {
	return &funcOption{
		f: f,
	}
}
