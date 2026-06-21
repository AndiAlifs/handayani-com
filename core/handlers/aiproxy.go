package handlers

// Reverse proxy to the internal Python AI service (session analysis + RAG
// knowledge-sync). This service is the single public front door; it forwards
// these endpoints to AI_SERVICE_URL and returns 502 if that service is down
// (mirroring the old FastAPI gateway's behaviour).

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
)

func aiServiceURL() string {
	if v := os.Getenv("AI_SERVICE_URL"); v != "" {
		return v
	}
	return "http://localhost:8081"
}

func newAIProxy() *httputil.ReverseProxy {
	target, err := url.Parse(aiServiceURL())
	if err != nil {
		// Fall back to the default; url.Parse rarely fails for these values.
		target, _ = url.Parse("http://localhost:8081")
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	// The target has no path, so the default Director preserves req.URL.Path
	// verbatim (e.g. /api/sessions/5/analyze, /api/rag/knowledge-sync) — exactly
	// what the Python routes expect. Authorization is not hop-by-hop, so it is
	// forwarded automatically.
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"AI service unavailable"}`))
	}
	return proxy
}

var aiProxy = newAIProxy()

// ProxyToAI forwards the current request to the Python AI service unchanged.
func ProxyToAI(c *gin.Context) {
	aiProxy.ServeHTTP(c.Writer, c.Request)
}

// Health is a public liveness probe for the Go gateway.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
