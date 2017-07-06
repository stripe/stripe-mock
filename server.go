package main

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ExpansionLevel represents expansions on a single "level" of resource. It may
// have subexpansions that are meant to take effect on resources that are
// nested below it (on other levels).
type ExpansionLevel struct {
	expansions map[string]*ExpansionLevel

	// wildcard specifies that everything should be expanded.
	wildcard bool
}

// ParseExpansionLevel parses a set of raw expansions from a request query
// string or form and produces a structure more useful for performing actual
// expansions.
func ParseExpansionLevel(raw []string) *ExpansionLevel {
	sort.Strings(raw)

	level := &ExpansionLevel{expansions: make(map[string]*ExpansionLevel)}
	groups := make(map[string][]string)

	for _, expansion := range raw {
		parts := strings.Split(expansion, ".")
		if len(parts) == 1 {
			if parts[0] == "*" {
				level.wildcard = true
			} else {
				level.expansions[parts[0]] = nil
			}
		} else {
			groups[parts[0]] = append(groups[parts[0]], strings.Join(parts[1:], "."))
		}
	}

	for key, subexpansions := range groups {
		level.expansions[key] = ParseExpansionLevel(subexpansions)
	}

	return level
}

// StubServer handles incoming HTTP requests and responds to them appropriately
// based off the set of OpenAPI routes that it's been configured with.
type StubServer struct {
	fixtures *Fixtures
	routes   map[HTTPVerb][]stubServerRoute
	spec     *OpenAPISpec
}

// stubServerRoute is a single route in a StubServer's routing table. It has a
// pattern to match an incoming path and a description of the method that would
// be executed in the event of a match.
type stubServerRoute struct {
	pattern *regexp.Regexp
	method  *OpenAPIMethod
}

// HandleRequest handes an HTTP request directed at the API stub.
func (s *StubServer) HandleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Printf("Request: %v %v (expansions: %+v)", r.Method, r.URL.Path)

	method := s.routeRequest(r)
	if method == nil {
		writeResponse(w, start, http.StatusNotFound, nil)
		return
	}

	response, ok := method.Responses["200"]
	if !ok {
		log.Printf("Couldn't find 200 response in spec")
		writeResponse(w, start, http.StatusInternalServerError, nil)
		return
	}

	if verbose {
		log.Printf("Response schema: %+v", response.Schema)
	}

	err := r.ParseForm()
	if err != nil {
		log.Printf("Couldn't parse query/body: %v", err)
		writeResponse(w, start, http.StatusInternalServerError, nil)
		return
	}

	expansions, rawExpansions := extractExpansions(r)
	if verbose {
		log.Printf("Expansions: %+v", rawExpansions)
	}

	generator := DataGenerator{s.spec.Definitions, s.fixtures}
	data, err := generator.Generate(response.Schema, r.URL.Path, expansions)
	if err != nil {
		log.Printf("Couldn't generate response: %v", err)
		writeResponse(w, start, http.StatusInternalServerError, nil)
		return
	}
	writeResponse(w, start, http.StatusOK, data)
}

func (s *StubServer) initializeRouter() {
	var numEndpoints int
	var numPaths int

	s.routes = make(map[HTTPVerb][]stubServerRoute)

	for path, verbs := range s.spec.Paths {
		numPaths++

		pathPattern := compilePath(path)

		if verbose {
			log.Printf("Compiled path: %v", pathPattern.String())
		}

		for verb, method := range verbs {
			numEndpoints++

			route := stubServerRoute{
				pattern: pathPattern,
				method:  method,
			}

			// net/http will always give us verbs in uppercase, so build our
			// routing table this way too
			verb = HTTPVerb(strings.ToUpper(string(verb)))

			s.routes[verb] = append(s.routes[verb], route)
		}
	}

	log.Printf("Routing to %v path(s) and %v endpoint(s)",
		numPaths, numEndpoints)
}

func (s *StubServer) routeRequest(r *http.Request) *OpenAPIMethod {
	verbRoutes := s.routes[HTTPVerb(r.Method)]
	for _, route := range verbRoutes {
		if route.pattern.MatchString(r.URL.Path) {
			return route.method
		}
	}
	return nil
}

// ---

var pathParameterPattern = regexp.MustCompile(`\{(\w+)\}`)

func compilePath(path OpenAPIPath) *regexp.Regexp {
	pattern := `\A`
	parts := strings.Split(string(path), "/")

	for _, part := range parts {
		if part == "" {
			continue
		}

		submatches := pathParameterPattern.FindAllStringSubmatch(part, -1)
		if submatches == nil {
			pattern += `/` + part
		} else {
			pattern += `/(?P<` + submatches[0][1] + `>\w+)`
		}
	}

	return regexp.MustCompile(pattern + `\z`)
}

func extractExpansions(r *http.Request) (*ExpansionLevel, []string) {
	var expansions []string
	expansions = append(expansions, r.Form["expand"]...)
	expansions = append(expansions, r.Form["expand[]"]...)
	return ParseExpansionLevel(expansions), expansions
}

func writeResponse(w http.ResponseWriter, start time.Time, status int, data interface{}) {
	if data == nil {
		data = []byte(http.StatusText(status))
	}

	encodedData, err := json.Marshal(&data)
	if err != nil {
		log.Printf("Error serializing response: %v", err)
		writeResponse(w, start, http.StatusInternalServerError, nil)
		return
	}

	w.WriteHeader(status)
	_, err = w.Write(encodedData)
	if err != nil {
		log.Printf("Error writing to client: %v", err)
	}
	log.Printf("Response: elapsed=%v status=%v", time.Now().Sub(start), status)
}
