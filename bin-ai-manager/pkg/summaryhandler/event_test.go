package summaryhandler

import (
	"context"
	"testing"

	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cfconference "monorepo/bin-conference-manager/models/conference"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// Test_EventCMConferenceUpdated covers EventCMConferenceUpdated's own boundary logic
// (VOIP-1422 activated this dispatch case, which had never run in production before).
// It does NOT re-test ContentProcess's internals -- contentProcessReferenceTypeConference
// already has deep, dedicated coverage in content_test.go
// (Test_contentProcessReferenceTypeConference). These cases exist to prove the newly-live
// wiring -- GetByReferenceID, the (deliberately absent) conference-Status guard, and the
// (deliberately present) summary-Status idempotency guard -- behaves correctly:
//   - case 1: no matching summary -- clean no-op.
//   - case 2: conference Status is Terminating (Delete()-API shape, NOT Destroy()'s
//     Terminated) -- pins the regression this ticket's own review caught: gating on the
//     conference's Status would silently drop summary finalization for every explicit API
//     deletion, since bin-conference-manager's two conference_deleted publish sites
//     disagree on that field at publish time.
//   - case 3: summary already summary.StatusDone -- pins a second, independent regression
//     this ticket's own review caught: bin-conference-manager can publish conference_deleted
//     TWICE for one conference (Delete()'s own publish, then Destroy()'s once the
//     asynchronously-kicked participants actually leave), and neither ContentProcess nor its
//     downstream is idempotent -- so EventCMConferenceUpdated must short-circuit on an
//     already-finalized summary instead of reprocessing it. Uses
//     summary.ReferenceTypeConference deliberately (not ReferenceTypeNone, unlike cases 1-2)
//     specifically because that reference type WOULD reach further, un-mocked DB/RPC calls
//     inside ContentProcess if the guard failed to short-circuit -- gomock's strict
//     controller would fail the test on the first unexpected call, so this case fails loudly
//     if the StatusDone guard regresses, not silently.
func Test_EventCMConferenceUpdated(t *testing.T) {

	tests := []struct {
		name string

		conference *cfconference.Conference

		// responseSummaries is what mockDB.SummaryList returns for the GetByReferenceID
		// lookup.
		responseSummaries []*summary.Summary
	}{
		{
			name: "no matching summary -- GetByReferenceID returns not-found, no panic",

			conference: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6e3f0c62-8b8c-11f0-9be7-af683eb63c9c"),
				},
				Status: cfconference.StatusTerminated,
			},

			responseSummaries: []*summary.Summary{},
		},
		{
			// Regression case for the Delete()-API publish shape: Status is
			// StatusTerminating (NOT StatusTerminated), matching exactly what
			// bin-conference-manager's Delete() actually publishes. Must still reach
			// ContentProcess -- if this handler regains a Status == StatusTerminated
			// gate, this case starts silently no-op'ing and this test would need to
			// catch that by asserting the mock expectation below was actually
			// satisfied (mc.Finish(), deferred, does exactly that).
			name: "summary found, conference Status is Terminating (Delete() API shape, not Destroy()'s Terminated) -- still reaches ContentProcess",

			conference: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6e60c2fc-8b8c-11f0-9a3f-4f846cd94d67"),
				},
				Status: cfconference.StatusTerminating,
			},

			responseSummaries: []*summary.Summary{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6e848c9e-8b8c-11f0-8b6c-53c1f37f5b70"),
					},
					ReferenceType: summary.ReferenceTypeNone,
					ReferenceID:   uuid.FromStringOrNil("6e60c2fc-8b8c-11f0-9a3f-4f846cd94d67"),
				},
			},
		},
		{
			name: "summary already StatusDone (second conference_deleted delivery for the same conference) -- must NOT reprocess",

			conference: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6ea5b0f8-8b8c-11f0-8f2d-7f2a6c9e1b3a"),
				},
				Status: cfconference.StatusTerminated,
			},

			responseSummaries: []*summary.Summary{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6ec4d2ba-8b8c-11f0-9f1e-3b6c8f2a5d70"),
					},
					// Deliberately ReferenceTypeConference, not ReferenceTypeNone: if the
					// StatusDone guard regresses, ContentProcess routes this to
					// contentProcessReferenceTypeConference, which needs several further
					// mocked DB/RPC calls this test does not set up -- gomock's strict
					// controller fails loudly on the first one, rather than this case
					// passing vacuously the way an unsupported-reference-type summary would.
					ReferenceType: summary.ReferenceTypeConference,
					ReferenceID:   uuid.FromStringOrNil("6ea5b0f8-8b8c-11f0-8f2d-7f2a6c9e1b3a"),
					Status:        summary.StatusDone,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &summaryHandler{
				db: mockDB,
			}
			ctx := context.Background()

			expectFilters := map[summary.Field]any{
				summary.FieldDeleted:     false,
				summary.FieldReferenceID: tt.conference.ID,
			}
			// GetByReferenceID must always be called -- there is no Status guard to
			// short-circuit it. Its absence here (via gomock's strict controller,
			// which fails the test on an unexpected call, or here failing on a missing
			// expected one via mc.Finish()) would itself be a signal something
			// regressed.
			mockDB.EXPECT().SummaryList(ctx, uint64(1), "", expectFilters).Return(tt.responseSummaries, nil)

			h.EventCMConferenceUpdated(ctx, tt.conference)
		})
	}
}
