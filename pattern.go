package urlpattern

import (
	"fmt"
	"net/url"
	"strings"
)

// Pattern stores information to match a URL
type Pattern struct {
	strictSlash    bool
	useEncodedPath bool
	regexp         routeRegexpGroup
	matchers       []*routeRegexp
	err            error
}

// NewPattern returns a new pattern instance with default options
func NewPattern() *Pattern {
	return NewPatternWithOptions(true, false)
}

// NewPatternWithOptions returns a new pattern instance with options
func NewPatternWithOptions(strictSlash, useEncodedPath bool) *Pattern {
	pattern := Pattern{
		strictSlash:    strictSlash,
		useEncodedPath: useEncodedPath,
		matchers:       []*routeRegexp{},
		err:            nil,
	}
	return &pattern
}

// Host adds a matcher for the URL host.
// It accepts a template with zero or more URL variables enclosed by {}.
// Variables can define an optional regexp pattern to be matched:
//
// - {name} matches anything until the next dot.
//
// - {name:pattern} matches the given regexp pattern.
//
// For example:
//
//     p := urlpattern.NewPattern()
//     p.Host("www.example.com")
//     p.Host("{subdomain}.domain.com")
//     p.Host("{subdomain:[a-z]+}.domain.com")
//
// Variable names must be unique in a given URL.
func (p *Pattern) Host(tpl string) *Pattern {
	p.err = p.addRegexpMatcher(tpl, true, false, false)
	return p
}

// Path adds a matcher for the URL path.
// It accepts a template with zero or more URL variables enclosed by {}. The
// template must start with a "/".
// Variables can define an optional regexp pattern to be matched:
//
// - {name} matches anything until the next slash.
//
// - {name:pattern} matches the given regexp pattern.
//
// For example:
//
//     p := urlpattern.NewPattern()
//     p.Path("/products/")
//     p.Path("/products/{key}")
//     p.Path("/articles/{category}/{id:[0-9]+}")
//
// Variable names must be unique in a given URL.
func (p *Pattern) Path(tpl string) *Pattern {
	p.err = p.addRegexpMatcher(tpl, false, false, false)
	return p
}

// PathPrefix adds a matcher for the URL path prefix. This matches if the given
// template is a prefix of the full URL path.
//
// Note that it does not treat slashes specially ("/foobar/" will be matched by
// the prefix "/foo") so you may want to use a trailing slash here.
//
// Also note that the setting of StrictSlash has no effect on routes
// with a PathPrefix matcher.
func (p *Pattern) PathPrefix(tpl string) *Pattern {
	p.err = p.addRegexpMatcher(tpl, false, true, false)
	return p
}

// Queries adds a matcher for URL query values.
// It accepts a sequence of key/value pairs. Values may define variables.
// For example:
//
//     p := urlpattern.NewPattern()
//     p.Queries("foo", "bar", "id", "{id:[0-9]+}")
//
// The above route will only match if the URL contains the defined queries
// values, e.g.: ?foo=bar&id=42.
//
// It the value is an empty string, it will match any value if the key is set.
//
// Variables can define an optional regexp pattern to be matched:
//
// - {name} matches anything until the next slash.
//
// - {name:pattern} matches the given regexp pattern.
func (p *Pattern) Queries(pairs ...string) *Pattern {
	length := len(pairs)
	if length%2 != 0 {
		p.err = fmt.Errorf(
			"mux: number of parameters must be multiple of 2, got %v", pairs)
		return nil
	}
	for i := 0; i < length; i += 2 {
		if p.err = p.addRegexpMatcher(pairs[i]+"="+pairs[i+1], false, false, true); p.err != nil {
			return p
		}
	}

	return p
}

// Match matches the pattern against the url.
func (p *Pattern) Match(u *url.URL) (map[string]string, bool) {
	if p.err != nil {
		return map[string]string{}, false
	}
	// Match everything.
	for _, m := range p.matchers {
		if matched := m.Match(u); !matched {
			return map[string]string{}, false
		}
	}
	// Set variables.
	vars := p.regexp.getMatch(u)
	return vars, true
}

// addRegexpMatcher adds a host or path matcher and builder to a route.
func (p *Pattern) addRegexpMatcher(tpl string, matchHost, matchPrefix, matchQuery bool) error {
	if p.err != nil {
		return p.err
	}
	if !matchHost && !matchQuery {
		if len(tpl) > 0 && tpl[0] != '/' {
			return fmt.Errorf("urlpattern: path must start with a slash, got %q", tpl)
		}
		if p.regexp.path != nil {
			tpl = strings.TrimRight(p.regexp.path.template, "/") + tpl
		}
	}
	rr, err := newRouteRegexp(tpl, matchHost, matchPrefix, matchQuery, p.strictSlash, p.useEncodedPath)
	if err != nil {
		return err
	}
	for _, q := range p.regexp.queries {
		if err = uniqueVars(rr.varsN, q.varsN); err != nil {
			return err
		}
	}
	if matchHost {
		if p.regexp.path != nil {
			if err = uniqueVars(rr.varsN, p.regexp.path.varsN); err != nil {
				return err
			}
		}
		p.regexp.host = rr
	} else {
		if p.regexp.host != nil {
			if err = uniqueVars(rr.varsN, p.regexp.host.varsN); err != nil {
				return err
			}
		}
		if matchQuery {
			p.regexp.queries = append(p.regexp.queries, rr)
		} else {
			p.regexp.path = rr
		}
	}

	// add the matcher
	if p.err == nil {
		p.matchers = append(p.matchers, rr)
	}
	return nil
}

// uniqueVars returns an error if two slices contain duplicated strings.
func uniqueVars(s1, s2 []string) error {
	for _, v1 := range s1 {
		for _, v2 := range s2 {
			if v1 == v2 {
				return fmt.Errorf("mux: duplicated route variable %q", v2)
			}
		}
	}
	return nil
}
