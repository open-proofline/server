package main

import (
	"net/url"
	"strings"
)

func buildViewerURL(viewerBase, token string) string {
	return joinURL(cleanBaseURL(viewerBase), "/i/"+url.PathEscape(token))
}

func joinURL(base, path string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}

func cleanBaseURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}
