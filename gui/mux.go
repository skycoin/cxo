package gui

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"
	"sync"
)

type Context struct {
	Response   http.ResponseWriter
	Request    *http.Request
	parameters params
}

type params map[string]string

type Handle func(ctx *Context) error

func NewRouter() *Router {
	return &Router{
		mu:       &sync.RWMutex{},
		branches: make(muxBranches, 10),
		routes:   make(map[string]bool),
	}
}

type Router struct {
	mu       *sync.RWMutex
	branches muxBranches
	routes   map[string]bool
}

func (r *Router) Serve(address string) error {
	for routeName := range r.routes {
		fmt.Println("routeName:", routeName)
	}
	server := http.Server{
		Addr:    address,
		Handler: r,
	}
	fmt.Println("Listening and serving on", address)

	return server.ListenAndServe()
}

func (r *Router) ServeHTTP(ww http.ResponseWriter, rr *http.Request) {
	h, parameters, err := r.findRoute(rr.Method, rr.URL.Path)
	if err != nil {
		http.Error(ww, "not found", http.StatusNotFound)
		return
	}
	logger.Debug("parameters: %#v", parameters)

	logger.Infof("%s", rr.URL.Path)

	newContext := Context{
		Request:    rr,
		Response:   ww,
		parameters: parameters,
	}
	err = h(&newContext)

	if err != nil {
		logger.Errorf("Error: %v", err)
	}

	// TODO use error
}

func (r *Router) StaticFile(route string, path string) {
	handler := func(ctx *Context) error {
		logger.Debug("Serving static file: %s", path)
		http.ServeFile(ctx.Response, ctx.Request, path)
		return nil
	}

	if err := r.add("GET", route, handler, false); err != nil {
		panic(err)
	}
	fmt.Printf("\n%#v\n", r.branches)

	found, _, err := r.TestHandle("GET", route)
	if err != nil {
		panic(err)
	}

	sf1 := reflect.ValueOf(handler)
	sf2 := reflect.ValueOf(found)
	if sf1.Pointer() != sf2.Pointer() {
		panic("found different: " + route)
	}
}

func (r *Router) Folder(route string, dir string) {

	handler := func(ctx *Context) error {
		logger.Debug("Serving static file: %s", path.Join(dir, route))
		hh := http.FileServer(http.Dir(dir))
		hh.ServeHTTP(ctx.Response, ctx.Request)
		return nil
	}

	if err := r.add("GET", route, handler, true); err != nil {
		panic(err)
	}
	//fmt.Printf("\n%#v\n", r.branches)

	found, _, err := r.TestHandle("GET", route)
	if err != nil {
		panic(err)
	}

	sf1 := reflect.ValueOf(handler)
	sf2 := reflect.ValueOf(found)
	if sf1.Pointer() != sf2.Pointer() {
		panic("found different: " + route)
	}
}

func (r *Router) GET(route string, handler Handle) {
	if err := r.add("GET", route, handler, false); err != nil {
		panic(err)
	}
	//fmt.Printf("\n%#v\n", r.branches)

	found, _, err := r.TestHandle("GET", route)
	if err != nil {
		panic(err)
	}

	sf1 := reflect.ValueOf(handler)
	sf2 := reflect.ValueOf(found)
	if sf1.Pointer() != sf2.Pointer() {
		panic("found different: " + route)
	}
}

func (r *Router) POST(route string, handler Handle) {
	if err := r.add("POST", route, handler, false); err != nil {
		panic(err)
	}
	//fmt.Printf("\n%#v\n", r.branches)

	found, _, err := r.TestHandle("POST", route)
	if err != nil {
		panic(err)
	}

	sf1 := reflect.ValueOf(handler)
	sf2 := reflect.ValueOf(found)
	if sf1.Pointer() != sf2.Pointer() {
		panic("found different: " + route)
	}
}
func (r *Router) DELETE(route string, handler Handle) {
	if err := r.add("DELETE", route, handler, false); err != nil {
		panic(err)
	}
	//fmt.Printf("\n%#v\n", r.branches)

	found, _, err := r.TestHandle("DELETE", route)
	if err != nil {
		panic(err)
	}

	sf1 := reflect.ValueOf(handler)
	sf2 := reflect.ValueOf(found)
	if sf1.Pointer() != sf2.Pointer() {
		panic("found different: " + route)
	}
}

func (r *Router) TestHandle(method string, route string) (Handle, params, error) {
	return r.findRoute(method, route)
}

func (r *Router) findRoute(method string, route string) (Handle, params, error) {
	u, err := url.ParseRequestURI(route)
	if err != nil {
		return nil, params{}, err
	}
	urlParts := strings.Split(u.Path, "/")

	if r.branches == nil {
		r.branches = make(muxBranches)
	}

	urlPartsClean := []string{}
	for _, v := range urlParts {
		if v == "" {
			continue
		}
		urlPartsClean = append(urlPartsClean, v)
	}

	urlPartsClean = append([]string{method}, urlPartsClean...)

	if len(urlPartsClean) == 1 {
		fmt.Println("is home")
	}

	var current muxBranches
	current = r.branches
	var parameters params = make(params)

	var this interface{}
	partsLength := len(urlPartsClean)

Loop:
	for partIndex, part := range urlPartsClean {
		t, ok := current[part]
		if ok {
			println(part)
			this = t.this
			if t.isFolder {
				return this.(Handle), parameters, nil
				break Loop
			}
		} else {
			t, ok = current[":::"]
			//println(":::")
			if ok {
				parameters[*t.name] = part
				this = t.this
				if t.isFolder {
					return this.(Handle), parameters, nil
					break Loop
				}
			} else {
				this = nil
				break Loop
			}
		}

		if partsLength == (partIndex + 1) {
			if t.this == nil {
				this = nil
				break Loop
			}
		}
		current = t.hs
		this = t.this
		if t.isFolder {
			return this.(Handle), parameters, nil
			break Loop
		}
		//fmt.Printf("\nv: %#v\n", this.(Handle))
	}

	//fmt.Printf("this: %#v", this)
	if this != nil {
		h, ok := this.(Handle)
		if !ok {
			return nil, params{}, errors.New("assert error")
		}
		return h, parameters, nil
	}

	return nil, params{}, errors.New("not found")
}

func (r *Router) add(method string, route string, handler Handle, isFolder bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, err := url.ParseRequestURI(route)
	if err != nil {
		return err
	}
	urlParts := strings.Split(u.Path, "/")

	if r.branches == nil {
		r.branches = make(muxBranches)
	}

	urlWithPlaceholders := method + "/"
	urlPartsClean := []string{}
	urlPartsClean = append([]string{method}, urlPartsClean...)
	for _, v := range urlParts {
		if v == "" {
			continue
		}
		urlPartsClean = append(urlPartsClean, v)

		if strings.HasPrefix(v, ":") {
			if len(v) == 1 {
				return errors.New("invalid url parameter name")
			}
			v = ":"
		}
		urlWithPlaceholders += v + "/"
	}

	if _, ok := r.routes[urlWithPlaceholders]; ok {
		return errors.New("route already exists: " + urlWithPlaceholders)
	}
	r.routes[urlWithPlaceholders] = true

	var current muxBranches
	current = r.branches

	if current == nil {
		current = make(muxBranches)
	}

	partsLength := len(urlPartsClean)
	for partIndex, v := range urlPartsClean {
		var parameterName string
		if strings.HasPrefix(v, ":") {
			parameterName = strings.TrimPrefix(v, ":")
			v = ":::"
		}
		_, ok := current[v]
		var thisBranch branch

		if !ok {
			thisBranch = branch{}
			thisBranch.hs = make(muxBranches)
			if strings.HasPrefix(v, ":") {
				thisBranch.name = &parameterName
				//v = ":::"
			}
			if current == nil {
				current = make(muxBranches)
			}
			if partsLength == (partIndex + 1) {
				thisBranch.isFolder = isFolder
				thisBranch.this = handler
			}
			current[v] = thisBranch
		} else {
			thisBranch = current[v]
			if partsLength == (partIndex + 1) {
				thisBranch.isFolder = isFolder
				thisBranch.this = handler
			}
			current[v] = thisBranch
		}

		current = current[v].hs

	}

	//fmt.Println("\n", route, ":")

	return nil
}

type branch struct {
	this     Handle
	name     *string
	hs       muxBranches
	isFolder bool
}

type muxBranches map[string]branch
