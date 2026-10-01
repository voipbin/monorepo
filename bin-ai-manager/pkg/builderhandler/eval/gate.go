package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Status of one pass-rule group once a human has judged the transcripts.
type Status string

const (
	StatusPass    Status = "pass"
	StatusFail    Status = "fail"
	StatusPending Status = "pending"
)

// GroupStatus is the state of one group.
type GroupStatus struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Passed  int    `json:"passed"`
	Failed  int    `json:"failed"`
	Total   int    `json:"total"`
	MinPass int    `json:"min_pass"`
	Aborted int    `json:"aborted"` // runs that stopped on an engine error and must be re-run
}

// GateReport is the outcome of design 2.5's pass condition.
type GateReport struct {
	AutoOK       bool          `json:"auto_ok"`
	AutoFailures []string      `json:"auto_failures,omitempty"`
	Groups       []GroupStatus `json:"groups"`
	// UnknownVerdicts are verdict keys that match no run (usually a typo).
	UnknownVerdicts []string `json:"unknown_verdicts,omitempty"`
	Pending         int      `json:"pending_groups"`
	Failed          int      `json:"failed_groups"`
	// AllGood is true only when every automatic item passed AND every group has
	// enough human "good" verdicts. It says nothing about provenance: a run
	// against fake engines, or verdicts with no recorded judge, can be AllGood.
	// Use Verdict to get the statement that may be shown to a reader.
	AllGood bool `json:"all_good"`
}

const maxParseFailureRate = 0.05

// Evaluate applies design 2.5's pass condition.
//
// Automatic items (must all pass): the JSON parse-failure rate is at most 5%,
// scenario 9 returns a draft in its first response, and no init_prompt names a
// forbidden tool. Human items: for each group, at least MinPass of Total runs
// judged good. verdicts maps run id to good/bad; a missing entry means not yet
// judged. A run that aborted on an engine error is not judgeable and stays
// unjudged until it is re-run. Scenario 15 (synthetic, no simulator) is read
// by a person and is part of this gate: pass the synthetic results in results
// as well, and record a verdict for each of s15-1#1 to s15-5#1.
func Evaluate(results []RunResult, verdicts map[string]bool) GateReport {
	g := GateReport{AutoOK: true, UnknownVerdicts: UnknownVerdictIDs(results, verdicts)}

	calls, fails := 0, 0
	for _, r := range results {
		calls += r.BuilderCalls
		fails += r.ParseFailures
		if r.Group == "s9" && r.Aborted == "" && !r.FirstResponseHasDraft {
			g.AutoFailures = append(g.AutoFailures, fmt.Sprintf("%s: scenario 9 must return a draft in the first response", r.RunID))
		}
		if len(r.ForbiddenToolWarnings) > 0 {
			g.AutoFailures = append(g.AutoFailures, fmt.Sprintf("%s: init_prompt names a forbidden tool (%s)", r.RunID, strings.Join(r.ForbiddenToolWarnings, "; ")))
		}
	}
	if calls > 0 {
		if rate := float64(fails) / float64(calls); rate > maxParseFailureRate {
			g.AutoFailures = append(g.AutoFailures, fmt.Sprintf("JSON parse failure rate %.1f%% exceeds %.0f%% (%d of %d builder calls)", rate*100, maxParseFailureRate*100, fails, calls))
		}
	}
	g.AutoOK = len(g.AutoFailures) == 0

	byGroup := map[string][]RunResult{}
	for _, r := range results {
		byGroup[r.Group] = append(byGroup[r.Group], r)
	}

	allPass := true
	for _, grp := range Groups() {
		st := GroupStatus{Name: grp.Name, Total: grp.Total, MinPass: grp.MinPass}
		for _, r := range byGroup[grp.Name] {
			if r.ParseFailures > 0 {
				// The builder's answer was unusable. No verdict can rescue it,
				// and a re-run is a new run, not a replacement.
				st.Failed++
				continue
			}
			if r.Aborted != "" {
				st.Aborted++
				continue
			}
			v, judged := verdicts[r.RunID]
			if !judged {
				continue
			}
			if v {
				st.Passed++
			} else {
				st.Failed++
			}
		}
		unjudged := grp.Total - st.Passed - st.Failed
		switch {
		case st.Passed >= grp.MinPass:
			st.Status = StatusPass
		case st.Passed+unjudged < grp.MinPass:
			st.Status = StatusFail
		default:
			st.Status = StatusPending
		}
		switch st.Status {
		case StatusPending:
			g.Pending++
			allPass = false
		case StatusFail:
			g.Failed++
			allPass = false
		}
		g.Groups = append(g.Groups, st)
	}
	g.AllGood = g.AutoOK && allPass && len(g.UnknownVerdicts) == 0
	return g
}

// Gate verdict words. Verdict is the only place that decides which one applies.
const (
	GatePass          = "PASS"
	GateFail          = "FAIL"
	GateNotDecided    = "NOT DECIDED"
	GateNotApplicable = "NOT APPLICABLE"
)

// Verdict is the one statement about the gate that may be shown or acted on.
// A run against fake engines is never PASS or FAIL, and verdicts with no
// recorded judge are never PASS.
func (g GateReport) Verdict(real bool, judge string) string {
	switch {
	case !real:
		return GateNotApplicable
	case g.AllGood && strings.TrimSpace(judge) == "":
		return GateNotDecided
	case g.AllGood:
		return GatePass
	case g.Failed > 0 || !g.AutoOK:
		return GateFail
	default:
		return GateNotDecided
	}
}

// Verdicts is the content of verdicts.json: who judged, and one good/bad
// verdict per run id. The judge name is recorded so the report can show it; the
// code cannot verify that the judge is not the author of the prompt, so
// independence stays a human responsibility that the report makes visible.
type Verdicts struct {
	Judge string          `json:"judge"`
	Runs  map[string]bool `json:"verdicts"`
}

// LoadVerdicts reads verdicts.json. A missing file means nothing has been
// judged yet. A corrupt file is an error: silently treating it as "no
// verdicts" would hide a typo, and silently treating it as passing would be
// worse.
func LoadVerdicts(path string) (Verdicts, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Verdicts{Runs: map[string]bool{}}, nil
		}
		return Verdicts{}, err
	}
	var v Verdicts
	if err := json.Unmarshal(b, &v); err != nil {
		return Verdicts{}, fmt.Errorf(`verdicts file %s must look like {"judge": "name", "verdicts": {"run id": true}}: %w`, path, err)
	}
	if v.Runs == nil {
		v.Runs = map[string]bool{}
	}
	return v, nil
}
