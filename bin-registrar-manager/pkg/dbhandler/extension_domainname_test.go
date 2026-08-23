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

	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

// Test_extensionNormalizeDomainName_servingRule covers the VOIP-1385
// post-cutover SERVING RULE at the dbhandler read path: serve Realm when set;
// when Realm is empty (deleted-customer orphan rows only) serve the stored
// domain_name column as-is. There is no computed fallback anymore.
func Test_extensionNormalizeDomainName_servingRule(t *testing.T) {

	customerID := uuid.FromStringOrNil("d1e2f3a4-7f81-11ee-8f5a-1b2c3d4e5f60")

	tests := []struct {
		name string

		ext *extension.Extension

		expectDomainName string
	}{
		{
			name: "realm set: serve realm over the stored domain_name",

			ext: &extension.Extension{
				Identity:   commonidentity.Identity{CustomerID: customerID},
				DomainName: customerID.String(), // stale stored value
				Realm:      "ab12.reg.voipbin.net",
			},

			expectDomainName: "ab12.reg.voipbin.net",
		},
		{
			name: "realm set and equal to domain_name (post-cutover row)",

			ext: &extension.Extension{
				Identity:   commonidentity.Identity{CustomerID: customerID},
				DomainName: "cd34.reg.voipbin.net",
				Realm:      "cd34.reg.voipbin.net",
			},

			expectDomainName: "cd34.reg.voipbin.net",
		},
		{
			name: "realm empty (orphan row): stored domain_name served as-is",

			ext: &extension.Extension{
				Identity:   commonidentity.Identity{CustomerID: customerID},
				DomainName: customerID.String(),
				Realm:      "",
			},

			expectDomainName: customerID.String(),
		},
		{
			name: "realm empty and domain_name empty: stays empty",

			ext: &extension.Extension{
				Identity: commonidentity.Identity{CustomerID: customerID},
			},

			expectDomainName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extensionNormalizeDomainName(tt.ext)

			if tt.ext.DomainName != tt.expectDomainName {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectDomainName, tt.ext.DomainName)
			}
		})
	}
}

// Test_ExtensionGet_domainNameServing exercises the serving rule through the
// real sqlite read path: a row with a realm must come back with the realm as
// domain_name while the stored column stays untouched.
func Test_ExtensionGet_domainNameServing(t *testing.T) {
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
	realm := "ab12.reg.voipbin.net"
	row := &extension.Extension{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("f3a4b5c6-7f81-11ee-8f5a-1b2c3d4e5f60"),
			CustomerID: customerID,
		},

		EndpointID: "1001@" + realm,
		AORID:      "1001@" + realm,
		AuthID:     "1001@" + realm,

		Extension:  "1001",
		DomainName: customerID.String(), // stale stored column value
		Realm:      realm,
		Username:   "1001",
		Password:   "secret",
	}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
	if err := h.ExtensionCreate(ctx, row); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	mockCache.EXPECT().ExtensionGet(ctx, row.ID).Return(nil, fmt.Errorf("cache miss"))
	mockCache.EXPECT().ExtensionSet(ctx, gomock.Any())
	res, err := h.ExtensionGet(ctx, row.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if res.DomainName != realm {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", realm, res.DomainName)
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
