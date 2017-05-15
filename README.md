urlpattern
===
A library to match URL patterns, [gorilla/mux](https://github.com/gorilla/mux) style

- Extract variables from a URL
- Validate a URL against a pattern template
- Match host, path, or path prefix

---
## Install
```
go get -u github.com/libertylocked/urlpattern
```

## Examples

### Matching paths
```go
p := urlpattern.NewPattern().
    Path("/api/event/{id:[0-9]+}")
u, _ := url.Parse("https://super.example.com/api/event/12345")
matches, matched := p.Match(u)
// matched: true
// matches: map[string]string{ "id": "12345" }
```

### Matching hosts and paths
```go
p := urlpattern.NewPattern().
    Host("{subdomain}.example.com").
    Path("/api/event/{id:[0-9]+}")
u, _ := url.Parse("https://super.example.com/api/event/12345")
matches, matched := p.Match(u)
// matched: true
// matches: map[string]string{ "subdomain": "super", "id": "12345" }
```

### Matching path prefixes
```go
p := urlpattern.NewPattern().
    PathPrefix("/api")
u, _ := url.Parse("https://super.example.com/api/event/12345")
matches, matched := p.Match(u)
// matched: true
// matches: map[string]string{ }
```

### All together
```go
p := NewPattern().
    Host("{subdomain}.example.com").
    PathPrefix("/api").
    Path("/events/{id:[0-9]+}")
u, _ := url.Parse("http://super.example.com/api/events/12345")
matches, matched := p.Match(u)
// matched: true
// matches: map[string]string{ "subdomain": "foo", "id": "12345" }
```
