package router

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/popochoo/L7-ingress-controller/internal/config"
)

type RegexRoute struct {
	Pattern *regexp.Regexp
	Route   config.RouteConfig
}

type Router struct {
	exactRoutes map[string]config.RouteConfig
	regexRoutes []RegexRoute
	trie        *Trie
}

func NewRouter(cfg *config.Config) *Router {
	r := &Router{
		exactRoutes: make(map[string]config.RouteConfig),
		regexRoutes: make([]RegexRoute, 0),
		trie:        NewTrie(),
	}

	for _, route := range cfg.Routes {
		switch route.MatchType {
		case "exact":
			r.exactRoutes[route.Path] = route
		case "prefix":
			r.trie.Insert(route.Path, route)
		case "regex":
			expr, err := regexp.Compile(route.Path)
			if err != nil {
				continue
			}
			r.regexRoutes = append(r.regexRoutes, RegexRoute{
				Pattern: expr,
				Route:   route,
			})
		}
	}

	return r
}

func (r *Router) Match(req *http.Request) (config.RouteConfig, bool) {
	path := req.URL.Path
	method := req.Method

	if route, found := r.exactRoutes[path]; found {
		if r.methodAllowed(route.Methods, method) {
			return route, true
		}
	}

	if route, found := r.trie.Search(path); found {
		if r.methodAllowed(route.Methods, method) {
			return route, true
		}
	}

	for _, regRoute := range r.regexRoutes {
		if regRoute.Pattern.MatchString(path) {
			if r.methodAllowed(regRoute.Route.Methods, method) {
				return regRoute.Route, true
			}
		}
	}

	return config.RouteConfig{}, false
}

func (r *Router) methodAllowed(allowedMethods []string, currentMethod string) bool {
	if len(allowedMethods) == 0 {
		return true
	}

	for _, m := range allowedMethods {
		if strings.ToUpper(m) == strings.ToUpper(currentMethod) {
			return true
		}
	}

	return false
}
