package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"log"
	"net/http"
	"os"
	"sort"
)

func main() {
	flag.Parse()
	packageName := "parser"
	if flag.NArg() > 0 {
		packageName = flag.Arg(0)
	}

	refs, err := getNamedCharRefs()
	if err != nil {
		log.Fatal(err)
	}

	var raw bytes.Buffer
	fmt.Fprintf(&raw, "// this file is mechanically generated, do not edit\npackage %s\nvar namedCharRefs = map[string]string {\n", packageName)

	// Create a slice of keys and sort them
	keys := make([]string, 0, len(refs))
	for k := range refs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Print keys in sorted order
	for _, k := range keys {
		fmt.Fprintf(&raw, "\t%q: %q,\n", k, refs[k])
	}
	fmt.Fprintln(&raw, "}")

	formatted, err := format.Source(raw.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	os.Stdout.Write(formatted)
}

func getNamedCharRefs() (map[string]string, error) {
	resp, err := http.Get("https://html.spec.whatwg.org/entities.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var m map[string]struct {
		Characters string `json:"characters"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}

	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v.Characters
	}

	return result, nil
}
