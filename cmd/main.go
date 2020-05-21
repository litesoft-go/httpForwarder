package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/litesoft-go/httpForwarder/pkg/utils/httpconfig"

	"github.com/litesoft-go/httpForwarder/pkg/utils/iso8601"
	"github.com/litesoft-go/httpForwarder/version"
)

// This application forwards requests like: http://localhost:9090/forward/172.17.0.10:8080/api/2.0
//      turning them into calls to (everything after "/forward/"): http://172.17.0.10:8080/api/2.0

const (
	PORT          = 13734
	forwardPrefix = "/forward/"
)

var (
	supportsRoot    Supports = 1
	supportsForward Supports = 2
	supportsFavicon Supports = 4
)
var methods = []string{"GET", "HEAD", "POST", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH"}

var methodMap = map[string]string{
	"GET":/* . . . */ "GET     #7", // The GET method requests a representation of the specified resource. Requests using GET should only retrieve data.
	"HEAD":/*  . . */ "HEAD    #2", // The HEAD method asks for a response identical to that of a GET request, but without the response body.
	"POST":/*  . . */ "POST    #2", // The POST method is used to submit an entity to the specified resource, often causing a change in state or side effects on the server.
	"PUT":/* . . . */ "PUT     #0", // The PUT method replaces all current representations of the target resource with the request payload.
	"DELETE":/*  . */ "DELETE  #2", // The DELETE method deletes the specified resource.
	"CONNECT":/* . */ "CONNECT #0", // The CONNECT method establishes a tunnel to the server identified by the target resource.
	"OPTIONS":/* . */ "OPTIONS #2", // The OPTIONS method is used to describe the communication options for the target resource.
	"TRACE":/* . . */ "TRACE   #3", // The TRACE method performs a message loop-back test along the path to the target resource.
	"PATCH":/* . . */ "PATCH   #0", // The PATCH method is used to apply partial modifications to a resource.
}

func handleUnsupportedMethod(pPathSupports Supports, w http.ResponseWriter, r *http.Request) {
	msg := "No http Methods supported"
	if pPathSupports.Any() {
		msg = "Only " + collectSupported(pPathSupports) + " currently supported"
	}
	returnText(w, 405, // Method Not Allowed
		msg+", but got ("+r.Method+") for path: "+r.URL.Path)
}

func collectSupported(pPathSupports Supports) string {
	var supported []string
	for _, method := range methods {
		if checkSupported(method, pPathSupports) {
			supported = append(supported, method)
		}
	}
	switch len(supported) {
	case 0:
		return "?None?"
	case 1:
		return wrap(supported, 0)
	case 2:
		return wrap2(supported, 1, 0)
	default:
		index := len(supported) - 1
		rv := wrap(supported, index)
		for index--; index > 1; index-- {
			rv += ", " + wrap(supported, index)
		}
		rv += ", " + wrap2(supported, 1, 0)
		return rv
	}
}

func wrap2(supported []string, index1, index2 int) string {
	return wrap(supported, index1) + " and " + wrap(supported, index2)
}

func wrap(supported []string, index int) string {
	return "'" + supported[index] + "'"
}

func checkSupported(method string, pPathSupports Supports) bool {
	methodSupports := parseSupports(mapMethod(method))
	return methodSupports.check(pPathSupports)
}

func parseSupports(mm string) (supports Supports) {
	mmLen := len(mm)
	if 6 <= mmLen {
		if mm[mmLen-2] == '#' {
			atoi, _ := strconv.Atoi(mm[mmLen-1:])
			supports = Supports(atoi)
		}
	}
	return
}

type Supports int

func (s Supports) Any() bool {
	return s != 0
}

func (s Supports) check(supportsType Supports) bool {
	return 0 != s&supportsType
}

type pathHandler func(http.ResponseWriter, *http.Request)

func dispatch(pMethodSupports, pPathSupport Supports, pPathHandler pathHandler, w http.ResponseWriter, r *http.Request) {
	fmt.Println("dispatch", r.Method, "-", pMethodSupports, "&", pPathSupport, ":", pMethodSupports.check(pPathSupport))
	if pMethodSupports.check(pPathSupport) {
		pPathHandler(w, r)
	} else {
		handleUnsupportedMethod(pPathSupport, w, r)
	}
}

func handleUnsupportedPath(w http.ResponseWriter, r *http.Request) {
	returnText(w, 404, // Not Found (path)
		"Path not supported: "+r.URL.Path)
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	handleUnsupportedPath(w, r) // TODO: Lame!
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	var msg string
	if r.Method == "GET" {
		msg = "httpForwarder vs: " + version.Version
	}
	returnText(w, 200, msg)
}

func handler(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	rURL := r.URL
	mappedMethod := mapMethod(method)
	fmt.Printf("Request: %s %s %v\n", iso8601.ToStringZmillis(nil), mappedMethod, rURL)
	methodSupports := parseSupports(mappedMethod)
	path := rURL.Path
	switch {
	case path == "/":
		dispatch(methodSupports, supportsRoot, handleRoot, w, r)
	case path == "/favicon.ico":
		dispatch(methodSupports, supportsFavicon, handleFavicon, w, r)
	case strings.HasPrefix(path, forwardPrefix):
		dispatch(methodSupports, supportsForward, handleForward, w, r)
	default:
		handleUnsupportedPath(w, r)
	}
}

func returnText(w http.ResponseWriter, statusCode int, msg string) {
	w.Header().Add("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(msg))
}

func mapMethod(method string) (longForm string) {
	longForm = methodMap[method]
	if longForm == "" {
		if method == "" {
			longForm = "?empty?"
		} else {
			longForm = "??" + method + "??"
		}
	}
	return
}

func main() {
	fmt.Printf("httpForwarder Version: %s\n", version.Version)
	fmt.Printf("Args: %v\n", os.Args[1:])

	http.HandleFunc("/", handler)
	fmt.Printf("Listening on %d\n", PORT)
	addr := fmt.Sprintf(":%d", PORT)
	err := http.ListenAndServe(addr, nil)
	if err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

var client = httpconfig.NewClient(30)

func handleForward(w http.ResponseWriter, r *http.Request) {
	rURL := r.URL
	scheme := rURL.Scheme
	if scheme == "" {
		scheme = "http"
	}
	newURL := scheme + "://" + rURL.Path[len(forwardPrefix):]
	if newURL == "" {
		returnText(w, 400, // Bad Request
			"Nothing after: "+forwardPrefix)
		return
	}
	query := rURL.RawQuery
	if query != "" {
		newURL += "?" + query
	}
	rh := ResponseHandler{rw: w}
	switch r.Method {
	case "GET":
		rh.handleResponse(client.Get(newURL))
	case "HEAD":
		rh.handleResponse(client.Head(newURL))
	case "POST":
		rh.handleResponse(sendPost(r, newURL))
	default:
		request, err := makeBodyLessRequest(r, newURL)
		if err != nil {
			msg := fmt.Sprintf("****** Error parsing new URL (%s): %s", newURL, err.Error())
			fmt.Println(msg)
			returnText(w, 400, // Bad Request
				msg)
		}
		rh.handleResponse(client.Do(request))
	}
}

//noinspection GoUnhandledErrorResult
func sendPost(in *http.Request, newURL string) (r *http.Response, err error) {
	readCloser := in.Body
	defer readCloser.Close()
	contentType := in.Header.Get("Content-type")
	return client.Post(newURL, contentType, readCloser)
}

// DELETE, OPTIONS, & TRACE
func makeBodyLessRequest(in *http.Request, newURL string) (*http.Request, error) {
	out := &http.Request{}
	parsedURL, err := url.Parse(newURL)
	if err == nil {
		*out = *in // copy simple
		out.URL = parsedURL
	}
	return out, err
}

type ResponseHandler struct {
	rw http.ResponseWriter
}

func (rh ResponseHandler) handleResponse(r *http.Response, err error) {
	rw := rh.rw
	if err == nil {
		var body []byte
		body, err = loadBody(r.Body)
		if err == nil {
			copyHeaders(r.Header, rw.Header())
			rw.WriteHeader(r.StatusCode)
			_ = writeBody(body, rw)
			return
		}
	}
	returnText(rw, 500, // Internal Server Error
		err.Error())
}

func copyHeaders(src, dst http.Header) {
	for k, v := range src {
		dst[k] = copySlice(v)
	}
}

func copySlice(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func loadBody(in io.ReadCloser) ([]byte, error) {
	defer func() {
		_ = in.Close() // Per Dave Cheney 2017 - auto drains!
	}()
	return ioutil.ReadAll(in)
}

func writeBody(body []byte, out io.Writer) error {
	for from := 0; from < len(body); {
		wrote, err := out.Write(body[from:])
		if err != nil {
			return err
		}
		from += wrote
	}
	return nil
}
