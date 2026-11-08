package messagehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
)

const (
	testSourceOwned   = "+15550100001"
	testSourceDefault = "+15550100002"
)

var (
	testDefaultNumberID = uuid.FromStringOrNil("6c0b8a1e-3d52-4f6e-9b1a-0d7e2f4a5c11")
	errTestRPC          = errors.New("rpc failed")
)

// ownedFilter is the filter that proves a number belongs to the customer as a normal active number.
func ownedFilter(key nmnumber.Field, value any) map[nmnumber.Field]any {
	return map[nmnumber.Field]any{
		nmnumber.FieldCustomerID: testCustomerID,
		key:                      value,
		nmnumber.FieldType:       nmnumber.TypeNormal,
		nmnumber.FieldStatus:     nmnumber.StatusActive,
		nmnumber.FieldDeleted:    false,
	}
}

func (f *sendFixture) expectNumberByNumber(number string, res []nmnumber.Number, err error) *gomock.Call {
	return f.req.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), ownedFilter(nmnumber.FieldNumber, number)).Return(res, err)
}

func (f *sendFixture) expectNumberByID(id uuid.UUID, res []nmnumber.Number, err error) *gomock.Call {
	return f.req.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), ownedFilter(nmnumber.FieldID, id)).Return(res, err)
}

func (f *sendFixture) expectOutboundConfig(res []cmoutboundconfig.OutboundConfig, err error) *gomock.Call {
	return f.req.EXPECT().CallV1OutboundConfigList(gomock.Any(), testCustomerID, uint64(1), "").Return(res, err)
}

// expectSourceOwned makes any source of the customer pass the ownership check, for the tests that are not about it.
func (f *sendFixture) expectSourceOwned() {
	f.req.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), gomock.Any()).Return([]nmnumber.Number{{}}, nil)
}

func Test_prepareSource(t *testing.T) {
	tests := []struct {
		name string

		source *commonaddress.Address

		expectMissing bool
		expectRes     *commonaddress.Address
		expectReason  string
	}{
		{"nil is missing", nil, true, nil, ""},
		{"empty target is missing", &commonaddress.Address{Type: commonaddress.TypeTel}, true, nil, ""},
		{"whitespace target with empty type is missing", &commonaddress.Address{Target: "   "}, true, nil, ""},
		{"empty target of another type is missing", &commonaddress.Address{Type: commonaddress.TypeSIP}, true, nil, ""},
		{
			"e164 tel passes",
			&commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceOwned, TargetName: "main"},
			false,
			&commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceOwned, TargetName: "main"},
			"",
		},
		{
			"empty type is read as tel",
			&commonaddress.Address{Target: testSourceOwned},
			false,
			&commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceOwned},
			"",
		},
		{
			"separators are normalized",
			&commonaddress.Address{Type: commonaddress.TypeTel, Target: " +1 (555) 010-0001 "},
			false,
			&commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceOwned},
			"",
		},
		{"alphanumeric sender id", &commonaddress.Address{Type: commonaddress.TypeTel, Target: "Acme"}, false, nil, "SOURCE_INVALID"},
		{"no plus", &commonaddress.Address{Type: commonaddress.TypeTel, Target: "15550100001"}, false, nil, "SOURCE_INVALID"},
		{"letters after the plus", &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+Acme1234567"}, false, nil, "SOURCE_INVALID"},
		{"letters between digits", &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+82ab1012345678"}, false, nil, "SOURCE_INVALID"},
		{"full width digits", &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+１５５５０１００００１"}, false, nil, "SOURCE_INVALID"},
		{"too short", &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1234"}, false, nil, "SOURCE_INVALID"},
		{"too long", &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1234567890123456"}, false, nil, "SOURCE_INVALID"},
		{"sip type", &commonaddress.Address{Type: commonaddress.TypeSIP, Target: testSourceOwned}, false, nil, "SOURCE_INVALID"},
		{"email type", &commonaddress.Address{Type: commonaddress.TypeEmail, Target: "a@example.com"}, false, nil, "SOURCE_INVALID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var before commonaddress.Address
			if tt.source != nil {
				before = *tt.source
			}

			res, err := prepareSource(tt.source)

			if tt.expectReason != "" {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				assertVoipbinError(t, err, cerrors.StatusInvalidArgument, tt.expectReason)
			} else if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if tt.expectMissing && res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if tt.expectRes != nil && !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			// the caller's address is never changed.
			if tt.source != nil && *tt.source != before {
				t.Errorf("The caller's address was changed.\nbefore: %v\nafter: %v", before, *tt.source)
			}
			if res != nil && res == tt.source {
				t.Errorf("The result must be a copy, not the caller's pointer.")
			}
		})
	}
}

func Test_resolveSource_supplied(t *testing.T) {
	owned := []nmnumber.Number{{Number: testSourceOwned}}

	tests := []struct {
		name string

		expect func(f *sendFixture)

		expectReason string
		expectStatus cerrors.Status
	}{
		{
			"owned number passes",
			func(f *sendFixture) { f.expectNumberByNumber(testSourceOwned, owned, nil) },
			"", "",
		},
		{
			"not owned fails and never falls back to the default",
			func(f *sendFixture) { f.expectNumberByNumber(testSourceOwned, []nmnumber.Number{}, nil) },
			"SOURCE_NOT_OWNED", cerrors.StatusPermissionDenied,
		},
		{
			"lookup error fails closed",
			func(f *sendFixture) { f.expectNumberByNumber(testSourceOwned, nil, errTestRPC) },
			"SOURCE_LOOKUP_FAILED", cerrors.StatusInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSendFixture(t)
			tt.expect(f)

			src := &commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceOwned, TargetName: "main"}
			res, err := f.handler.resolveSource(context.Background(), testCustomerID, src)

			if tt.expectReason != "" {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				assertVoipbinError(t, err, tt.expectStatus, tt.expectReason)
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, src) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", src, res)
			}
		})
	}
}

func Test_resolveSource_cancelledContext(t *testing.T) {
	f := newSendFixture(t)
	f.req.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), gomock.Any()).Return(nil, context.Canceled)

	src := &commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceOwned}
	res, err := f.handler.resolveSource(context.Background(), testCustomerID, src)
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	assertVoipbinError(t, err, cerrors.StatusInternal, "SOURCE_LOOKUP_FAILED")
}

func Test_resolveSource_default(t *testing.T) {
	cfgWith := func(id uuid.UUID) []cmoutboundconfig.OutboundConfig {
		return []cmoutboundconfig.OutboundConfig{{DefaultOutgoingSourceNumberID: id}}
	}

	tests := []struct {
		name string

		expect func(f *sendFixture)

		expectReason string
		expectStatus cerrors.Status
	}{
		{
			"default number is used",
			func(f *sendFixture) {
				f.expectOutboundConfig(cfgWith(testDefaultNumberID), nil)
				f.expectNumberByID(testDefaultNumberID, []nmnumber.Number{{Number: testSourceDefault}}, nil)
			},
			"", "",
		},
		{
			"no outbound config row",
			func(f *sendFixture) { f.expectOutboundConfig([]cmoutboundconfig.OutboundConfig{}, nil) },
			"SOURCE_REQUIRED", cerrors.StatusInvalidArgument,
		},
		{
			"default disabled with the nil id",
			func(f *sendFixture) { f.expectOutboundConfig(cfgWith(uuid.Nil), nil) },
			"SOURCE_REQUIRED", cerrors.StatusInvalidArgument,
		},
		{
			"stale default (released, inactive, virtual or foreign)",
			func(f *sendFixture) {
				f.expectOutboundConfig(cfgWith(testDefaultNumberID), nil)
				f.expectNumberByID(testDefaultNumberID, []nmnumber.Number{}, nil)
			},
			"DEFAULT_SOURCE_INVALID", cerrors.StatusFailedPrecondition,
		},
		{
			"outbound config lookup error",
			func(f *sendFixture) { f.expectOutboundConfig(nil, errTestRPC) },
			"SOURCE_LOOKUP_FAILED", cerrors.StatusInternal,
		},
		{
			"default number lookup error",
			func(f *sendFixture) {
				f.expectOutboundConfig(cfgWith(testDefaultNumberID), nil)
				f.expectNumberByID(testDefaultNumberID, nil, errTestRPC)
			},
			"SOURCE_LOOKUP_FAILED", cerrors.StatusInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSendFixture(t)
			tt.expect(f)

			res, err := f.handler.resolveSource(context.Background(), testCustomerID, nil)

			if tt.expectReason != "" {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				assertVoipbinError(t, err, tt.expectStatus, tt.expectReason)
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			expect := &commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceDefault}
			if !reflect.DeepEqual(res, expect) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
			}
		})
	}
}

func Test_normalizeDestinations(t *testing.T) {
	dests := []commonaddress.Address{
		{Type: commonaddress.TypeTel, Target: " +1 555 010 0003 "},
		{Type: commonaddress.TypeTel, Target: "+1-555-010-0004"},
		{Type: commonaddress.TypeTel, Target: "Acme"},
	}

	normalizeDestinations(dests)

	expect := []string{"+15550100003", "+15550100004", "Acme"}
	for i, d := range dests {
		if d.Target != expect[i] {
			t.Errorf("Wrong match. index: %d, expect: %s, got: %s", i, expect[i], d.Target)
		}
	}
}

// Test_Send_sourceRejected checks that a rejected sender ends Send before the balance check, the create and the
// dispatch (the mocks fail the test on any other call), and that the identity gate still runs first.
func Test_Send_sourceRejected(t *testing.T) {
	dests := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550100009"}}

	tests := []struct {
		name string

		source *commonaddress.Address
		expect func(f *sendFixture)

		expectStatus cerrors.Status
		expectReason string
	}{
		{"invalid format needs no rpc", &commonaddress.Address{Type: commonaddress.TypeTel, Target: "Acme"}, func(f *sendFixture) {}, cerrors.StatusInvalidArgument, "SOURCE_INVALID"},
		{"not owned", &commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceOwned}, func(f *sendFixture) {
			f.expectNumberByNumber(testSourceOwned, []nmnumber.Number{}, nil)
		}, cerrors.StatusPermissionDenied, "SOURCE_NOT_OWNED"},
		{"missing and no default", nil, func(f *sendFixture) {
			f.expectOutboundConfig([]cmoutboundconfig.OutboundConfig{}, nil)
		}, cerrors.StatusInvalidArgument, "SOURCE_REQUIRED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSendFixture(t)
			f.expectVerified()
			tt.expect(f)

			res, err := f.handler.Send(context.Background(), testMessageID, testCustomerID, tt.source, dests, "hi")
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			assertVoipbinError(t, err, tt.expectStatus, tt.expectReason)
		})
	}
}

func Test_Send_unverifiedCustomerNeverLooksUpTheSource(t *testing.T) {
	f := newSendFixture(t)
	f.req.EXPECT().CustomerV1CustomerGet(gomock.Any(), testCustomerID).Return(&cucustomer.Customer{ID: testCustomerID, IdentityVerificationStatus: cucustomer.IdentityVerificationStatusNone}, nil)

	src := &commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceOwned}
	dests := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550100009"}}
	if res, err := f.handler.Send(context.Background(), testMessageID, testCustomerID, src, dests, "hi"); res != nil || err == nil {
		t.Errorf("Wrong match. expect: rejected, got: %v, %v", res, err)
	}
}

// Test_Send_createdWithResolvedSource checks the source that is stored: the default number when the request has none,
// the canonical form of a supplied number, and that the caller's address is left alone.
func Test_Send_createdWithResolvedSource(t *testing.T) {
	dests := func() []commonaddress.Address {
		return []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550100009"}}
	}

	tests := []struct {
		name string

		source *commonaddress.Address
		expect func(f *sendFixture)

		expectSource commonaddress.Address
	}{
		{
			"default number for a missing source",
			&commonaddress.Address{},
			func(f *sendFixture) {
				f.expectOutboundConfig([]cmoutboundconfig.OutboundConfig{{DefaultOutgoingSourceNumberID: testDefaultNumberID}}, nil)
				f.expectNumberByID(testDefaultNumberID, []nmnumber.Number{{Number: testSourceDefault}}, nil)
			},
			commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceDefault},
		},
		{
			"canonical form of a supplied number",
			&commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1 (555) 010-0001", TargetName: "main"},
			func(f *sendFixture) {
				f.expectNumberByNumber(testSourceOwned, []nmnumber.Number{{Number: testSourceOwned}}, nil)
			},
			commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceOwned, TargetName: "main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSendFixture(t)
			f.expectVerified()
			tt.expect(f)
			f.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(gomock.Any(), testCustomerID, gomock.Any(), "", 1).Return(true, nil)

			var created *message.Message
			f.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, m *message.Message) error {
				created = m
				return errTestRPC // ends Send before the dispatch goroutine
			})

			before := *tt.source
			_, err := f.handler.Send(context.Background(), testMessageID, testCustomerID, tt.source, dests(), "hi")
			if err == nil {
				t.Fatalf("Wrong match. expect: the create error, got: nil")
			}
			if created == nil || created.Source == nil {
				t.Fatalf("Wrong match. The message was not created with a source: %v", created)
			}
			if *created.Source != tt.expectSource {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectSource, *created.Source)
			}
			if *tt.source != before {
				t.Errorf("The caller's address was changed.\nbefore: %v\nafter: %v", before, *tt.source)
			}
		})
	}
}

func (f *sendMMSFixture) expectSourceOwned() {
	f.req.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), gomock.Any()).Return([]nmnumber.Number{{}}, nil)
}

// Test_SendMMS_sourceRejected checks that a rejected sender ends SendMMS before the media check, the balance check and
// the create (the mocks fail the test on any other call).
func Test_SendMMS_sourceRejected(t *testing.T) {
	dests := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550100009"}}

	tests := []struct {
		name string

		source *commonaddress.Address
		expect func(f *sendMMSFixture)

		expectStatus cerrors.Status
		expectReason string
	}{
		{"invalid format needs no rpc", &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+Acme1234567"}, func(f *sendMMSFixture) {}, cerrors.StatusInvalidArgument, "SOURCE_INVALID"},
		{"not owned", &commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceOwned}, func(f *sendMMSFixture) {
			f.req.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), ownedFilter(nmnumber.FieldNumber, testSourceOwned)).Return([]nmnumber.Number{}, nil)
		}, cerrors.StatusPermissionDenied, "SOURCE_NOT_OWNED"},
		{"lookup failed", &commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceOwned}, func(f *sendMMSFixture) {
			f.req.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), gomock.Any()).Return(nil, errTestRPC)
		}, cerrors.StatusInternal, "SOURCE_LOOKUP_FAILED"},
		{"stale default", nil, func(f *sendMMSFixture) {
			f.req.EXPECT().CallV1OutboundConfigList(gomock.Any(), testCustomerID, uint64(1), "").Return([]cmoutboundconfig.OutboundConfig{{DefaultOutgoingSourceNumberID: testDefaultNumberID}}, nil)
			f.req.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), ownedFilter(nmnumber.FieldID, testDefaultNumberID)).Return([]nmnumber.Number{}, nil)
		}, cerrors.StatusFailedPrecondition, "DEFAULT_SOURCE_INVALID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSendMMSFixture(t)
			f.expectVerifiedCustomer()
			tt.expect(f)

			res, err := f.handler.SendMMS(context.Background(), testMessageID, testCustomerID, tt.source, append([]commonaddress.Address(nil), dests...), "hi", []uuid.UUID{testMediaID1})
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			assertVoipbinError(t, err, tt.expectStatus, tt.expectReason)
		})
	}
}

// Test_SendMMS_createdWithDefaultSource checks that a missing source is replaced by the default number in the stored
// message, and that the lookups run before the media check.
func Test_SendMMS_createdWithDefaultSource(t *testing.T) {
	f := newSendMMSFixture(t)
	f.expectVerifiedCustomer()

	gomock.InOrder(
		f.req.EXPECT().CallV1OutboundConfigList(gomock.Any(), testCustomerID, uint64(1), "").Return([]cmoutboundconfig.OutboundConfig{{DefaultOutgoingSourceNumberID: testDefaultNumberID}}, nil),
		f.req.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), ownedFilter(nmnumber.FieldID, testDefaultNumberID)).Return([]nmnumber.Number{{Number: testSourceDefault}}, nil),
		f.req.EXPECT().StorageV1FileGet(gomock.Any(), testMediaID1).Return(testMediaFile(testMediaID1, testCustomerID, "a.png", 10), nil),
	)
	f.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(gomock.Any(), testCustomerID, gomock.Any(), "", 1).Return(true, nil)
	f.req.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testMediaID1).Return("https://example.invalid/a.png", nil)

	var created *message.Message
	f.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, m *message.Message) error {
		created = m
		return errTestRPC // ends SendMMS before the dispatch goroutine
	})

	src := &commonaddress.Address{}
	_, err := f.handler.SendMMS(context.Background(), testMessageID, testCustomerID, src, []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550100009"}}, "hi", []uuid.UUID{testMediaID1})
	if err == nil {
		t.Fatalf("Wrong match. expect: the create error, got: nil")
	}
	expect := commonaddress.Address{Type: commonaddress.TypeTel, Target: testSourceDefault}
	if created == nil || created.Source == nil || *created.Source != expect {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, created)
	}
	if *src != (commonaddress.Address{}) {
		t.Errorf("The caller's address was changed: %v", *src)
	}
}
