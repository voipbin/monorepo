package extensionhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
)

func TestCreateExtension(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	h := &extensionHandler{
		dbAst: mockDBAst,
		dbBin: mockDBBin,
	}

	type test struct {
		name     string
		ext      *models.Extension
		domain   *models.Domain
		aor      *models.AstAOR
		auth     *models.AstAuth
		endpoint *models.AstEndpoint
	}

	tests := []test{
		{
			"test normal",
			&models.Extension{
				UserID:    1,
				DomainID:  uuid.FromStringOrNil("ce060aae-6ec1-11eb-a550-cb46a3229b89"),
				Extension: "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4",
				Password:  "cf6917ba-6ec1-11eb-8810-e3829c2dfab8",
			},
			&models.Domain{
				ID:         uuid.FromStringOrNil("ce060aae-6ec1-11eb-a550-cb46a3229b89"),
				DomainName: "test.sip.voipbin.net",
			},
			&models.AstAOR{
				ID:          getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@test.sip.voipbin.net"),
				MaxContacts: getIntegerPointer(1),
			},
			&models.AstAuth{
				ID:       getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@test.sip.voipbin.net"),
				AuthType: getStringPointer("userpass"),
				Username: getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4"),
				Password: getStringPointer("cf6917ba-6ec1-11eb-8810-e3829c2dfab8"),
				Realm:    getStringPointer("test.sip.voipbin.net"),
			},
			&models.AstEndpoint{
				ID:   getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@test.sip.voipbin.net"),
				AORs: getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@test.sip.voipbin.net"),
				Auth: getStringPointer("ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4@test.sip.voipbin.net"),
			},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		mockDBBin.EXPECT().DomainGet(gomock.Any(), tt.ext.DomainID).Return(tt.domain, nil)
		mockDBAst.EXPECT().AstAORCreate(gomock.Any(), tt.aor).Return(nil)
		mockDBAst.EXPECT().AstAuthCreate(gomock.Any(), tt.auth).Return(nil)
		mockDBAst.EXPECT().AstEndpointCreate(gomock.Any(), tt.endpoint).Return(nil)
		mockDBBin.EXPECT().ExtensionCreate(gomock.Any(), gomock.Any()).Return(nil)
		mockDBBin.EXPECT().ExtensionGet(gomock.Any(), gomock.Any()).Return(tt.ext, nil)
		_, err := h.CreateExtension(ctx, tt.ext.UserID, tt.ext.DomainID, tt.ext.Extension, tt.ext.Password)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}
