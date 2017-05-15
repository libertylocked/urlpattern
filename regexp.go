package urlpattern

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// newRouteRegexp parses a route template and returns a routeRegexp,
// used to match a host, a path or a query string.
//
// It will extract named variables, assemble a regexp to be matched, create
// a "reverse" template to build URLs and compile regexps to validate variable
// values used in URL building.
//
// Previously we accepted only Python-like identifiers for variable
// names ([a-zA-Z_][a-zA-Z0-9_]*), but currently the only restriction is that
// name and pattern can't be empty, and names can't contain a colon.
func newRouteRegexp(tpl string, matchHost, matchPrefix, matchQuery, strictSlash, useEncodedPath bool) (*routeRegexp, error) {
	// Check if it is well-formed.
	idxs, errBraces := braceIndices(tpl)
	if errBraces != nil {
		return nil, errBraces
	}
	// Backup the original.
	template := tpl
	// Now let's parse it.
	defaultPattern := "[^/]+"
	if matchQuery {
		defaultPattern = "[^?&]*"
	} else if matchHost {
		defaultPattern = "[^.]+"
		matchPrefix = false
	}
	// Only match strict slash if not matching
	if matchPrefix || matchHost || matchQuery {
		strictSlash = false
	}
	// Set a flag for strictSlash.
	endSlash := false
	if strictSlash && strings.HasSuffix(tpl, "/") {
		tpl = tpl[:len(tpl)-1]
		endSlash = true
	}
	varsN := make([]string, len(idxs)/2)
	varsR := make([]*regexp.Regexp, len(idxs)/2)
	pattern := bytes.NewBufferString("")
	pattern.WriteByte('^')
	reverse := bytes.NewBufferString("")
	var end int
	var err error
	for i := 0; i < len(idxs); i += 2 {
		// Set all values we are interested in.
		raw := tpl[end:idxs[i]]
		end = idxs[i+1]
		parts := strings.SplitN(tpl[idxs[i]+1:end-1], ":", 2)
		name := parts[0]
		patt := defaultPattern
		if len(parts) == 2 {
			patt = parts[1]
		}
		// Name or pattern can't be empty.
		if name == "" || patt == "" {
			return nil, fmt.Errorf("mux: missing name or pattern in %q",
				tpl[idxs[i]:end])
		}
		// Build the regexp pattern.
		fmt.Fprintf(pattern, "%s(?P<%s>%s)", regexp.QuoteMeta(raw), varGroupName(i/2), patt)

		// Build the reverse template.
		fmt.Fprintf(reverse, "%s%%s", raw)

		// Append variable name and compiled pattern.
		varsN[i/2] = name
		varsR[i/2], err = regexp.Compile(fmt.Sprintf("^%s$", patt))
		if err != nil {
			return nil, err
		}
	}
	// Add the remaining.
	raw := tpl[end:]
	pattern.WriteString(regexp.QuoteMeta(raw))
	if strictSlash {
		pattern.WriteString("[/]?")
	}
	if matchQuery {
		// Add the default pattern if the query value is empty
		if queryVal := strings.SplitN(template, "=", 2)[1]; queryVal == "" {
			pattern.WriteString(defaultPattern)
		}
	}
	if !matchPrefix {
		pattern.WriteByte('$')
	}
	reverse.WriteString(raw)
	if endSlash {
		reverse.WriteByte('/')
	}
	// Compile full regexp.
	reg, errCompile := regexp.Compile(pattern.String())
	if errCompile != nil {
		return nil, errCompile
	}

	// Check for capturing groups which used to work in older versions
	if reg.NumSubexp() != len(idxs)/2 {
		panic(fmt.Sprintf("route %s contains capture groups in its regexp. ", template) +
			"Only non-capturing groups are accepted: e.g. (?:pattern) instead of (pattern)")
	}

	// Done!
	return &routeRegexp{
		template:       template,
		matchHost:      matchHost,
		matchQuery:     matchQuery,
		strictSlash:    strictSlash,
		useEncodedPath: useEncodedPath,
		regexp:         reg,
		reverse:        reverse.String(),
		varsN:          varsN,
		varsR:          varsR,
	}, nil
}

// routeRegexp stores a regexp to match a host or path and information to
// collect and validate route variables.
type routeRegexp struct {
	// The unmodified template.
	template string
	// True for host match, false for path or query string match.
	matchHost bool
	// True for query string match, false for path and host match.
	matchQuery bool
	// The strictSlash value defined on the route, but disabled if PathPrefix was used.
	strictSlash bool
	// Determines whether to use encoded path from getPath function or unencoded
	// req.URL.Path for path matching
	useEncodedPath bool
	// Expanded regexp.
	regexp *regexp.Regexp
	// Reverse template.
	reverse string
	// Variable names.
	varsN []string
	// Variable regexps (validators).
	varsR []*regexp.Regexp
}

// Match matches the regexp against the URL host or path.
func (r *routeRegexp) Match(u *url.URL) bool {
	if !r.matchHost {
		if r.matchQuery {
			return r.matchQueryString(u)
		}
		path := u.Path
		if r.useEncodedPath {
			path = getPath(u)
		}
		return r.regexp.MatchString(path)
	}
	host := getHost(u)
	return r.regexp.MatchString(host)
}

// url builds a URL part using the given values.
func (r *routeRegexp) url(values map[string]string) (string, error) {
	urlValues := make([]interface{}, len(r.varsN))
	for k, v := range r.varsN {
		value, ok := values[v]
		if !ok {
			return "", fmt.Errorf("mux: missing route variable %q", v)
		}
		urlValues[k] = value
	}
	rv := fmt.Sprintf(r.reverse, urlValues...)
	if !r.regexp.MatchString(rv) {
		// The URL is checked against the full regexp, instead of checking
		// individual variables. This is faster but to provide a good error
		// message, we check individual regexps if the URL doesn't match.
		for k, v := range r.varsN {
			if !r.varsR[k].MatchString(values[v]) {
				return "", fmt.Errorf(
					"mux: variable %q doesn't match, expected %q", values[v],
					r.varsR[k].String())
			}
		}
	}
	return rv, nil
}

// getURLQuery returns a single query parameter from a request URL.
// For a URL with foo=bar&baz=ding, we return only the relevant key
// value pair for the routeRegexp.
func (r *routeRegexp) getURLQuery(u *url.URL) string {
	if !r.matchQuery {
		return ""
	}
	templateKey := strings.SplitN(r.template, "=", 2)[0]
	for key, vals := range u.Query() {
		if key == templateKey && len(vals) > 0 {
			return key + "=" + vals[0]
		}
	}
	return ""
}

func (r *routeRegexp) matchQueryString(u *url.URL) bool {
	return r.regexp.MatchString(r.getURLQuery(u))
}

// braceIndices returns the first level curly brace indices from a string.
// It returns an error in case of unbalanced braces.
func braceIndices(s string) ([]int, error) {
	var level, idx int
	var idxs []int
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			if level++; level == 1 {
				idx = i
			}
		case '}':
			if level--; level == 0 {
				idxs = append(idxs, idx, i+1)
			} else if level < 0 {
				return nil, fmt.Errorf("mux: unbalanced braces in %q", s)
			}
		}
	}
	if level != 0 {
		return nil, fmt.Errorf("mux: unbalanced braces in %q", s)
	}
	return idxs, nil
}

// varGroupName builds a capturing group name for the indexed variable.
func varGroupName(idx int) string {
	return "v" + strconv.Itoa(idx)
}

// ----------------------------------------------------------------------------
// routeRegexpGroup
// ----------------------------------------------------------------------------

// routeRegexpGroup groups the route matchers that carry variables.
type routeRegexpGroup struct {
	host    *routeRegexp
	path    *routeRegexp
	queries []*routeRegexp
}

// getMatch extracts the variables from the URL once a route matches.
func (v *routeRegexpGroup) getMatch(u *url.URL) map[string]string {
	vars := map[string]string{}
	// Store host variables.
	if v.host != nil {
		host := getHost(u)
		matches := v.host.regexp.FindStringSubmatchIndex(host)
		if len(matches) > 0 {
			extractVars(host, matches, v.host.varsN, vars)
		}
	}
	path := u.Path
	// Store path variables.
	if v.path != nil {
		if v.path.useEncodedPath {
			path = getPath(u)
		}
		matches := v.path.regexp.FindStringSubmatchIndex(path)
		if len(matches) > 0 {
			extractVars(path, matches, v.path.varsN, vars)
		}
	}
	// Store query string variables.
	for _, q := range v.queries {
		queryURL := q.getURLQuery(u)
		matches := q.regexp.FindStringSubmatchIndex(queryURL)
		if len(matches) > 0 {
			extractVars(queryURL, matches, q.varsN, vars)
		}
	}
	return vars
}

// getHost tries its best to return the request host.
func getHost(u *url.URL) string {
	if u.IsAbs() {
		return u.Host
	}
	host := u.Host
	// Slice off any port information.
	if i := strings.Index(host, ":"); i != -1 {
		host = host[:i]
	}
	return host

}

// getPath returns the escaped path if possible; doing what URL.EscapedPath()
// which was added in go1.5 does
func getPath(u *url.URL) string {
	if u.RequestURI() != "" {
		// Extract the path from RequestURI (which is escaped unlike URL.Path)
		// as detailed here as detailed in https://golang.org/pkg/net/url/#URL
		// for < 1.5 server side workaround
		// http://localhost/path/here?v=1 -> /path/here
		path := u.RequestURI()
		path = strings.TrimPrefix(path, u.Scheme+`://`)
		path = strings.TrimPrefix(path, u.Host)
		if i := strings.LastIndex(path, "?"); i > -1 {
			path = path[:i]
		}
		if i := strings.LastIndex(path, "#"); i > -1 {
			path = path[:i]
		}
		return path
	}
	return u.Path
}

func extractVars(input string, matches []int, names []string, output map[string]string) {
	for i, name := range names {
		output[name] = input[matches[2*i+2]:matches[2*i+3]]
	}
}
