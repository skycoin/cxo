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

type Router struct {
	mu       *sync.RWMutex
	branches muxBranches
	routes   map[string]bool
}

type Handle func(ctx *Context) error

type params map[string]string

type branch struct {
	this     Handle
	name     *string
	hs       muxBranches
	isFolder bool
}

type muxBranches map[string]branch

type IRouterApi interface {
	Register(router *Router)
}

// NewRouter creates and returns pointer to a new router object
func NewRouter() *Router {
	return &Router{
		mu:       &sync.RWMutex{},
		branches: make(muxBranches, 10),
		routes:   make(map[string]bool),
	}
}

// Serve starts the server
func (r *Router) Serve(address string) error {
	server := http.Server{
		Addr:    address,
		Handler: r,
	}

	fmt.Println("Listening and serving on", address)

	return server.ListenAndServe()
}

// TODO: make this non-accessible
func (r *Router) ServeHTTP(ww http.ResponseWriter, rr *http.Request) {
	if origin := rr.Header.Get("Origin"); origin != "" {
		ww.Header().Set("Access-Control-Allow-Origin", origin)
		ww.Header().Set("Access-Control-Allow-Methods", "POST")
		ww.Header().Set("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	}

	if rr.Method == "OPTIONS" {
		return
	}
	// create new context
	newContext := Context{
		Request:  rr,
		Response: ww,
	}

	// find the handler
	h, parameters, err := r.findRoute(rr.Method, rr.URL.Path)
	if err != nil {
		newContext.ErrNotFound(ErrorNotFound.Error())
		return
	}
	if len(parameters) > 0 {
		logger.Debug("parameters: %#v", parameters)
	}

	logger.Infof("%v %s", rr.Method, rr.URL.Path)

	// set parameters in context
	newContext.parameters = parameters

	// serve handler
	err = h(&newContext)

	if err != nil {
		logger.Errorf("Error: %v", err)
	}
}

// StaticFile creates a route for a static file
func (r *Router) StaticFile(route string, path string) {
	handler := func(ctx *Context) error {
		logger.Debug("Serving static file: %s", path)
		http.ServeFile(ctx.Response, ctx.Request, path)
		return nil
	}

	if err := r.addRoute("GET", route, handler, false); err != nil {
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

// Folder creates a route for a folder
func (r *Router) Folder(route string, dir string) {

	handler := func(ctx *Context) error {
		logger.Debug("Serving static file: %s", path.Join(dir, route))
		hh := http.FileServer(http.Dir(dir))
		hh.ServeHTTP(ctx.Response, ctx.Request)
		return nil
	}

	if err := r.addRoute("GET", route, handler, true); err != nil {
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

// GET: handle GET method requests for this route with this Handle
func (r *Router) GET(route string, handler Handle) {
	if err := r.addRoute("GET", route, handler, false); err != nil {
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

// POST: handle POST method requests for this route with this Handle
func (r *Router) POST(route string, handler Handle) {
	if err := r.addRoute("POST", route, handler, false); err != nil {
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

// DELETE: handle DELETE method requests for this route with this Handle
func (r *Router) DELETE(route string, handler Handle) {
	if err := r.addRoute("DELETE", route, handler, false); err != nil {
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

// TestHandle returns the handle for the specified combination of method and route
func (r *Router) TestHandle(method string, route string) (Handle, params, error) {
	return r.findRoute(method, route)
}

func (r *Router) addRoute(method string, route string, handler Handle, isFolder bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	fmt.Println("Route:", method, route)

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
				return errors.New("url parameter name must be at least character long")
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

	var current muxBranches
	current = r.branches
	var parameters params = make(params)

	var this interface{}
	partsLength := len(urlPartsClean)
Loop:
	for partIndex, part := range urlPartsClean {
		t, ok := current[part]
		if ok {
			//println(part)
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

	return nil, params{}, ErrorNotFound
}
