package urlpattern

import (
	"net/url"
	"testing"
)

func TestMatchHostLetters(t *testing.T) {
	p := NewPattern()
	p.Host("{subdomain:[a-z]+}.example.com")
	u, _ := url.Parse("http://foo.example.com/api/events/12345")
	matches, matched := p.Match(u)
	if !matched {
		t.Fail()
	}
	if len(matches) != 1 {
		t.Fail()
	}
	if matches["subdomain"] != "foo" {
		t.Fail()
	}
}

func TestMatchHostNumbers(t *testing.T) {
	p := NewPattern()
	p.Host("{subdomain:[0-9]+}.example.com")
	u, _ := url.Parse("http://42.example.com/api/events/12345")
	matches, matched := p.Match(u)
	if !matched {
		t.Fail()
	}
	if len(matches) != 1 {
		t.Fail()
	}
	if matches["subdomain"] != "42" {
		t.Fail()
	}
}

func TestMatchHostNoMatch(t *testing.T) {
	p := NewPattern()
	p.Host("{subdomain}.example.com")
	u, _ := url.Parse("http://example.com/api/events/12345")
	matches, matched := p.Match(u)
	if matched {
		t.Fail()
	}
	if len(matches) != 0 {
		t.Fail()
	}
}

func TestMatchPath(t *testing.T) {
	p := NewPattern()
	p.Path("/api/{action:[A-Za-z]+}/{id:[0-9]+}")
	u, _ := url.Parse("http://example.com/api/events/12345")
	matches, matched := p.Match(u)
	if !matched {
		t.Fail()
	}
	if len(matches) != 2 {
		t.Fail()
	}
	if matches["action"] != "events" {
		t.Fail()
	}
	if matches["id"] != "12345" {
		t.Fail()
	}
}

func TestMatchPathPrefix(t *testing.T) {
	p := NewPattern()
	p.PathPrefix("/api")
	u, _ := url.Parse("http://example.com/api/events/12345")
	_, matched := p.Match(u)
	if !matched {
		t.Fail()
	}
}

func TestMatchPathPrefixNoMatch(t *testing.T) {
	p := NewPattern()
	p.PathPrefix("/client")
	u, _ := url.Parse("http://example.com/api/events/12345")
	_, matched := p.Match(u)
	if matched {
		t.Fail()
	}
}

func TestMatchHostAndPathAndPathPrefix(t *testing.T) {
	p := NewPattern().
		Host("{subdomain}.example.com").
		PathPrefix("/api").
		Path("/events/{id:[0-9]+}")
	u, _ := url.Parse("http://foo.example.com/api/events/12345")
	matches, matched := p.Match(u)
	if !matched {
		t.Fail()
	}
	if len(matches) != 2 {
		t.Fail()
	}
	if matches["subdomain"] != "foo" {
		t.Fail()
	}
	if matches["id"] != "12345" {
		t.Fail()
	}
}
