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
)

func main() {
	flag.Parse()
	packageName := "main"
	if flag.NArg() > 0 {
		packageName = flag.Arg(0)
	}

	var raw bytes.Buffer
	if refs, err := getNamedCharRefs(); err != nil {
		log.Fatal(err)
	} else {
		fmt.Fprintln(&raw, "// this file is mechanically generated, do not edit")
		fmt.Fprintln(&raw, "package "+packageName)
		fmt.Fprintln(&raw, "var namedCharRefs = map[string]string {")
		for k, v := range refs {
			fmt.Fprintf(&raw, "\t%q: %q,\n", k, v)
		}
		fmt.Fprintln(&raw, "}")
	}

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

	type charref struct {
		Codepoints []int  `json:"codepoints"`
		Characters string `json:"characters"`
	}

	m := make(map[string]charref)

	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}

	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v.Characters
	}

	return result, nil
}
