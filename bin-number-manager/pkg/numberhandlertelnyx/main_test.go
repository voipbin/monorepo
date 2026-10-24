package numberhandlertelnyx

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/providernumber"
	"monorepo/bin-number-manager/pkg/requestexternal"
	"monorepo/bin-number-manager/pkg/requestexternal/models/telnyx"
)

func newTestHandler(mockExternal *requestexternal.MockRequestExternal) *numberHandlerTelnyx {
	return &numberHandlerTelnyx{
		connectionID:    "conn-1",
		profileID:       "prof-1",
		token:           "test-token",
		requestExternal: mockExternal,
	}
}

func Test_NewNumberHandler(t *testing.T) {
	h := NewNumberHandler(nil, nil, "conn-1", "prof-1", "test-token")

	res, ok := h.(*numberHandlerTelnyx)
	if !ok {
		t.Fatalf("Wrong type. got: %T", h)
	}
	if res.connectionID != "conn-1" || res.profileID != "prof-1" || res.token != "test-token" {
		t.Errorf("Wrong injected values. got: %q, %q, %q", res.connectionID, res.profileID, res.token)
	}
	if res.requestExternal == nil {
		t.Errorf("Wrong match. expect: request external, got: nil")
	}
}

func Test_NumberPurchase(t *testing.T) {
	orderErr := errors.New("order rejected")
	lookupErr := errors.New("lookup failed")

	tests := []struct {
		name string

		num string

		responseOrder     *telnyx.OrderNumber
		responseOrderErr  error
		responseLookup    *telnyx.PhoneNumber
		responseLookupErr error
		expectLookupCall  bool

		expectRes *providernumber.ProviderNumber
		expectErr error
		wrapped   bool
	}{
		{
			name: "normal",
			num:  "+15550100001",

			responseOrder: &telnyx.OrderNumber{ID: "order-1"},
			responseLookup: &telnyx.PhoneNumber{
				ID:                   "pn-1",
				Status:               telnyx.PhoneNumberStatusActive,
				T38FaxGatewayEnabled: true,
				EmergencyEnabled:     true,
			},
			expectLookupCall: true,

			expectRes: &providernumber.ProviderNumber{
				ID:               "pn-1",
				Status:           number.StatusActive,
				T38Enabled:       true,
				EmergencyEnabled: true,
			},
		},
		{
			name: "order fails and the cause is wrapped",
			num:  "+15550100001",

			responseOrder:    &telnyx.OrderNumber{},
			responseOrderErr: orderErr,

			expectErr: orderErr,
			wrapped:   true,
		},
		{
			name: "lookup fails and the cause is returned as it is",
			num:  "+15550100001",

			responseOrder:     &telnyx.OrderNumber{ID: "order-1"},
			responseLookupErr: lookupErr,
			expectLookupCall:  true,

			expectErr: lookupErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockExternal := requestexternal.NewMockRequestExternal(mc)
			h := newTestHandler(mockExternal)

			mockExternal.EXPECT().TelnyxNumberOrdersPost("test-token", []string{tt.num}, "conn-1", "prof-1").Return(tt.responseOrder, tt.responseOrderErr)
			if tt.expectLookupCall {
				mockExternal.EXPECT().TelnyxPhoneNumbersGetByNumber("test-token", tt.num).Return(tt.responseLookup, tt.responseLookupErr)
			}

			res, err := h.NumberPurchase(tt.num)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				if tt.wrapped == (err == tt.expectErr) {
					t.Errorf("Wrong wrapping. expect wrapped: %v, got: %v", tt.wrapped, err)
				}
				if res != nil {
					t.Errorf("Wrong result. expect: nil, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_NumberRelease(t *testing.T) {
	deleteErr := errors.New("delete failed")

	tests := []struct {
		name string

		responseErr error
	}{
		{"normal", nil},
		{"provider error is returned", deleteErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockExternal := requestexternal.NewMockRequestExternal(mc)
			h := newTestHandler(mockExternal)

			n := &number.Number{
				Identity:            commonidentity.Identity{ID: uuid.FromStringOrNil("2b1f4a3e-0c1d-11ef-8f1a-6b0e1e5d9a11")},
				ProviderReferenceID: "pn-1",
			}

			mockExternal.EXPECT().TelnyxPhoneNumbersIDDelete("test-token", "pn-1").Return(&telnyx.PhoneNumber{ID: "pn-1"}, tt.responseErr)

			err := h.NumberRelease(context.Background(), n)
			if err != tt.responseErr {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseErr, err)
			}
		})
	}
}

func Test_NumberUpdateTags(t *testing.T) {
	updateErr := errors.New("update failed")

	tests := []struct {
		name string

		tags []string

		responseErr error
	}{
		{"normal", []string{"CustomerID_1", "NumberID_2"}, nil},
		{"provider error is returned", []string{"CustomerID_1"}, updateErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockExternal := requestexternal.NewMockRequestExternal(mc)
			h := newTestHandler(mockExternal)

			n := &number.Number{
				Identity:            commonidentity.Identity{ID: uuid.FromStringOrNil("2b1f4a3e-0c1d-11ef-8f1a-6b0e1e5d9a11")},
				ProviderReferenceID: "pn-1",
			}

			mockExternal.EXPECT().TelnyxPhoneNumbersIDUpdate("test-token", "pn-1", map[string]interface{}{"tags": tt.tags}).Return(&telnyx.PhoneNumber{ID: "pn-1"}, tt.responseErr)

			err := h.NumberUpdateTags(context.Background(), n, tt.tags)
			if err != tt.responseErr {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseErr, err)
			}
		})
	}
}

func Test_GetAvailableNumbers(t *testing.T) {
	providerErr := errors.New("provider failed")

	tests := []struct {
		name string

		countryCode string
		limit       uint

		response    []*telnyx.AvailableNumber
		responseErr error

		expectRes []*availablenumber.AvailableNumber
		expectErr error
	}{
		{
			name:        "normal",
			countryCode: "US",
			limit:       2,
			response: []*telnyx.AvailableNumber{
				{
					PhoneNumber: "+15550100001",
					RegionInformation: []telnyx.AvailableRegionInformation{
						{RegionName: "US", RegionType: "country_code"},
						{RegionName: "IL", RegionType: "state"},
					},
					Features: []telnyx.AvailableFeature{{Name: "voice"}, {Name: "sms"}},
				},
				{PhoneNumber: "+15550100002"},
			},
			expectRes: []*availablenumber.AvailableNumber{
				{
					Number:       "+15550100001",
					ProviderName: number.ProviderNameTelnyx,
					Country:      "US",
					Region:       "IL",
					Features:     []availablenumber.Feature{availablenumber.FeatureVoice, availablenumber.FeatureSMS},
				},
				{Number: "+15550100002", ProviderName: number.ProviderNameTelnyx},
			},
		},
		{
			name:        "no numbers returns an empty non-nil list",
			countryCode: "US",
			limit:       5,
			response:    []*telnyx.AvailableNumber{},
			expectRes:   []*availablenumber.AvailableNumber{},
		},
		{
			name:        "provider error",
			countryCode: "US",
			limit:       5,
			responseErr: providerErr,
			expectErr:   providerErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockExternal := requestexternal.NewMockRequestExternal(mc)
			h := newTestHandler(mockExternal)

			mockExternal.EXPECT().TelnyxAvailableNumberGets("test-token", tt.countryCode, "", "", tt.limit).Return(tt.response, tt.responseErr)

			res, err := h.GetAvailableNumbers(tt.countryCode, tt.limit)
			if tt.expectErr != nil {
				if err != tt.expectErr {
					t.Fatalf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("Wrong result. expect: nil, got: %v", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
