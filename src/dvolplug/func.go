package main

// This is a basic VMDK Docker Data Volume driver.
//
// It accepts VMDK name (in format of <filename>.vmdk or filename.vmdk@datastore)
// if the qualifiers is rwc (tbd: )
import (
	"fmt"
	"net/http"
	//		"net/http/httputil"
	//	"log"
	//		"strings"
	"encoding/json"
	"os"
	//	"runtime"
)

/*
#cgo LDFLAGS: -v -L. -lclient
int doit(int cid, int port);
*/
import "C"

type pluginRequest struct {
	Name string
	Opts map[string]string
}

func getDockerVolumeRequest(r *http.Request) (*pluginRequest, error) {
	request := &pluginRequest{}
	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		return nil, err
	}
	//	log.Debugf("Request from docker: %v", request)
	fmt.Fprintf(os.Stderr, "Request from docker: \n name=%s\n opts= %v\n", request.Name, request.Opts)
	return request, nil
}

func msg(w http.ResponseWriter, r *http.Request, n string) {
	//	_, file, line, _ := runtime.Caller(1)
	//		out, err := httputil.DumpRequestOut(r, true)
	//		if err != nil {
	//			log.Fatal(err)
	//		}
	//		fmt.Println(strings.Replace(string(out), "\r", "", -1))
	//	fmt.Fprintf(w, "%s => URL %s (%s: %d)!", n, r.URL.Path[1:], file, line)
	fmt.Fprintf(os.Stderr, "%s URI '%v %v'\n", r.URL.Path[1:], r.Method, r.RequestURI)
	getDockerVolumeRequest(r)
}

func sayOk(w http.ResponseWriter) {
	fmt.Fprintf(w, "{\"Err\": null}") // just say OK
}

func sayFakeMountpoint(w http.ResponseWriter) {
	fmt.Fprintf(w,
		"{"+
			"\"Mountpoint\": \"/path/to/directory/on/host\","+
			"\"Err\": null"+
			"}")
}

func activate(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "{\n\"Implements\": [\"VolumeDriver\"]}")
}

func unmount(w http.ResponseWriter, r *http.Request) {
	sayOk(w)
	msg(w, r, "UnMount")
}

func mount(w http.ResponseWriter, r *http.Request) {
	sayFakeMountpoint(w)
	msg(w, r, "Mount")
}

func create(w http.ResponseWriter, r *http.Request) {
	sayOk(w)
	msg(w, r, "Create")
}

func remove(w http.ResponseWriter, r *http.Request) {
	sayOk(w)
	msg(w, r, "Remove")
}

func path(w http.ResponseWriter, r *http.Request) {
	sayFakeMountpoint(w)
	msg(w, r, "Path")
}

func fallback(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(os.Stderr, "FALLBACK: Hi there, I love %s!\n", r.URL.Path[1:])
	//	fmt.Fprintf(w, "FALLBACK === Hi there, I love %s!", r.URL.Path[1:]) // write err
}
func main() {
	var api = map[string]func(w http.ResponseWriter, r *http.Request){
		"/Plugin.Activate":      activate,
		"/VolumeDriver.Mount":   mount,
		"/VolumeDriver.Unmount": unmount,
		"/VolumeDriver.Create":  create,
		"/VolumeDriver.Remove":  remove,
		"/VolumeDriver.Path":    path,
		"/":                     fallback,
	}

	// register handles
	for h, _ := range api {
		http.HandleFunc(h, api[h])
	}

	ret := C.doit(2, 15000)
	fmt.Println("C.doit() returned %d", ret)
	// and now serve the API
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("failed to start http server")
	}
}
