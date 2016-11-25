package logic

import (
	"net/http"
)

type HttpSrv struct {
	Host    string
	Routers map[string]http.HandlerFunc
}

func (hs *HttpSrv) Route(pattern string, f http.HandlerFunc) {
	hs.Routers[pattern] = f
}

func (hs *HttpSrv) Run() {
	for p, f := range hs.Routers {
		http.Handle(p, f)
	}
	err := http.ListenAndServe(hs.Host, nil)
	if err != nil {
		panic(err)
	}
}
