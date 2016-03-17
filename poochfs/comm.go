package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var tr = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}
var client = &http.Client{Transport: tr}

func readOntology() (query []string, tag []string) {
	resp, err := client.Get(os.ExpandEnv("https://$POOCHSRV/ontology?apiToken=$POOCHKEY"))
	must(err)
	defer resp.Body.Close()
	var out []interface{}
	must(json.NewDecoder(resp.Body).Decode(&out))
	all := []string{}
	all = flattenAppend(out, all)

	for i := range all {
		if strings.HasPrefix(all[i], "#%") {
			if len(all[i]) < 3 {
				continue
			}
			query = append(query, all[i][2:])
		} else {
			if len(all[i]) < 2 {
				continue
			}
			if all[i][1] == '#' {
				continue
			}
			tag = append(tag, all[i][1:])
		}
	}
	return query, tag
}

func flattenAppend(in []interface{}, v []string) []string {
	for i := range in {
		switch o := in[i].(type) {
		case string:
			v = append(v, o)
		case map[string]interface{}:
			v = append(v, o["data"].(string))
			v = flattenAppend(o["children"].([]interface{}), v)
		default:
			log.Fatal("wrong type %T", o)
		}
	}
	return v
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

func (p *Priority) String() string {
	switch *p {
	case STICKY:
		return "sticky"
	case NOW:
		return "now"
	case LATER:
		return "later"
	case NOTES:
		return "notes"
	case TIMED:
		return "timed"
	case DONE:
		return "done"
	}
	return "unknown"
}

type listResponse struct {
	ParseError    string
	RetrieveError string
	Results       []listResult
}

type listResult struct {
	Id        string
	Title     string
	Text      string
	Priority  Priority
	TriggerAt string
	Sort      string
}

func readList(query string) listResponse {
	resp, err := client.Get(os.ExpandEnv("https://$POOCHSRV/list.json?apiToken=$POOCHKEY&q=" + url.QueryEscape(query)))
	must(err)
	defer resp.Body.Close()

	var out listResponse
	must(json.NewDecoder(resp.Body).Decode(&out))

	return out
}

func saveEntry(id, body string) error {
	_, err := client.PostForm(os.ExpandEnv("https://$POOCHSRV/nf/update.json?apiToken=$POOCHKEY"), url.Values{"id": {"#id=" + id}, "body": {body}})
	return err
}
