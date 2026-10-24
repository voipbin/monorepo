package apiv1

import (
	"net/http"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
)

func Test_ApplyRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	ApplyRoutes(&engine.RouterGroup)

	got := map[string]bool{}
	for _, r := range engine.Routes() {
		got[r.Method+" "+r.Path] = true
	}

	want := []string{
		http.MethodPost + " /v1.0/billing/paddle",
		http.MethodPost + " /v1.0/emails/:target",
		http.MethodPost + " /v1.0/messages/:target",
	}
	for _, m := range []string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodHead,
		http.MethodOptions, http.MethodDelete, http.MethodConnect, http.MethodTrace,
	} {
		want = append(want, m+" /v1.0/conversation/*any")
	}

	for _, w := range want {
		if !got[w] {
			t.Errorf("route not registered: %s", w)
		}
	}

	if len(got) != len(want) {
		var keys []string
		for k := range got {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		t.Errorf("unexpected route set. expect: %d routes, got: %d: %v", len(want), len(got), keys)
	}
}
