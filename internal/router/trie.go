package router

import (
	"strings"

	"github.com/popochoo/L7-ingress-controller/internal/config"
)

type TrieNode struct {
	children map[string]*TrieNode
	isRoute  bool
	route    config.RouteConfig
}

type Trie struct {
	root *TrieNode
}

func NewTrie() *Trie {
	return &Trie{
		root: &TrieNode{children: make(map[string]*TrieNode)},
	}
}

func (t *Trie) Insert(path string, route config.RouteConfig) {
	segments := parsePath(path)
	current := t.root

	for _, segment := range segments {
		if _, exists := current.children[segment]; !exists {
			current.children[segment] = &TrieNode{children: make(map[string]*TrieNode)}
		}
		current = current.children[segment]
	}
	current.isRoute = true
	current.route = route
}

func (t *Trie) Search(path string) (config.RouteConfig, bool) {
	segments := parsePath(path)
	current := t.root

	var lastMatchedRoute config.RouteConfig
	var found bool

	for _, segment := range segments {
		next, exists := current.children[segment]
		if !exists {
			break
		}
		current = next
		if current.isRoute {
			lastMatchedRoute = current.route
			found = true
		}
	}

	return lastMatchedRoute, found
}

func parsePath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}
