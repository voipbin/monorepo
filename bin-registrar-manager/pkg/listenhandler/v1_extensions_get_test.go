package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
)

func Test_processV1ExtensionsIDGet(t *testing.T) {
	id := uuid.FromStringOrNil("3f4bc63e-6ebf-11eb-b7de-df47266bf559")
	resExtension := &extension.Extension{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("2e341ffa-7fed-11ec-9667-1357b91d745d"),
		},
		Name:      "test name",
		Extension: "1001",
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExtension := extensionhandler.NewMockExtensionHandler(mc)
	h := &listenHandler{extensionHandler: mockExtension}

	mockExtension.EXPECT().Get(gomock.Any(), id).Return(resExtension, nil)

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/extensions/3f4bc63e-6ebf-11eb-b7de-df47266bf559",
		Method: sock.RequestMethodGet,
	})
	if err != nil {
		t.Fatalf("processRequest() error = %v", err)
	}
	if res.StatusCode != 200 || res.DataType != "application/json" {
		t.Fatalf("response = %d %q, want 200 application/json", res.StatusCode, res.DataType)
	}

	var got extension.Extension
	if err := json.Unmarshal(res.Data, &got); err != nil {
		t.Fatalf("could not decode the response body: %v", err)
	}
	if !reflect.DeepEqual(&got, resExtension) {
		t.Errorf("response body = %+v, want %+v", got, *resExtension)
	}
}

func Test_processV1ExtensionsIDGet_handlerError(t *testing.T) {
	id := uuid.FromStringOrNil("3f4bc63e-6ebf-11eb-b7de-df47266bf559")
	errGet := errors.New("lookup failed")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExtension := extensionhandler.NewMockExtensionHandler(mc)
	h := &listenHandler{extensionHandler: mockExtension}

	mockExtension.EXPECT().Get(gomock.Any(), id).Return(nil, errGet)

	res, err := h.processV1ExtensionsIDGet(context.Background(), &sock.Request{
		URI:    "/v1/extensions/3f4bc63e-6ebf-11eb-b7de-df47266bf559",
		Method: sock.RequestMethodGet,
	})
	if !errors.Is(err, errGet) {
		t.Errorf("error = %v, want the handler error", err)
	}
	if res != nil {
		t.Errorf("response = %+v, want nil", res)
	}
}

func Test_processV1ExtensionsIDGet_invalidUUID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExtension := extensionhandler.NewMockExtensionHandler(mc)
	h := &listenHandler{extensionHandler: mockExtension}

	// an unparsable id becomes uuid.Nil; the handler must still be asked, not skipped
	mockExtension.EXPECT().Get(gomock.Any(), uuid.Nil).Return(nil, errors.New("not found"))

	if _, err := h.processV1ExtensionsIDGet(context.Background(), &sock.Request{
		URI:    "/v1/extensions/not-a-uuid",
		Method: sock.RequestMethodGet,
	}); err == nil {
		t.Errorf("expected the handler error to be returned")
	}
}

func Test_processV1ExtensionsIDGet_badURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &listenHandler{extensionHandler: extensionhandler.NewMockExtensionHandler(mc)}

	res, err := h.processV1ExtensionsIDGet(context.Background(), &sock.Request{
		URI:    "/v1/extensions/%zz",
		Method: sock.RequestMethodGet,
	})
	if err == nil || res != nil {
		t.Errorf("response/error = %+v/%v, want nil and an error", res, err)
	}
}
