package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type ListJsonAnswer struct {
	ParseError    error
	RetrieveError error
	Results       []UnmarshalEntry
}

type UnmarshalEntry struct {
	Id        string
	Title     string
	Text      string
	Priority  Priority
	TriggerAt string
	Sort      string
}

type Priority int

const (
	STICKY Priority = Priority(iota)
	NOW
	LATER
	NOTES
	TIMED
	DONE
	INVALID
)

type OntoEntry struct {
	Data     string
	Children []OntoEntry
}

func readToken() (basePath, tok string) {
	fh, err := os.Open(os.ExpandEnv("$HOME/.config/POOCH_TOKEN"))
	must(err)
	defer fh.Close()
	bs, err := ioutil.ReadAll(fh)
	must(err)
	s := strings.TrimSpace(string(bs))
	v := strings.SplitN(s, ";", 2)
	return v[0], v[1]
}

func readUrl(client *http.Client, u string) ListJsonAnswer {
	resp, err := client.Get(u)
	must(err)
	defer resp.Body.Close()
	bodyb, err := ioutil.ReadAll(resp.Body)
	var lja ListJsonAnswer
	json.Unmarshal(bodyb, &lja)
	return lja
}

func readOntology(client *http.Client, u string) []OntoEntry {
	resp, err := client.Get(u)
	must(err)
	defer resp.Body.Close()
	bodyb, err := ioutil.ReadAll(resp.Body)
	var f []interface{}
	json.Unmarshal(bodyb, &f)
	return convertIntoOntoEntries(f)
}

func convertIntoOntoEntries(f []interface{}) []OntoEntry {
	r := make([]OntoEntry, len(f))
	for i := range f {
		switch e := f[i].(type) {
		case string:
			r[i].Data = e
		case map[string]interface{}:
			r[i].Data = e["data"].(string)
			r[i].Children = convertIntoOntoEntries(e["children"].([]interface{}))
		}
	}
	return r
}

func parseCols(cols string) map[string]string {
	r := map[string]string{}

	for _, line := range strings.Split(cols, "\n") {
		v := strings.SplitN(line, ": ", 2)
		switch len(v) {
		case 1:
			r[v[0]] = ""
		case 2:
			r[v[0]] = v[1]
		}
	}

	return r
}

func findNode(ontology []OntoEntry, name string) *OntoEntry {
	if ontology == nil {
		return nil
	}
	for i := range ontology {
		if ontology[i].Data == name {
			return &ontology[i]
		}
		if oe := findNode(ontology[i].Children, name); oe != nil {
			return oe
		}
	}
	return nil
}

func ontoEntriesToStrings(ontology []OntoEntry) []string {
	r := make([]string, len(ontology))
	for i := range ontology {
		r[i] = ontology[i].Data
	}
	return r
}

func findSubcols(ontology []OntoEntry, name string) []string {
	oe := findNode(ontology, name)
	if oe == nil {
		return []string{}
	} else {
		return ontoEntriesToStrings(oe.Children)
	}
}
