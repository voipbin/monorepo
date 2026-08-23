package dbhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/common"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

// Test_extensionNormalizeDomainName_servingRule covers the VOIP-1385 SERVING
// RULE at the dbhandler read path: serve Realm when set; when Realm is empty
// (legacy pre-Feb-2024 rows) serve the computed legacy realm; never the raw
// stored bare-uuid domain_name.
func Test_extensionNormalizeDomainName_servingRule(t *testing.T) {

	customerID := uuid.FromStringOrNil("d1e2f3a4-7f81-11ee-8f5a-1b2c3d4e5f60")

	tests := []struct {
		name string

		setBaseDomain bool
		ext           *extension.Extension

		expectDomainName string
	}{
		{
			name: "realm set: serve realm",

			setBaseDomain: true,
			ext: &extension.Extension{
				Identity:   commonidentity.Identity{CustomerID: customerID},
				DomainName: customerID.String(), // raw legacy bare uuid
				Realm:      "ab12.reg.voipbin.net",
			},

			expectDomainName: "ab12.reg.voipbin.net",
		},
		{
			name: "realm empty (legacy row): serve computed legacy realm, never the raw bare uuid",

			setBaseDomain: true,
			ext: &extension.Extension{
				Identity:   commonidentity.Identity{CustomerID: customerID},
				DomainName: customerID.String(),
				Realm:      "",
			},

			expectDomainName: customerID.String() + ".registrar.voipbin.net",
		},
		{
			name: "realm empty and base uninitialized: stored value left as-is (unit-test safety)",

			setBaseDomain: false,
			ext: &extension.Extension{
				Identity:   commonidentity.Identity{CustomerID: customerID},
				DomainName: customerID.String(),
				Realm:      "",
			},

			expectDomainName: customerID.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			common.ResetBaseDomainNamesForTest()
			if tt.setBaseDomain {
				if errSet := common.SetBaseDomainNames("registrar.voipbin.net", "trunk.voipbin.net"); errSet != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
				}
				defer common.ResetBaseDomainNamesForTest()
			}

			extensionNormalizeDomainName(tt.ext)

			if tt.ext.DomainName != tt.expectDomainName {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectDomainName, tt.ext.DomainName)
			}
		})
	}
}

// Test_ExtensionGet_legacyDomainNameServing exercises the serving rule through
// the real sqlite read path: a legacy row (NULL realm, bare uuid domain_name)
// must come back with the computed legacy realm as domain_name.
func Test_ExtensionGet_legacyDomainNameServing(t *testing.T) {
	common.ResetBaseDomainNamesForTest()
	if errSet := common.SetBaseDomainNames("registrar.voipbin.net", "trunk.voipbin.net"); errSet != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
	}
	defer common.ResetBaseDomainNamesForTest()

	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx := context.Background()

	customerID := uuid.FromStringOrNil("e2f3a4b5-7f81-11ee-8f5a-1b2c3d4e5f60")
	legacy := &extension.Extension{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("f3a4b5c6-7f81-11ee-8f5a-1b2c3d4e5f60"),
			CustomerID: customerID,
		},

		EndpointID: "1001@" + customerID.String() + ".registrar.voipbin.net",
		AORID:      "1001@" + customerID.String() + ".registrar.voipbin.net",
		AuthID:     "1001@" + customerID.String() + ".registrar.voipbin.net",

		Extension:  "1001",
		DomainName: customerID.String(), // legacy: bare uuid in the stored column
		Realm:      "",                  // legacy: NULL realm
		Username:   "1001",
		Password:   "secret",
	}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
	if err := h.ExtensionCreate(ctx, legacy); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	mockCache.EXPECT().ExtensionGet(ctx, legacy.ID).Return(nil, fmt.Errorf("cache miss"))
	mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
	res, err := h.ExtensionGet(ctx, legacy.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectDomainName := customerID.String() + ".registrar.voipbin.net"
	if res.DomainName != expectDomainName {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expectDomainName, res.DomainName)
	}

	// the stored column stays untouched (only the in-memory model is normalized)
	var stored string
	if errScan := dbTest.QueryRow("select domain_name from registrar_extensions where extension = '1001'").Scan(&stored); errScan != nil {
		t.Fatalf("could not read the stored domain_name. err: %v", errScan)
	}
	if stored != customerID.String() {
		t.Errorf("Wrong match. expect stored: %s, got: %s", customerID.String(), stored)
	}
}
