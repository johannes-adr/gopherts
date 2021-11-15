package transpiler

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/dop251/goja"
	"github.com/johannes-adr/typescripttranspiler/bytefmt"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/js"
)

const TYPESCRIPT_SOURCE_ONLINE = "https://raw.githubusercontent.com/microsoft/TypeScript/main/lib/typescript.js"

type TypeScript struct {

	//arg1: sourcecode written in typescript
	//arg2: compilerOptions
	//returns: javascript code
	Transpile func(string, map[string]interface{}) string
	Version   string

	//Binding to native javascript object
	TypeScriptObject *goja.Object
}

func getTypeScript(tssrc string) (*TypeScript, error) {

	tsProgram, err := goja.Compile("typescript.js", tssrc, false)
	if err != nil {
		return nil, err
	}
	vm := goja.New()
	vm.RunProgram(tsProgram)

	var transpile func(string, map[string]interface{}) string
	ts := vm.Get("ts").ToObject(vm)
	err = vm.ExportTo(ts.Get("transpile"), &transpile)
	if err != nil {
		return nil, err
	}
	return &TypeScript{Transpile: transpile, Version: ts.Get("version").Export().(string), TypeScriptObject: ts}, nil
}

func GetTypescript() (*TypeScript, error) {

	ts, err := fetchTypescriptOrExisting()
	if err == nil {
		return ts, nil
	}

	return nil, nil
}

func fetchTypescriptOrExisting() (*TypeScript, error) {
	cachedir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	foldername := path.Join(cachedir, "golang_typescript")
	filename := path.Join(foldername, "typescript.js")

	stats, err := os.Stat(filename)
	if err != nil && stats == nil {

		log.Printf("File '%s' not found - downloading from '%s'", filename, TYPESCRIPT_SOURCE_ONLINE)
		start := time.Now()
		res, err := http.Get(TYPESCRIPT_SOURCE_ONLINE)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			return nil, errors.New(res.Status)
		}
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		log.Printf("Successfully downloaded typescript source (%s - %s)", bytefmt.ByteSize(uint64(len(body))), time.Since(start))
		err = os.MkdirAll(foldername, os.ModeDir)
		if err != nil {
			return nil, err
		}
		file, err := os.Create(filename)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		minifier := minify.New()
		minifier.AddFunc("application/javascript", js.Minify)
		minified, err := minifier.Bytes("application/javascript", body)
		if err != nil {
			return nil, err
		}
		file.Write(minified)
		log.Printf("Successfully saved minified typescript source (%s)", bytefmt.ByteSize(uint64(len(minified))))
	}
	_, err = os.Stat(filename)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return getTypeScript(string(data))
}
