package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"hash/fnv"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/runtime"
	rt "github.com/arnodel/golua/runtime"
)

//go:embed templates
var templatesFS embed.FS

var templates *template.Template

var (
	cpuLimit uint64
	memLimit uint64
)

func main() {
	var port uint

	flag.UintVar(&port, "port", 8080, "port to listen on")
	flag.Uint64Var(&cpuLimit, "cpulimit", 1000000, "cpu limit")
	flag.Uint64Var(&memLimit, "memlimit", 1000000, "mem limit")
	flag.Parse()

	var err error

	templates, err = template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", handleRequest)

	log.Printf("Listening on :%d", port)

	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		handleGet(w, r)
	case "POST":
		handlePost(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "playground.html", defaultModel)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 20000)
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	code := r.FormValue("code")
	output, ctx := runCode([]byte(code))
	err = templates.ExecuteTemplate(w, "playground.html", playgroundModel{
		Source:  code,
		Output:  output,
		Context: ctx,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func runCode(src []byte) (string, rt.RuntimeContext) {
	h := hashCode(src)
	log.Printf("Running code %d", h)
	defer log.Printf("Done running code %d", h)
	stdout := cappedBuffer{MaxSize: 10000}
	r := runtime.New(&stdout)
	lib.LoadAll(r)
	t := r.MainThread()
	clos, err := t.CompileAndLoadLuaChunk("playground", src, r.GlobalEnv())
	if err != nil {
		_, _ = io.WriteString(&stdout, err.Error())
		return stdout.String(), nil
	}
	ctx := r.CallContext(rt.RuntimeContextDef{
		CpuLimit:    cpuLimit,
		MemLimit:    memLimit,
		SafetyFlags: rt.ComplyCpuSafe | rt.ComplyIoSafe | rt.ComplyMemSafe,
	}, func() {
		cerr := rt.Call(t, rt.FunctionValue(clos), nil, rt.NewTerminationWith(0, false))
		if cerr != nil {
			_, _ = io.WriteString(&stdout, cerr.Traceback())
		}
	})
	return stdout.String(), ctx
}

type cappedBuffer struct {
	bytes.Buffer
	MaxSize int
}

func (b *cappedBuffer) Write(p []byte) (n int, err error) {
	maxLen := b.MaxSize - b.Len()
	if maxLen < len(p) {
		p = p[:maxLen]
	}
	return b.Buffer.Write(p)
}

func hashCode(src []byte) uint64 {
	h := fnv.New64a()
	h.Write(src)
	return h.Sum64()
}

type playgroundModel struct {
	Source  string
	Output  string
	Context rt.RuntimeContext
}

var defaultModel = playgroundModel{Source: `
local a = "x"
while true do
	a = a .. a
	print(a)
end
`}
