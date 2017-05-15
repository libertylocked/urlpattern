urlpattern
===
A library to match URL patterns, [gorilla/mux](https://github.com/gorilla/mux) style

- Extract variables from a URL
- Validate a URL against a pattern template
- Match host, path, path prefix, or query parameters

---
## Install
```
go get -u github.com/libertylocked/urlpattern
```

## Examples

### Matching paths
```go
p := urlpattern.NewPattern().
    Path("/api/events/{id:[0-9]+}")
u, _ := url.Parse("https://super.example.com/api/events/12345")
matches, matched := p.Match(u)
// matched: true
// matches: map[string]string{ "id": "12345" }
```

### Matching hosts and paths
```go
p := urlpattern.NewPattern().
    Host("{subdomain}.example.com").
    Path("/api/events/{id:[0-9]+}")
u, _ := url.Parse("https://super.example.com/api/events/12345")
matches, matched := p.Match(u)
// matched: true
// matches: map[string]string{ "subdomain": "super", "id": "12345" }
```

### Matching path prefixes
```go
p := urlpattern.NewPattern().
    PathPrefix("/api")
u, _ := url.Parse("https://super.example.com/api/events/12345")
matches, matched := p.Match(u)
// matched: true
// matches: map[string]string{ }
```

### Matching query params
```go
p := urlpattern.NewPattern().
    Queries("key", "{key:[0-9]+}")
u, _ := url.Parse("http://super.example.com/api/events/12345?key=42")
matches, matched := p.Match(u)
// matched: true
// matches: map[string]string{ "key": "42" }
```

### All together
```go
p := urlpattern.NewPattern().
    Host("{subdomain}.example.com").
    PathPrefix("/api").
    Path("/events/{id:[0-9]+}").
    Query("key", "{key}")
u, _ := url.Parse("http://super.example.com/api/events/12345?key=42")
matches, matched := p.Match(u)
// matched: true
// matches: map[string]string{ "subdomain": "super", "id": "12345", "key": "42" }
```
