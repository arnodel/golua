package main

import (
	"bytes"
	"embed"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/arnodel/golua/lib"
	"github.com/arnodel/golua/lib/debuglib"
	"github.com/arnodel/golua/runtime"
	rt "github.com/arnodel/golua/runtime"
)

//go:embed templates
var templatesFS embed.FS

var templates *template.Template

var (
	cpuLimit uint64
	memLimit uint64
	saveDir  string
)

const maxCodeLength = 10000

var errCodeTooLarge = fmt.Errorf("Code too large (max len %d)", maxCodeLength)

func main() {
	var port uint

	flag.UintVar(&port, "port", 8080, "port to listen on")
	flag.Uint64Var(&cpuLimit, "cpulimit", 1000000, "cpu limit")
	flag.Uint64Var(&memLimit, "memlimit", 1000000, "mem limit")
	flag.StringVar(&saveDir, "savedir", "", "directory to save source code")
	flag.Parse()

	var err error

	dir, err := os.Stat(saveDir)
	if err != nil {
		log.Printf("Saving disabled: unable to open saving directory at %s: %s", saveDir, err)
		saveDir = ""
	} else if !dir.IsDir() {
		log.Printf("Saving disabled: save directory %s is not a directory", saveDir)
		saveDir = ""
	}

	templates, err = template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/", handleRequest)
	http.HandleFunc("/save", handleSave)
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

func handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if saveDir == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	code, err := getCode(w, r)
	if err != nil {
		return
	}
	codeBytes := []byte(code)
	h := codeHash(codeBytes)
	err = os.WriteFile(path.Join(saveDir, h+".lua"), codeBytes, 0600)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	newURL := *r.URL
	newURL.Path = ""
	newURL.RawQuery = fmt.Sprintf("h=%s", h)
	http.Redirect(w, r, newURL.String(), http.StatusSeeOther)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	h := r.URL.Query().Get("h")
	var code = defaultCode
	if h != "" && saveDir != "" {
		filePath := path.Join(saveDir, h+".lua")
		codeBytes, err := os.ReadFile(filePath)
		if errors.Is(err, os.ErrNotExist) {
			newURL := *r.URL
			newURL.RawQuery = ""
			http.Redirect(w, r, newURL.String(), http.StatusSeeOther)
			return
		}
		if err != nil {
			log.Printf("Error reading file %s: %s", filePath, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		code = string(codeBytes)
	}

	err := templates.ExecuteTemplate(w, "playground.html", playgroundModel{
		Source: code,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	code, err := getCode(w, r)
	if err != nil {
		return
	}
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

func getCode(w http.ResponseWriter, r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCodeLength*3) // need more bytes because of encoding
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return "", err
	}
	// Annoyingly new lines come as CRLF
	code := normalizeNewLines(r.FormValue("code"))
	if len(code) > maxCodeLength {
		log.Printf("Request too large: %d > %d", len(code), maxCodeLength)
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return "", errCodeTooLarge
	}
	return code, nil
}

func runCode(src []byte) (string, rt.RuntimeContext) {
	h := codeHash(src)
	log.Printf("Running code %s", h)
	defer log.Printf("Done running code %s", h)
	stdout := cappedBuffer{MaxSize: 10000}
	r := runtime.New(&stdout)
	lib.LoadAll(r)
	t := r.MainThread()
	clos, err := t.CompileAndLoadLuaChunk("playground", src, r.GlobalEnv())
	if err != nil {
		_, _ = io.WriteString(&stdout, err.Error())
		return stdout.String(), nil
	}
	ctx, callErr := r.CallContext(rt.RuntimeContextDef{
		HardLimits: rt.RuntimeResources{
			Cpu: cpuLimit,
			Mem: memLimit,
		},
		RequiredFlags:  rt.ComplyCpuSafe | rt.ComplyIoSafe | rt.ComplyMemSafe,
		MessageHandler: debuglib.Traceback,
	}, func() *rt.Error {
		return rt.Call(t, rt.FunctionValue(clos), nil, rt.NewTerminationWith(nil, 0, false))
	})
	if callErr != nil {
		stdout.WriteString(callErr.Error())
	}
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

func codeHash(src []byte) string {
	h := fnv.New64a()
	h.Write(src)
	return strconv.FormatUint(h.Sum64(), 36)
}

type playgroundModel struct {
	Source  string
	Output  string
	Context rt.RuntimeContext
}

func (m playgroundModel) Status() string {
	if m.Context == nil {
		return "Unknown"
	}
	switch m.Context.Status() {
	case rt.StatusDone:
		return "Completed wuccessfully"
	case rt.StatusError:
		return "Completed with error"
	case rt.StatusKilled:
		return "Killed"
	case rt.StatusLive:
		return "Live"
	default:
		return "Unknown"
	}
}

func (m playgroundModel) Cpu() uint64 {
	return cpuLimit
}

func (m playgroundModel) Mem() uint64 {
	return memLimit
}

func (m playgroundModel) SavingEnabled() bool {
	return saveDir != ""
}

const defaultCode = `
local a = "x"
while true do
	a = a .. a
	print(a)
end
`

var newLines = regexp.MustCompile(`(?s)\r\n|\n\r|\r`)

func normalizeNewLines(s string) string {
	if strings.IndexByte(s, '\r') == -1 {
		return s
	}
	return newLines.ReplaceAllLiteralString(s, "\n")
}
