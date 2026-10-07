package logmask

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func Test_Mask(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"accesskey in access log", `| 200 | POST     "/v1.0/x?accesskey=SECRET&foo=bar"` + "\n", `| 200 | POST     "/v1.0/x?accesskey=***&foo=bar"` + "\n"},
		{"token last param keeps closing quote", `GET     "/v1.0/x?foo=bar&token=SECRET"`, `GET     "/v1.0/x?foo=bar&token=***"`},
		{"token middle param", `"/x?a=1&token=SECRET&b=2"`, `"/x?a=1&token=***&b=2"`},
		{"repeated key", `"/x?token=A&token=B"`, `"/x?token=***&token=***"`},
		{"empty value", `"/x?token=&a=1"`, `"/x?token=***&a=1"`},
		{"no value is not a pair", `"/x?token&a=1"`, `"/x?token&a=1"`},
		{"percent-encoded key accesskey", `"/x?%61ccesskey=SECRET&a=1"`, `"/x?%61ccesskey=***&a=1"`},
		{"percent-encoded key token", `"/x?a=1&tok%65n=SECRET"`, `"/x?a=1&tok%65n=***"`},
		{"double-encoded key is not the credential", `"/x?%2561ccesskey=V"`, `"/x?%2561ccesskey=V"`},
		{"escaped quote in value", `"/x?token=AB\"CD&a=b"`, `"/x?token=***&a=b"`},
		{"escaped quote in value last", `"/x?token=AB\"CD"`, `"/x?token=***"`},
		{"raw quote in recovery dump", "GET /x?token=AB\"CD&a=b HTTP/1.1\r\n", "GET /x?token=***&a=b HTTP/1.1\r\n"},
		{"backslash before ampersand in raw dump (token)", "GET /x?a\\&token=SECRET HTTP/1.1\r\n", "GET /x?a\\&token=*** HTTP/1.1\r\n"},
		{"backslash before ampersand in raw dump (accesskey)", "GET /x?a\\&accesskey=SECRET HTTP/1.1\r\n", "GET /x?a\\&accesskey=*** HTTP/1.1\r\n"},
		{"backslash in quoted access log", `"/x?a\\&accesskey=SECRET"`, `"/x?a\\&accesskey=***"`},
		{"encoded quote value", `"/x?token=AB%22CD&a=b"`, `"/x?token=***&a=b"`},
		{"access_token untouched", `"/x?access_token=V&xtoken=W"`, `"/x?access_token=V&xtoken=W"`},
		{"other params untouched", `"/x?page_size=10&page_token=abc"`, `"/x?page_size=10&page_token=abc"`},
		{"case sensitive like c.Query", `"/x?Token=V&AccessKey=W"`, `"/x?Token=V&AccessKey=W"`},
		{"referer query masked too", "Referer: http://h/p?accesskey=R\r\n", "Referer: http://h/p?accesskey=***\r\n"},
		{"cookie single", "Cookie: token=SECRET\r\n", "Cookie: token=***\r\n"},
		{"cookie multiple keeps separators", "Cookie: a=b; token=X; accesskey=Y;c=d\r\n", "Cookie: a=b; token=***; accesskey=***;c=d\r\n"},
		{"cookie quoted value", `Cookie: token="X Y"` + "\r\n", "Cookie: token=***\r\n"},
		{"cookie lookalike names untouched", "Cookie: xtoken=1; Token=2; mytoken=3\r\n", "Cookie: xtoken=1; Token=2; mytoken=3\r\n"},
		{"token-looking text outside Cookie line untouched", "X-Note: token=V; accesskey=W\r\n", "X-Note: token=V; accesskey=W\r\n"},
		{"plain text", "[GIN-debug] Listening on :8080\n", "[GIN-debug] Listening on :8080\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Mask(tt.input); got != tt.want {
				t.Errorf("Mask()\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}

type failWriter struct{}

func (failWriter) Write(p []byte) (int, error) { return 0, errors.New("sink failed") }

func Test_Write(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	in := "GET /x?token=SECRET HTTP/1.1\n"
	n, err := w.Write([]byte(in))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(in) {
		t.Errorf("Write() n = %d, want original length %d", n, len(in))
	}
	if buf.String() != "GET /x?token=*** HTTP/1.1\n" {
		t.Errorf("Write() output = %q", buf.String())
	}

	if _, err := NewWriter(failWriter{}).Write([]byte("x")); err == nil {
		t.Errorf("Write() expected sink error to propagate")
	}
}

// Credential values are kept out of the lines that call runRequest, because
// the recovery dump prints source lines of the stack and would otherwise show
// the literal.
const (
	credA = "cred-value-aaaa"
	credB = "cred-value-bbbb"
)

func setDebugMode(t *testing.T) {
	t.Helper()
	prev := gin.Mode()
	gin.SetMode(gin.DebugMode)
	t.Cleanup(func() { gin.SetMode(prev) })
}

// runRequest sends a request through an engine wired like production and
// returns what the access logger and the recovery logger wrote.
//
// mask=true uses the public production entry point Middlewares; mask=false uses
// the unmasked wiring as the negative control.
func runRequest(t *testing.T, mask bool, target, cookie string, panicInHandler bool) (accessLog, recoveryLog string) {
	t.Helper()
	setDebugMode(t)

	var out, errOut bytes.Buffer
	app := gin.New()
	if mask {
		app.Use(Middlewares(nil, &out, &errOut)...)
	} else {
		app.Use(middlewares(nil, &out, &errOut, false)...)
	}
	app.GET("/v1.0/x", func(c *gin.Context) {
		if panicInHandler {
			panic("boom")
		}
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, target, nil)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	app.ServeHTTP(httptest.NewRecorder(), req)

	return out.String(), errOut.String()
}

func Test_Middlewares_AccessLog(t *testing.T) {
	access, _ := runRequest(t, true, "/v1.0/x?accesskey="+credA+"&foo=bar", "", false)

	if strings.Contains(access, credA) {
		t.Errorf("access log leaks the credential: %q", access)
	}
	if !strings.Contains(access, `"/v1.0/x?accesskey=***&foo=bar"`) {
		t.Errorf("access log format changed or not masked: %q", access)
	}
}

func Test_Middlewares_Recovery(t *testing.T) {
	_, rec := runRequest(t, true, "/v1.0/x?token="+credA, "accesskey="+credB, true)

	// Positive assertions: the dump was really printed, so the absence of the
	// secrets below is not vacuous.
	if !strings.Contains(rec, "Cookie:") || !strings.Contains(rec, "/v1.0/x?token=***") {
		t.Fatalf("recovery dump was not printed as expected: %q", rec)
	}
	if strings.Contains(rec, credA) || strings.Contains(rec, credB) {
		t.Errorf("recovery dump leaks a credential: %q", rec)
	}
}

// Test_Middlewares_NegativeControl proves the assertions above fail without the
// fix: with masking disabled the credentials do appear in both sinks.
func Test_Middlewares_NegativeControl(t *testing.T) {
	access, _ := runRequest(t, false, "/v1.0/x?accesskey="+credA+"&foo=bar", "", false)
	if !strings.Contains(access, credA) {
		t.Errorf("negative control: unmasked access log should contain the credential: %q", access)
	}

	_, rec := runRequest(t, false, "/v1.0/x?token="+credA, "accesskey="+credB, true)
	if !strings.Contains(rec, credA) || !strings.Contains(rec, credB) {
		t.Errorf("negative control: unmasked recovery dump should contain the credentials: %q", rec)
	}
}

func Test_Middlewares_NilWritersFallback(t *testing.T) {
	if got := len(Middlewares(nil, nil, nil)); got != 2 {
		t.Errorf("Middlewares() len = %d, want 2", got)
	}
}
