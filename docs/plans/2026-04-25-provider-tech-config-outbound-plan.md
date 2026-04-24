# Provider tech prefix/postfix/headers — apply on outbound dial

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `Provider.TechPrefix`, `Provider.TechPostfix`, and `Provider.TechHeaders` actually affect outbound tel-type calls in `bin-call-manager`. Today they are stored and API-editable but silently dropped in `getDialURITel`.

**Architecture:** One focused patch in `bin-call-manager/pkg/callhandler/`. Add a `mergeTechHeaders` helper with a reserved-key denylist and CRLF/invalid-char sanitization. Change `getDialURITel` to return `(uri, techHeaders, error)`; update the dispatcher `getDialURI` and its two peers similarly. In `createChannelOutgoing`, seed `channelVariables` with the tech headers before the system-set transport/caller-ID functions run — that ordering gives system headers automatic collision-wins behavior for keys the system code path actually writes, and the denylist covers keys the system only writes conditionally.

**Tech Stack:** Go 1.x, gomock (`go.uber.org/mock`), table-driven tests, logrus. Linked design doc: [2026-04-25-provider-tech-config-outbound-design.md](./2026-04-25-provider-tech-config-outbound-design.md).

**Worktree:** `/home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-apply-provider-tech-config-on-outbound`
**Branch:** `NOJIRA-apply-provider-tech-config-on-outbound`

**Scope ground rules:**
- All file paths below are relative to the worktree root.
- Only `bin-call-manager` is touched — no OpenAPI, RST, DB schema, or cross-service changes. The `WebhookMessage` already exposes all three fields; RST already describes the behavior correctly.
- Every task ends with a green test run before committing.
- Full verification workflow runs only once at the very end (Task 7) per monorepo convention — but `go test ./...` runs after each code change within each task.

---

## Task 1: Add the `mergeTechHeaders` helper with tests

Write the helper that enforces the reserved-key denylist and CRLF/char sanitization. TDD: test first.

**Files:**
- Create: `bin-call-manager/pkg/callhandler/tech_headers.go`
- Create: `bin-call-manager/pkg/callhandler/tech_headers_test.go`

### Step 1.1: Write the failing test file

Create `bin-call-manager/pkg/callhandler/tech_headers_test.go` with the full table-driven test. The helper takes an existing channel-variables map (`dst`), a raw `tech_headers` map from the Provider (`src`), and a `*logrus.Entry` for warnings; it mutates `dst` in place and returns `(applied, skipped int)`.

```go
package callhandler

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func Test_mergeTechHeaders(t *testing.T) {

	tests := []struct {
		name string

		dst map[string]string
		src map[string]string

		expectDst     map[string]string
		expectApplied int
		expectSkipped int
	}{
		{
			"nil src leaves dst unchanged",

			map[string]string{"CALLERID(num)": "+820000000000"},
			nil,

			map[string]string{"CALLERID(num)": "+820000000000"},
			0,
			0,
		},
		{
			"empty src leaves dst unchanged",

			map[string]string{"CALLERID(num)": "+820000000000"},
			map[string]string{},

			map[string]string{"CALLERID(num)": "+820000000000"},
			0,
			0,
		},
		{
			"normal header gets wrapped and applied",

			map[string]string{},
			map[string]string{"X-Carrier-Auth": "token-abc"},

			map[string]string{
				"PJSIP_HEADER(add,X-Carrier-Auth)": "token-abc",
			},
			1,
			0,
		},
		{
			"empty key is skipped",

			map[string]string{},
			map[string]string{"": "whatever"},

			map[string]string{},
			0,
			1,
		},
		{
			"key with CRLF is skipped",

			map[string]string{},
			map[string]string{"X-Bad\r\nInject": "v"},

			map[string]string{},
			0,
			1,
		},
		{
			"key with parenthesis is skipped (blocks PJSIP_HEADER pre-wrap)",

			map[string]string{},
			map[string]string{"PJSIP_HEADER(add,X)": "v"},

			map[string]string{},
			0,
			1,
		},
		{
			"key with comma is skipped",

			map[string]string{},
			map[string]string{"X,Evil": "v"},

			map[string]string{},
			0,
			1,
		},
		{
			"reserved wrapped key P-Asserted-Identity is skipped",

			map[string]string{},
			map[string]string{"P-Asserted-Identity": "<tel:+100>"},

			map[string]string{},
			0,
			1,
		},
		{
			"reserved wrapped key Privacy is skipped",

			map[string]string{},
			map[string]string{"Privacy": "id"},

			map[string]string{},
			0,
			1,
		},
		{
			"reserved wrapped key SDP-Transport is skipped",

			map[string]string{},
			map[string]string{"VBOUT-SDP_Transport": "RTP/AVP"},

			map[string]string{},
			0,
			1,
		},
		{
			"reserved raw key CALLERID(name) is skipped",

			map[string]string{},
			map[string]string{"CALLERID(name)": "foo"},

			map[string]string{},
			0,
			1,
		},
		{
			"value with CR is skipped",

			map[string]string{},
			map[string]string{"X-Header": "ok\rbad"},

			map[string]string{},
			0,
			1,
		},
		{
			"value with LF is skipped",

			map[string]string{},
			map[string]string{"X-Header": "ok\nbad"},

			map[string]string{},
			0,
			1,
		},
		{
			"mixed valid and invalid — valid applied, invalid counted skipped",

			map[string]string{},
			map[string]string{
				"X-Good":              "ok",
				"":                    "drop-empty-key",
				"X-Bad\r\nInject":     "drop-bad-key",
				"P-Asserted-Identity": "drop-reserved",
			},

			map[string]string{
				"PJSIP_HEADER(add,X-Good)": "ok",
			},
			1,
			3,
		},
		{
			"pre-existing dst key is overwritten by tech header (merge semantics; system fns re-overwrite later in createChannelOutgoing)",

			map[string]string{
				"PJSIP_HEADER(add,X-Carrier-Auth)": "old",
			},
			map[string]string{
				"X-Carrier-Auth": "new",
			},

			map[string]string{
				"PJSIP_HEADER(add,X-Carrier-Auth)": "new",
			},
			1,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logrus.WithField("test", tt.name)

			applied, skipped := mergeTechHeaders(tt.dst, tt.src, log)

			if applied != tt.expectApplied {
				t.Errorf("Wrong applied count. expect: %d, got: %d", tt.expectApplied, applied)
			}
			if skipped != tt.expectSkipped {
				t.Errorf("Wrong skipped count. expect: %d, got: %d", tt.expectSkipped, skipped)
			}

			if len(tt.dst) != len(tt.expectDst) {
				t.Errorf("Wrong dst size. expect: %d, got: %d. dst=%v", len(tt.expectDst), len(tt.dst), tt.dst)
			}
			for k, v := range tt.expectDst {
				if got, ok := tt.dst[k]; !ok || got != v {
					t.Errorf("Wrong dst entry. key=%s expect=%q got=%q (present=%v)", k, v, got, ok)
				}
			}
		})
	}
}
```

### Step 1.2: Run the test — expect a compile failure

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_mergeTechHeaders`
Expected: compile error — `undefined: mergeTechHeaders`. If anything else fails, stop and read carefully before proceeding.

### Step 1.3: Implement the helper

Create `bin-call-manager/pkg/callhandler/tech_headers.go`:

```go
package callhandler

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// reservedTechHeaderKeys lists channel-variable keys that operator-supplied
// tech_headers are forbidden from setting. The set covers:
//   - headers the system writes conditionally (P-Asserted-Identity / Privacy
//     are only set for anonymous calls, so ordering alone cannot protect them),
//   - headers the system writes unconditionally (SDP-Transport, CALLERID(*)),
//   - internal trace headers (VB-CALL-ID / VB-CONFBRIDGE-ID) whose values must
//     match call-manager's own UUIDs for downstream correlation.
//
// Keys are checked both in their raw form (for CALLERID/PJSIP_HEADER entries)
// and after PJSIP_HEADER(add,...) wrapping (for SIP header names).
var reservedTechHeaderKeys = map[string]struct{}{
	"PJSIP_HEADER(add,P-Asserted-Identity)":     {},
	"PJSIP_HEADER(add,Privacy)":                 {},
	"PJSIP_HEADER(add,VBOUT-SDP_Transport)":     {},
	"PJSIP_HEADER(add,VB-CALL-ID)":              {},
	"PJSIP_HEADER(add,VB-CONFBRIDGE-ID)":        {},
	"PJSIP_HEADER(add,VB-DIRECTION)":            {},
	"CALLERID(name)":                            {},
	"CALLERID(num)":                             {},
	"CALLERID(pres)":                            {},
}

// mergeTechHeaders copies sanitized entries from src (raw operator-supplied
// tech_headers) into dst (Asterisk channel variables), wrapping each key
// with PJSIP_HEADER(add,...) so Asterisk attaches the header to the outgoing
// INVITE.
//
// Entries are skipped and logged as Warn when:
//   - key is empty
//   - key contains \r, \n, (, ), or ,
//   - wrapped key matches reservedTechHeaderKeys
//   - raw key itself matches reservedTechHeaderKeys (covers CALLERID(*) attempts)
//   - value contains \r or \n (CRLF injection defense)
//
// Skipped entries never fail the call — the rest of the tech config and the
// call itself proceed.
//
// Returns counts of applied and skipped entries for a single summary log at
// the call site.
func mergeTechHeaders(dst map[string]string, src map[string]string, log *logrus.Entry) (applied int, skipped int) {
	for k, v := range src {
		if k == "" {
			log.Warnf("Skipping tech_header with empty key.")
			skipped++
			continue
		}
		if strings.ContainsAny(k, "\r\n(),") {
			log.Warnf("Skipping tech_header with invalid key char. key=%q", k)
			skipped++
			continue
		}
		if _, reserved := reservedTechHeaderKeys[k]; reserved {
			log.Warnf("Skipping tech_header that collides with system-reserved key. key=%q", k)
			skipped++
			continue
		}
		if strings.ContainsAny(v, "\r\n") {
			log.Warnf("Skipping tech_header with CRLF in value. key=%q", k)
			skipped++
			continue
		}

		varKey := "PJSIP_HEADER(add," + k + ")"
		if _, reserved := reservedTechHeaderKeys[varKey]; reserved {
			log.Warnf("Skipping tech_header that collides with system-reserved header. key=%q", k)
			skipped++
			continue
		}

		dst[varKey] = v
		applied++
	}
	return applied, skipped
}
```

### Step 1.4: Run the test — expect PASS

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_mergeTechHeaders`
Expected: all 14 subtests pass. If any fail, the helper logic disagrees with the test table — fix the helper, not the tests (unless the test itself is demonstrably wrong).

### Step 1.5: Commit

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-apply-provider-tech-config-on-outbound
git add bin-call-manager/pkg/callhandler/tech_headers.go bin-call-manager/pkg/callhandler/tech_headers_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-apply-provider-tech-config-on-outbound

Add mergeTechHeaders helper with reserved-key denylist and CRLF/
invalid-char sanitization. Helper will be wired into createChannelOutgoing
in a follow-up task once getDialURITel returns tech_headers.

- bin-call-manager: Add pkg/callhandler/tech_headers.go with mergeTechHeaders
  helper and reservedTechHeaderKeys set
- bin-call-manager: Add pkg/callhandler/tech_headers_test.go covering empty/
  nil src, normal pass-through, empty/invalid-char/reserved key skips, CRLF
  value skip, CALLERID reserved raw-key skip, mixed valid/invalid, and merge-
  overwrite semantics
EOF
)"
```

---

## Task 2: Update `getDialURITel` to apply prefix/postfix and return tech headers

Change the signature to `(string, map[string]string, error)`, wrap the user part with prefix/postfix, and return the raw `tech_headers` for the caller to merge. Test first.

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go:316-346` (function `getDialURITel` only — do NOT touch other functions yet)
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call_test.go:1028-1100` (test `Test_getDialURI_Tel`)

### Step 2.1: Extend `Test_getDialURI_Tel` to cover prefix/postfix/headers and new return shape

Open `bin-call-manager/pkg/callhandler/outgoing_call_test.go` and **replace** `Test_getDialURI_Tel` (lines 1028–1100) with:

```go
func Test_getDialURI_Tel(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		responseProvider *rmprovider.Provider

		expectProviderID uuid.UUID
		expectRes        string
		expectTechHdrs   map[string]string
	}{
		{
			"no tech config (backwards compat)",

			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("04e5d530-5d96-11ed-bbc8-cfb95f6d6085"),
					CustomerID: uuid.FromStringOrNil("f7a14b8c-534c-11ed-9fb1-c7c376f2730b"),
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821121656521",
				},
				DialrouteID: uuid.FromStringOrNil("ae237dd0-5db1-11ed-97c4-a7ff6f170fbb"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("ae237dd0-5db1-11ed-97c4-a7ff6f170fbb"),
						ProviderID: uuid.FromStringOrNil("8730a3da-5350-11ed-aa47-7f44741127c1"),
					},
				},
			},

			&rmprovider.Provider{
				ID:       uuid.FromStringOrNil("ae237dd0-5db1-11ed-97c4-a7ff6f170fbb"),
				Hostname: "sip.telnyx.com",
			},

			uuid.FromStringOrNil("8730a3da-5350-11ed-aa47-7f44741127c1"),
			"pjsip/call-out/sip:+821121656521@sip.telnyx.com;transport=udp",
			nil,
		},
		{
			"prefix only",

			&call.Call{
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "15551234"},
				DialrouteID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
				Dialroutes: []rmroute.Route{
					{ID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
						ProviderID: uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000001")},
				},
			},

			&rmprovider.Provider{
				Hostname:   "carrier.example.com",
				TechPrefix: "0011",
			},

			uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000001"),
			"pjsip/call-out/sip:001115551234@carrier.example.com;transport=udp",
			nil,
		},
		{
			"postfix only",

			&call.Call{
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234"},
				DialrouteID: uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000002"),
				Dialroutes: []rmroute.Route{
					{ID: uuid.FromStringOrNil("a0000002-0000-0000-0000-000000000002"),
						ProviderID: uuid.FromStringOrNil("b0000002-0000-0000-0000-000000000002")},
				},
			},

			&rmprovider.Provider{
				Hostname:    "carrier.example.com",
				TechPostfix: "#",
			},

			uuid.FromStringOrNil("b0000002-0000-0000-0000-000000000002"),
			"pjsip/call-out/sip:+15551234#@carrier.example.com;transport=udp",
			nil,
		},
		{
			"prefix and postfix both",

			&call.Call{
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234"},
				DialrouteID: uuid.FromStringOrNil("a0000003-0000-0000-0000-000000000003"),
				Dialroutes: []rmroute.Route{
					{ID: uuid.FromStringOrNil("a0000003-0000-0000-0000-000000000003"),
						ProviderID: uuid.FromStringOrNil("b0000003-0000-0000-0000-000000000003")},
				},
			},

			&rmprovider.Provider{
				Hostname:    "carrier.example.com",
				TechPrefix:  "0011",
				TechPostfix: "#",
			},

			uuid.FromStringOrNil("b0000003-0000-0000-0000-000000000003"),
			"pjsip/call-out/sip:0011+15551234#@carrier.example.com;transport=udp",
			nil,
		},
		{
			"headers only — returned raw (unsanitized), caller sanitizes via mergeTechHeaders",

			&call.Call{
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551234"},
				DialrouteID: uuid.FromStringOrNil("a0000004-0000-0000-0000-000000000004"),
				Dialroutes: []rmroute.Route{
					{ID: uuid.FromStringOrNil("a0000004-0000-0000-0000-000000000004"),
						ProviderID: uuid.FromStringOrNil("b0000004-0000-0000-0000-000000000004")},
				},
			},

			&rmprovider.Provider{
				Hostname: "carrier.example.com",
				TechHeaders: map[string]string{
					"X-Carrier-Auth": "tok-abc",
				},
			},

			uuid.FromStringOrNil("b0000004-0000-0000-0000-000000000004"),
			"pjsip/call-out/sip:+15551234@carrier.example.com;transport=udp",
			map[string]string{"X-Carrier-Auth": "tok-abc"},
		},
		{
			"all three together",

			&call.Call{
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "15551234"},
				DialrouteID: uuid.FromStringOrNil("a0000005-0000-0000-0000-000000000005"),
				Dialroutes: []rmroute.Route{
					{ID: uuid.FromStringOrNil("a0000005-0000-0000-0000-000000000005"),
						ProviderID: uuid.FromStringOrNil("b0000005-0000-0000-0000-000000000005")},
				},
			},

			&rmprovider.Provider{
				Hostname:    "carrier.example.com",
				TechPrefix:  "0011",
				TechPostfix: "#",
				TechHeaders: map[string]string{"X-Route-Hint": "premium"},
			},

			uuid.FromStringOrNil("b0000005-0000-0000-0000-000000000005"),
			"pjsip/call-out/sip:001115551234#@carrier.example.com;transport=udp",
			map[string]string{"X-Route-Hint": "premium"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.expectProviderID).Return(tt.responseProvider, nil)

			res, techHdrs, err := h.getDialURI(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}

			if len(techHdrs) != len(tt.expectTechHdrs) {
				t.Errorf("Wrong techHdrs size. expect: %d, got: %d. techHdrs=%v", len(tt.expectTechHdrs), len(techHdrs), techHdrs)
			}
			for k, v := range tt.expectTechHdrs {
				if got, ok := techHdrs[k]; !ok || got != v {
					t.Errorf("Wrong techHdrs entry. key=%s expect=%q got=%q (present=%v)", k, v, got, ok)
				}
			}
		})
	}
}
```

Note: this test calls `h.getDialURI` (the dispatcher), not `getDialURITel` directly — matching the existing test style. The dispatcher's signature must therefore change at the same time as `getDialURITel`. Step 2.3 updates both atomically.

### Step 2.2: Run the test — expect compile failure

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_getDialURI_Tel`
Expected: compile error — `h.getDialURI(...)` returns 2 values but test assigns 3. If the error is different, re-read step 2.1 for a typo before proceeding.

### Step 2.3: Rewrite `getDialURITel` and the dispatcher to new signatures

In `bin-call-manager/pkg/callhandler/outgoing_call.go`, **replace the entire block from line 316 through line 408** with the following. This updates `getDialURITel`, `getDialURISIP`, `getDialURISIPDirect`, and `getDialURI` together so the file stays compilable.

```go
// getDialURITel returns dial uri and provider-supplied tech_headers for the
// given tel type destination. Prefix/postfix from the Provider wrap the
// user part of the URI; tech_headers are returned raw for the caller to
// merge via mergeTechHeaders (so sanitization and reserved-key enforcement
// happen next to the channel-variable assembly).
func (h *callHandler) getDialURITel(ctx context.Context, c *call.Call) (string, map[string]string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "getDialURITel",
		"call_id": c.ID,
	})

	providerID := uuid.Nil
	for _, dialroute := range c.Dialroutes {
		if dialroute.ID == c.DialrouteID {
			providerID = dialroute.ProviderID
			break
		}
	}

	if providerID == uuid.Nil {
		log.Debugf("No available dialroute left.")
		return "", nil, fmt.Errorf("no available dialroute left")
	}

	// get provider info
	pr, err := h.reqHandler.RouteV1ProviderGet(ctx, providerID)
	if err != nil {
		log.Errorf("Could not get provider info. err: %v", err)
		return "", nil, err
	}

	userPart := pr.TechPrefix + c.Destination.Target + pr.TechPostfix
	res := fmt.Sprintf("pjsip/%s/sip:%s@%s;transport=%s", pjsipEndpointOutgoing, userPart, pr.Hostname, constTransportUDP)

	return res, pr.TechHeaders, nil
}

// getDialURISIP returns dial uri of the given sip type destination.
func (h *callHandler) getDialURISIP(ctx context.Context, c *call.Call) (string, map[string]string, error) {
	endpoint := c.Destination.Target
	if !strings.HasPrefix(c.Destination.Target, "sip:") && !strings.HasPrefix(c.Destination.Target, "sips:") {
		endpoint = "sip:" + endpoint
	}

	res := fmt.Sprintf("pjsip/%s/%s", pjsipEndpointOutgoing, endpoint)
	return res, nil, nil
}

// getDialURISIPDirect returns dial uri of the given sip type destination via the direct endpoint.
func (h *callHandler) getDialURISIPDirect(ctx context.Context, c *call.Call) (string, map[string]string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "getDialURISIPDirect",
		"destination_target": c.Destination.Target,
	})

	endpointTarget := c.Destination.Target
	if !strings.HasPrefix(c.Destination.Target, "sip:") && !strings.HasPrefix(c.Destination.Target, "sips:") {
		endpointTarget = "sip:" + endpointTarget
	}
```

**CRITICAL:** The block above ends mid-function at `endpointTarget := ...`. The remaining body of `getDialURISIPDirect` (from the current line 370 onward through line 388) must be preserved verbatim; only the function signature changes. Do the same for `getDialURI` (current lines 389–408) — change the signature and the three `return` statements inside to return the third value.

After editing, the dispatcher block should read:

```go
// getDialURI returns the given destination address's dial URI and optional
// provider tech_headers for Asterisk's dialing. tech_headers is nil for
// non-provider paths.
func (h *callHandler) getDialURI(ctx context.Context, c *call.Call) (string, map[string]string, error) {

	switch c.Destination.Type {
	case commonaddress.TypeTel:
		return h.getDialURITel(ctx, c)

	case commonaddress.TypeSIP:
		if strings.Contains(c.Destination.Target, "transport=ws") {
			return h.getDialURISIPDirect(ctx, c)
		}
		return h.getDialURISIP(ctx, c)

	default:
		return "", nil, fmt.Errorf("unsupported address type for get dial uri")
	}
}
```

Do NOT touch `createChannelOutgoing` yet — it still calls the old 2-return `getDialURI`, and will be fixed in Task 3. The build WILL break at that call site; that is expected and Task 3 fixes it.

### Step 2.4: Confirm `getDialURITel` test passes (ignore other compile errors)

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_getDialURI_Tel`

Expected outcome is one of:
1. All 6 subtests pass (if the rest of the package still compiles).
2. The package fails to compile at `createChannelOutgoing` or `Test_getDialURI_SIP` / `Test_getDialURI_error` / `Test_getDialURISIP` / `Test_getDialURISIPDirect` / `Test_createChannel` because they still assume the 2-return signature.

If outcome (2), that is fine — proceed to Task 3 immediately; those call sites are fixed there. Do NOT commit Task 2 in isolation; bundle the commit with Task 3 so the tree is always buildable at each committed revision.

---

## Task 3: Update `createChannelOutgoing` and remaining tests for the 3-return signature

Wire tech headers into `createChannelOutgoing` via `mergeTechHeaders`, then update the other existing tests (`Test_getDialURI_SIP`, `Test_getDialURI_error`, `Test_getDialURISIP`, `Test_getDialURISIPDirect`, `Test_createChannel`) to the new signatures.

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call.go:546-589` (`createChannelOutgoing`)
- Modify: `bin-call-manager/pkg/callhandler/outgoing_call_test.go` at lines 1102–1171 (`Test_getDialURI_SIP`), 1173–1239 (`Test_getDialURI_error`), 2652–2710 (`Test_getDialURISIP`), 2710–2765 (`Test_getDialURISIPDirect`), 1241–1325 (`Test_createChannel`)

### Step 3.1: Update `createChannelOutgoing` to merge tech headers

In `bin-call-manager/pkg/callhandler/outgoing_call.go`, **replace** the body of `createChannelOutgoing` (lines ~553–589, starting at `dialURI, err := h.getDialURI(...)`) with:

```go
	// get dial uri and provider-supplied tech headers
	dialURI, techHeaders, err := h.getDialURI(ctx, c)
	if err != nil {
		log.Errorf("Could not create a destination endpoint. err: %v", err)
		return err
	}

	// set channel variables — tech_headers first so system-set headers
	// (transport, CALLERID, PAI when anonymous) overwrite on collision.
	// mergeTechHeaders additionally enforces the reserved-key denylist.
	channelVariables := map[string]string{}
	techApplied, techSkipped := mergeTechHeaders(channelVariables, techHeaders, log)

	transport := getDestinationTransport(dialURI)
	setChannelVariableTransport(channelVariables, transport)
	anonymous := c.Data[call.DataTypeAnonymous] == "true"
	if err := setChannelVariablesCallerID(channelVariables, c, anonymous); err != nil {
		log.Errorf("Could not set caller ID variables. err: %v", err)
		return err
	}

	if techApplied > 0 || techSkipped > 0 {
		log.Infof("Applied provider tech config. headers_applied=%d headers_skipped=%d",
			techApplied, techSkipped)
	}

	log.Debugf("Endpoint detail. endpoint_destination: %s, variables: %v, anonymous: %v", dialURI, channelVariables, anonymous)

	// set app args
	appArgs := fmt.Sprintf("%s=%s,%s=%s,%s=%s,%s=%s,%s=%s",
		channel.StasisDataTypeContextType, channel.ContextTypeCall,
		channel.StasisDataTypeContext, channel.ContextCallOutgoing,
		channel.StasisDataTypeCallID, c.ID,
		channel.StasisDataTypeTransport, transport,
		channel.StasisDataTypeDirection, channel.DirectionOutgoing,
	)

	// create a channel
	tmp, err := h.channelHandler.StartChannel(ctx, requesthandler.AsteriskIDCall, c.ChannelID, appArgs, dialURI, "", "", "", channelVariables)
	if err != nil {
		log.Errorf("Could not create a channel for outgoing call. err: %v", err)
		return err
	}
	log.WithField("channel", tmp).Debugf("Created a new channel. channel_id: %s", tmp.ID)

	return nil
}
```

Tech headers are seeded into `channelVariables` **before** `setChannelVariableTransport` and `setChannelVariablesCallerID` run. For keys those functions write, the later write wins — that is the "system wins" guarantee for unconditionally-written keys. For keys they only write conditionally (PAI/Privacy on anonymous calls, which aren't in the non-anonymous write path), `mergeTechHeaders`'s reserved-key check still blocks the collision. Both layers together satisfy the Q3 "system wins" contract.

### Step 3.2: Update remaining tests to the 3-return signature

Do each of the following edits in `bin-call-manager/pkg/callhandler/outgoing_call_test.go`. These are mechanical — only the return destructuring changes, not the test logic.

1. `Test_getDialURI_SIP` (around line 1161): change `res, err := h.getDialURI(ctx, tt.c)` to `res, _, err := h.getDialURI(ctx, tt.c)`.
2. `Test_getDialURI_error` (around line 1233): change `_, err := h.getDialURI(ctx, tt.call)` to `_, _, err := h.getDialURI(ctx, tt.call)`.
3. `Test_getDialURISIP` (around line 2697): change `res, err := h.getDialURISIP(ctx, tt.call)` to `res, _, err := h.getDialURISIP(ctx, tt.call)`.
4. `Test_getDialURISIPDirect` (around line 2755): change `res, err := h.getDialURISIPDirect(ctx, tt.call)` to `res, _, err := h.getDialURISIPDirect(ctx, tt.call)`.
5. `Test_createChannel` at lines 1241–1325: no signature changes needed (this test calls `createChannelOutgoing`, not `getDialURI*`), but the test's existing case has empty `TechPrefix`/`TechPostfix`/`TechHeaders` on the mock provider, so the asserted `expectDialURI` and `expectVariables` must still match. Since `pr.TechPrefix + c.Destination.Target + pr.TechPostfix` with empty prefix/postfix equals the target, and nil `TechHeaders` yields zero applied/skipped (no log, no new channel variables), the existing expected values at lines 1289–1294 are already correct. Verify by re-running the test — no edit needed.

### Step 3.3: Run the full callhandler test suite — expect PASS

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/`

Expected: every test in `pkg/callhandler/` passes, including the new `Test_mergeTechHeaders` and the updated `Test_getDialURI_Tel`. If `Test_createChannel` fails, re-check Step 3.1 — the `channelVariables` map must still equal the existing expected map (empty tech config → zero tech headers applied → no extra entries).

### Step 3.4: Add one new `Test_createChannel` case that exercises tech config

At the end of the `tests := []struct{...}{...}` slice in `Test_createChannel` (inside the `}` at line ~1295, before the closing `}` of the slice), append the following case. This is the end-to-end integration test for the whole feature.

```go
		{
			name: "provider tech config applied — prefix, postfix, header",

			call: &call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000002"),
				},

				ChannelID: "c1c1c1c1-0000-0000-0000-000000000003",
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "15551234",
				},
				DialrouteID: uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000004"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000004"),
						ProviderID: uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000005"),
					},
				},
			},

			responseProvider: &rmprovider.Provider{
				ID:          uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000005"),
				Hostname:    "carrier.example.com",
				TechPrefix:  "0011",
				TechPostfix: "#",
				TechHeaders: map[string]string{"X-Route-Hint": "premium"},
			},

			expectProviderID: uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000005"),
			expectArgs:       "context_type=call,context=call-out,call_id=c1c1c1c1-0000-0000-0000-000000000001,transport=udp,direction=outgoing",
			expectDialURI:    "pjsip/call-out/sip:001115551234#@carrier.example.com;transport=udp",
			expectVariables: map[string]string{
				"PJSIP_HEADER(add,X-Route-Hint)":                         "premium",
				"CALLERID(name)":                                         "",
				"CALLERID(num)":                                          "+821100000002",
				"PJSIP_HEADER(add," + common.SIPHeaderSDPTransport + ")": "RTP/AVP",
			},
		},
```

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_createChannel`
Expected: both subtests (`normal` and `provider tech config applied — prefix, postfix, header`) pass.

### Step 3.5: Commit

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-apply-provider-tech-config-on-outbound
git add bin-call-manager/pkg/callhandler/outgoing_call.go bin-call-manager/pkg/callhandler/outgoing_call_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-apply-provider-tech-config-on-outbound

Apply provider tech_prefix, tech_postfix, and tech_headers on outbound
tel-type calls. Provider fields that were previously stored, editable via
API, and documented in RST were silently dropped in getDialURITel; this
change wraps the SIP URI user part with tech_prefix/tech_postfix and
attaches tech_headers as PJSIP_HEADER(add,...) channel variables, with the
reserved-key denylist from mergeTechHeaders protecting correctness/
security-critical system headers from operator override.

- bin-call-manager: Change getDialURITel, getDialURI, getDialURISIP, and
  getDialURISIPDirect return signatures to (string, map[string]string,
  error); tech_headers is nil for non-provider paths
- bin-call-manager: Embed TechPrefix/TechPostfix in the SIP URI user part in
  getDialURITel
- bin-call-manager: Seed tech_headers into channelVariables via
  mergeTechHeaders in createChannelOutgoing before system-set transport and
  caller-ID variables, so system keys win on collision
- bin-call-manager: Add Info log summarizing applied/skipped tech config
- bin-call-manager: Extend Test_getDialURI_Tel with prefix-only, postfix-
  only, both, headers-only, and all-three cases
- bin-call-manager: Add tech-config subtest to Test_createChannel covering
  end-to-end dial URI and channel variable assembly
- bin-call-manager: Update Test_getDialURI_SIP, Test_getDialURI_error,
  Test_getDialURISIP, Test_getDialURISIPDirect for the 3-return signature
EOF
)"
```

---

## Task 4: Verify the provider-call test endpoint inherits the fix

Finding 5 in the design doc requires a grep-level check that the provider-call verification endpoint reaches `getDialURITel`. No code change is expected.

### Step 4.1: Trace the provider-call path

Run:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-apply-provider-tech-config-on-outbound
grep -rn "ProviderCallCreate\|RouteV1ProviderCallCreate" bin-route-manager/pkg/ bin-call-manager/pkg/ | head -20
```

Confirm: `bin-route-manager`'s providercall handler ultimately creates a Call via `bin-call-manager`'s `CreateCallsOutgoing` (which calls `createChannelOutgoing` → `getDialURI` → `getDialURITel`). If the chain terminates at `getDialURITel`, the fix is inherited for free; note in the PR description.

If the trace reveals a separate dial URI builder used only for provider-call verification: STOP and surface the finding to the user before continuing. Do not expand scope without approval.

### Step 4.2: No commit

Nothing to commit in this task — it's a verification-only step whose output is reflected in the PR description body.

---

## Task 5: Full monorepo verification on bin-call-manager

Run the mandatory verification workflow before pushing. Root `CLAUDE.md` §"CRITICAL: Before Committing Changes" is explicit: all five steps, no skipping.

### Step 5.1: Run the full verification workflow

Run from the worktree:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-apply-provider-tech-config-on-outbound/bin-call-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: every step exits 0. If `go mod tidy` changed `go.mod` or `go.sum`, stage those changes. `go mod vendor` must succeed but the `vendor/` directory is gitignored — do NOT `git add -f` it.

### Step 5.2: Commit any `go.mod` / `go.sum` updates (only if changed)

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-apply-provider-tech-config-on-outbound
git status --porcelain bin-call-manager/go.mod bin-call-manager/go.sum
# If the previous command prints changes:
git add bin-call-manager/go.mod bin-call-manager/go.sum
git commit -m "$(cat <<'EOF'
NOJIRA-apply-provider-tech-config-on-outbound

- bin-call-manager: Run go mod tidy after callhandler changes
EOF
)"
```

If `go.mod` and `go.sum` are unchanged (no new imports — this is the expected case since the helper only adds `strings` and `logrus` which are already imported in the package), skip the commit. `strings` and `github.com/sirupsen/logrus` are both already present in `outgoing_call.go`, so no new transitive dep should appear.

### Step 5.3: Run lint one more time to catch anything `go generate` rewrote

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-apply-provider-tech-config-on-outbound/bin-call-manager && golangci-lint run -v --timeout 5m`
Expected: exit 0 with no issues. If generated mocks changed (`pkg/*/mock_*.go`), they should be committed — but `go generate` on `pkg/callhandler/` should not regenerate anything related to the changed helpers since they are neither interface methods nor `//go:generate` targets.

---

## Task 6: Check for conflicts against latest `main`

Monorepo `CLAUDE.md` §"CRITICAL: Pull Main and Check Conflicts Before PR/Merge" is mandatory.

### Step 6.1: Fetch and diff

Run (from the worktree):
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-apply-provider-tech-config-on-outbound
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "NO CONFLICTS"
git log --oneline HEAD..origin/main | head -20
```

Expected: `NO CONFLICTS`. If any conflict lines print, STOP: rebase or merge `main` into the branch, resolve conflicts manually, then re-run Task 5 (verification) before continuing.

### Step 6.2: No commit

Nothing to commit — either the branch is clean against main (proceed) or a conflict is surfaced for manual resolution.

---

## Task 7: Push and open PR

### Step 7.1: Push the branch

Run:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-apply-provider-tech-config-on-outbound
git push -u origin NOJIRA-apply-provider-tech-config-on-outbound
```

### Step 7.2: Create the PR

Use the project's PR title/body format from root `CLAUDE.md` — narrative summary first, then dashed project-prefixed bullets. **Do NOT** include AI attribution or a "Test plan" section.

```bash
gh pr create --title "NOJIRA-apply-provider-tech-config-on-outbound" --body "$(cat <<'EOF'
Make Provider.TechPrefix, Provider.TechPostfix, and Provider.TechHeaders
actually affect outbound tel-type calls. These fields have been stored,
editable via the Providers API, and documented in the RST overview/struct
pages for a long time, but bin-call-manager's getDialURITel only used the
hostname — so operators who configured them got no effect and the docs
contradicted the code.

- bin-call-manager: Wrap the SIP URI user part with TechPrefix and
  TechPostfix in getDialURITel
- bin-call-manager: Attach TechHeaders as PJSIP_HEADER(add,...) channel
  variables on the outgoing INVITE
- bin-call-manager: Add mergeTechHeaders helper with a reserved-key
  denylist (P-Asserted-Identity, Privacy, SDP-Transport, VB-CALL-ID,
  VB-CONFBRIDGE-ID, VB-DIRECTION, CALLERID(*)) and CRLF / invalid-char
  sanitization on keys and values
- bin-call-manager: Seed tech headers into channelVariables before the
  system-set transport and caller-ID functions run, so system-set keys
  win on collision for unconditionally-written headers; the denylist
  covers conditionally-written headers (PAI/Privacy on anonymous calls)
- bin-call-manager: Change getDialURITel, getDialURI, getDialURISIP,
  getDialURISIPDirect to return (string, map[string]string, error); the
  header map is nil for non-provider paths
- bin-call-manager: Add Info log summarizing applied/skipped tech config
  and Warn logs per skipped header entry with a reason
- bin-call-manager: Add Test_mergeTechHeaders covering nil/empty src,
  normal pass-through, empty/invalid-char/reserved key skips, CRLF value
  skip, CALLERID reserved raw-key skip, mixed valid/invalid, and merge
  overwrite semantics
- bin-call-manager: Extend Test_getDialURI_Tel with prefix-only, postfix-
  only, both, headers-only, and all-three cases
- bin-call-manager: Add end-to-end tech-config subtest to Test_createChannel
- bin-call-manager: Update Test_getDialURI_SIP, Test_getDialURI_error,
  Test_getDialURISIP, Test_getDialURISIPDirect for the 3-return signature
EOF
)"
```

### Step 7.3: Done

Return the PR URL to the user. Do NOT merge without explicit user authorization, and when authorized, use `gh pr merge <num> --squash --delete-branch`.

---

## Summary of commits expected on this branch

1. `43fefd9af` — design doc (already committed before this plan was written).
2. Task 1 — add `mergeTechHeaders` helper and tests.
3. Task 3 — apply tech config in `getDialURITel` / `createChannelOutgoing` + update existing tests (bundles with Task 2 since the tree must stay buildable).
4. Task 5 — (only if `go mod tidy` changed `go.mod` / `go.sum`, which is not expected).

Task 2, 4, 6, 7 have no independent commits. This yields a clean 2–3 commit branch (plus the design-doc commit), all squashed into one `main` commit on merge per project policy.
