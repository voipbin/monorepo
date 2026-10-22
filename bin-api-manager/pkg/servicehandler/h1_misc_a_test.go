package servicehandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	commonidentity "monorepo/bin-common-handler/models/identity"
	omoutdial "monorepo/bin-outdial-manager/models/outdial"
	smfile "monorepo/bin-storage-manager/models/file"
)

// ---- outbound config ----

func h1OutboundConfig(customerID string) *cmoutboundconfig.OutboundConfig {
	return &cmoutboundconfig.OutboundConfig{
		ID:         h1UUID(h1ResourceID),
		CustomerID: h1UUID(customerID),
		Name:       "oc",
	}
}

func h1OutboundConfigWebhook(customerID string) *cmoutboundconfig.WebhookMessage {
	return &cmoutboundconfig.WebhookMessage{
		ID:         h1UUID(h1ResourceID),
		CustomerID: h1UUID(customerID),
		Name:       "oc",
	}
}

// h1OutboundConfigSuperCases builds the table for the super-admin operations
// that look the config up after the permission check.
func h1OutboundConfigSuperCases(
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error),
	expectAct func(m *h1Mocks, ret *cmoutboundconfig.OutboundConfig, err error),
) []h1Case {
	id := h1UUID(h1ResourceID)
	now := h1Now()

	return []h1Case{
		{
			name:    "direct access",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Direct()) },
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name:    "customer admin is not super admin",
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Admin()) },
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:    "get error",
			setup:   func(m *h1Mocks) { m.req.EXPECT().CallV1OutboundConfigGet(h1Ctx(), id).Return(nil, errH1RPC) },
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Super()) },
			wantErr: errH1RPC,
		},
		{
			name: "config without id returns a plain not-found message",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CallV1OutboundConfigGet(h1Ctx(), id).Return(&cmoutboundconfig.OutboundConfig{}, nil)
			},
			call:        func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Super()) },
			wantErrText: "not found: outbound_config",
		},
		{
			name: "deleted config",
			setup: func(m *h1Mocks) {
				c := h1OutboundConfig(h1CustomerID)
				c.TMDelete = &now
				m.req.EXPECT().CallV1OutboundConfigGet(h1Ctx(), id).Return(c, nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Super()) },
			wantErr: serviceerrors.ErrStateInvalid,
		},
		{
			name: "action rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CallV1OutboundConfigGet(h1Ctx(), id).Return(h1OutboundConfig(h1CustomerID), nil)
				expectAct(m, nil, errH1RPC)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Super()) },
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CallV1OutboundConfigGet(h1Ctx(), id).Return(h1OutboundConfig(h1CustomerID), nil)
				expectAct(m, h1OutboundConfig(h1CustomerID), nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Super()) },
			wantRes: h1OutboundConfigWebhook(h1CustomerID),
		},
	}
}

func Test_OutboundConfigCreate_Matrix(t *testing.T) {
	req := &cmoutboundconfig.UpdateRequest{}
	other := h1UUID(h1OtherID)

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigCreate(ctx, h1Direct(), req)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "customer admin is not super admin",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigCreate(ctx, h1Admin(), req)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:  "rpc error",
			setup: func(m *h1Mocks) { m.req.EXPECT().CallV1OutboundConfigCreate(h1Ctx(), other, req).Return(nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigCreate(ctx, h1Super(), req)
			},
			wantErr: errH1RPC,
		},
		{
			name: "success is created for the caller's customer",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CallV1OutboundConfigCreate(h1Ctx(), other, req).Return(h1OutboundConfig(h1OtherID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigCreate(ctx, h1Super(), req)
			},
			wantRes: h1OutboundConfigWebhook(h1OtherID),
		},
	})
}

func Test_OutboundConfigDelete_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1OutboundConfigSuperCases(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.OutboundConfigDelete(ctx, a, id)
		},
		func(m *h1Mocks, ret *cmoutboundconfig.OutboundConfig, err error) {
			m.req.EXPECT().CallV1OutboundConfigDelete(h1Ctx(), id).Return(ret, err)
		},
	))
}

func Test_OutboundConfigUpdate_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	req := &cmoutboundconfig.UpdateRequest{}
	h1Run(t, h1OutboundConfigSuperCases(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.OutboundConfigUpdate(ctx, a, id, req)
		},
		func(m *h1Mocks, ret *cmoutboundconfig.OutboundConfig, err error) {
			m.req.EXPECT().CallV1OutboundConfigUpdate(h1Ctx(), id, req).Return(ret, err)
		},
	))
}

func Test_OutboundConfigGet_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	// Get has no action RPC: keep the lookup cases and re-point the success case.
	var tests []h1Case
	for _, c := range h1OutboundConfigSuperCases(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.OutboundConfigGet(ctx, a, id)
		},
		func(m *h1Mocks, ret *cmoutboundconfig.OutboundConfig, err error) {},
	) {
		if c.name == "action rpc error" {
			continue
		}
		if c.name == "success" {
			c.setup = func(m *h1Mocks) {
				m.req.EXPECT().CallV1OutboundConfigGet(h1Ctx(), id).Return(h1OutboundConfig(h1CustomerID), nil)
			}
		}
		tests = append(tests, c)
	}
	h1Run(t, tests)
}

func Test_OutboundConfigList_Matrix(t *testing.T) {
	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigList(ctx, h1Direct(), 5, "t")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "customer admin is not super admin",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigList(ctx, h1Admin(), 5, "t")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "empty token and rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().CallV1OutboundConfigList(h1Ctx(), uuid.Nil, uint64(5), "2026-01-01T00:00:00.000000Z").Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigList(ctx, h1Super(), 5, "")
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().CallV1OutboundConfigList(h1Ctx(), uuid.Nil, uint64(5), "tok").Return([]cmoutboundconfig.OutboundConfig{*h1OutboundConfig(h1CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigList(ctx, h1Super(), 5, "tok")
			},
			wantRes: []*cmoutboundconfig.WebhookMessage{h1OutboundConfigWebhook(h1CustomerID)},
		},
	})
}

func Test_OutboundConfigSelfUpdate_Matrix(t *testing.T) {
	cid := h1UUID(h1CustomerID)
	id := h1UUID(h1ResourceID)
	req := &cmoutboundconfig.UpdateRequest{}
	listOwn := func(m *h1Mocks, ret []cmoutboundconfig.OutboundConfig, err error) {
		m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
		m.req.EXPECT().CallV1OutboundConfigList(h1Ctx(), cid, uint64(1), "2026-01-01T00:00:00.000000Z").Return(ret, err)
	}

	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigSelfUpdate(ctx, h1Direct(), req)
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "manager is not admin",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigSelfUpdate(ctx, h1Manager(), req)
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name:  "list error",
			setup: func(m *h1Mocks) { listOwn(m, nil, errH1RPC) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigSelfUpdate(ctx, h1Admin(), req)
			},
			wantErr: errH1RPC,
		},
		{
			name:  "no config for the customer",
			setup: func(m *h1Mocks) { listOwn(m, []cmoutboundconfig.OutboundConfig{}, nil) },
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigSelfUpdate(ctx, h1Admin(), req)
			},
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name: "update error",
			setup: func(m *h1Mocks) {
				listOwn(m, []cmoutboundconfig.OutboundConfig{*h1OutboundConfig(h1CustomerID)}, nil)
				m.req.EXPECT().CallV1OutboundConfigUpdate(h1Ctx(), id, req).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigSelfUpdate(ctx, h1Admin(), req)
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				listOwn(m, []cmoutboundconfig.OutboundConfig{*h1OutboundConfig(h1CustomerID)}, nil)
				m.req.EXPECT().CallV1OutboundConfigUpdate(h1Ctx(), id, req).Return(h1OutboundConfig(h1CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigSelfUpdate(ctx, h1Admin(), req)
			},
			wantRes: h1OutboundConfigWebhook(h1CustomerID),
		},
	})
}

func Test_OutboundConfigSelfGet_Matrix(t *testing.T) {
	cid := h1UUID(h1CustomerID)
	h1Run(t, []h1Case{
		{
			name: "no config for the customer",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().CallV1OutboundConfigList(h1Ctx(), cid, uint64(1), "2026-01-01T00:00:00.000000Z").Return([]cmoutboundconfig.OutboundConfig{}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigSelfGet(ctx, h1Admin())
			},
			wantErr: serviceerrors.ErrNotFound,
		},
		{
			name: "manager is not admin",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutboundConfigSelfGet(ctx, h1Manager())
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
	})
}

// ---- outdials ----

func h1Outdial(customerID string) *omoutdial.Outdial {
	return &omoutdial.Outdial{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		Name: "od",
	}
}

func h1OutdialWebhook(customerID string) *omoutdial.WebhookMessage {
	return &omoutdial.WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		Name: "od",
	}
}

func h1OutdialSpec(
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error),
	actExpect func(m *h1Mocks, ret *omoutdial.Outdial, err error),
) h1ActSpec {
	id := h1UUID(h1ResourceID)
	spec := h1ActSpec{
		call:    call,
		getOK:   func(m *h1Mocks, c string) { m.req.EXPECT().OutdialV1OutdialGet(h1Ctx(), id).Return(h1Outdial(c), nil) },
		getErr:  func(m *h1Mocks, err error) { m.req.EXPECT().OutdialV1OutdialGet(h1Ctx(), id).Return(nil, err) },
		okAgent: h1Manager(),
		want:    h1OutdialWebhook(h1CustomerID),
	}
	if actExpect != nil {
		spec.actExpect = func(m *h1Mocks, c string, err error) {
			if err != nil {
				actExpect(m, nil, err)
				return
			}
			actExpect(m, h1Outdial(c), nil)
		}
	}
	return spec
}

func Test_OutdialGet_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1OutdialSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.OutdialGet(ctx, a, id)
		}, nil)))
}

func Test_OutdialDelete_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1OutdialSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.OutdialDelete(ctx, a, id)
		},
		func(m *h1Mocks, ret *omoutdial.Outdial, err error) {
			m.req.EXPECT().OutdialV1OutdialDelete(h1Ctx(), id).Return(ret, err)
		})))
}

func Test_OutdialUpdateBasicInfo_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1OutdialSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.OutdialUpdateBasicInfo(ctx, a, id, "n", "d")
		},
		func(m *h1Mocks, ret *omoutdial.Outdial, err error) {
			m.req.EXPECT().OutdialV1OutdialUpdateBasicInfo(h1Ctx(), id, "n", "d").Return(ret, err)
		})))
}

func Test_OutdialUpdateCampaignID_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	cmp := h1UUID(h1AgentID)
	h1Run(t, h1ActCases(h1OutdialSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.OutdialUpdateCampaignID(ctx, a, id, cmp)
		},
		func(m *h1Mocks, ret *omoutdial.Outdial, err error) {
			m.req.EXPECT().OutdialV1OutdialUpdateCampaignID(h1Ctx(), id, cmp).Return(ret, err)
		})))
}

func Test_OutdialUpdateData_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1OutdialSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.OutdialUpdateData(ctx, a, id, "data")
		},
		func(m *h1Mocks, ret *omoutdial.Outdial, err error) {
			m.req.EXPECT().OutdialV1OutdialUpdateData(h1Ctx(), id, "data").Return(ret, err)
		})))
}

func Test_OutdialCreate_Matrix(t *testing.T) {
	cid := h1UUID(h1CustomerID)
	cmp := h1UUID(h1AgentID)
	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutdialCreate(ctx, h1Direct(), cmp, "n", "d", "x")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutdialCreate(ctx, h1NoPerm(), cmp, "n", "d", "x")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().OutdialV1OutdialCreate(h1Ctx(), cid, cmp, "n", "d", "x").Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutdialCreate(ctx, h1Admin(), cmp, "n", "d", "x")
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().OutdialV1OutdialCreate(h1Ctx(), cid, cmp, "n", "d", "x").Return(h1Outdial(h1CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutdialCreate(ctx, h1Manager(), cmp, "n", "d", "x")
			},
			wantRes: h1OutdialWebhook(h1CustomerID),
		},
	})
}

func Test_OutdialGetsByCustomerID_Matrix(t *testing.T) {
	filters := map[omoutdial.Field]any{omoutdial.FieldCustomerID: h1UUID(h1CustomerID)}
	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutdialGetsByCustomerID(ctx, h1Direct(), 5, "t")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutdialGetsByCustomerID(ctx, h1NoPerm(), 5, "t")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
		{
			name: "empty token and rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().OutdialV1OutdialList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(5), filters).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutdialGetsByCustomerID(ctx, h1Admin(), 5, "")
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().OutdialV1OutdialList(h1Ctx(), "tok", uint64(5), filters).Return([]omoutdial.Outdial{*h1Outdial(h1CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.OutdialGetsByCustomerID(ctx, h1Manager(), 5, "tok")
			},
			wantRes: []*omoutdial.WebhookMessage{h1OutdialWebhook(h1CustomerID)},
		},
	})
}

// ---- storage files ----

func h1File(customerID string) *smfile.File {
	return &smfile.File{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		Name: "f",
	}
}

func h1FileWebhook(customerID string) *smfile.WebhookMessage {
	return &smfile.WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         h1UUID(h1ResourceID),
			CustomerID: h1UUID(customerID),
		},
		Name: "f",
	}
}

func h1FileSpec(
	call func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error),
	actExpect func(m *h1Mocks, ret *smfile.File, err error),
) h1ActSpec {
	id := h1UUID(h1ResourceID)
	now := h1Now()
	spec := h1ActSpec{
		call:   call,
		getOK:  func(m *h1Mocks, c string) { m.req.EXPECT().StorageV1FileGet(h1Ctx(), id).Return(h1File(c), nil) },
		getErr: func(m *h1Mocks, err error) { m.req.EXPECT().StorageV1FileGet(h1Ctx(), id).Return(nil, err) },
		getDeleted: func(m *h1Mocks) {
			f := h1File(h1CustomerID)
			f.TMDelete = &now
			m.req.EXPECT().StorageV1FileGet(h1Ctx(), id).Return(f, nil)
		},
		okAgent: h1Manager(),
		want:    h1FileWebhook(h1CustomerID),
	}
	if actExpect != nil {
		spec.actExpect = func(m *h1Mocks, c string, err error) {
			if err != nil {
				actExpect(m, nil, err)
				return
			}
			actExpect(m, h1File(c), nil)
		}
	}
	return spec
}

func Test_StorageFileGet_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1FileSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.StorageFileGet(ctx, a, id)
		}, nil)))
}

func Test_StorageFileDelete_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	h1Run(t, h1ActCases(h1FileSpec(
		func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
			return h.StorageFileDelete(ctx, a, id)
		},
		func(m *h1Mocks, ret *smfile.File, err error) {
			m.req.EXPECT().StorageV1FileDelete(h1Ctx(), id, 60000).Return(ret, err)
		})))
}

func Test_StorageFileDownloadRedirect_Matrix(t *testing.T) {
	id := h1UUID(h1ResourceID)
	call := func(ctx context.Context, h *serviceHandler, a *h1Identity) (any, error) {
		return h.StorageFileDownloadRedirect(ctx, a, id)
	}
	spec := h1FileSpec(call, nil)
	// the refresh call returns a string, not a file: adapt the action expectations
	spec.actExpect = func(m *h1Mocks, c string, err error) {
		if err != nil {
			m.req.EXPECT().StorageV1FileDownloadURIRefresh(h1Ctx(), id).Return("", err)
			return
		}
		m.req.EXPECT().StorageV1FileDownloadURIRefresh(h1Ctx(), id).Return("https://example.com/refreshed", nil)
	}
	spec.want = "https://example.com/refreshed"

	tests := h1ActCases(spec)

	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)
	tests = append(tests,
		h1Case{
			name: "valid cached download uri is returned without refresh",
			setup: func(m *h1Mocks) {
				f := h1File(h1CustomerID)
				f.TMDownloadExpire = &future
				f.URIDownload = "https://example.com/cached"
				m.req.EXPECT().StorageV1FileGet(h1Ctx(), id).Return(f, nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Manager()) },
			wantRes: "https://example.com/cached",
		},
		h1Case{
			name: "expired cached download uri is refreshed",
			setup: func(m *h1Mocks) {
				f := h1File(h1CustomerID)
				f.TMDownloadExpire = &past
				f.URIDownload = "https://example.com/stale"
				m.req.EXPECT().StorageV1FileGet(h1Ctx(), id).Return(f, nil)
				m.req.EXPECT().StorageV1FileDownloadURIRefresh(h1Ctx(), id).Return("https://example.com/refreshed", nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Manager()) },
			wantRes: "https://example.com/refreshed",
		},
		h1Case{
			name: "unexpired but empty cached uri is refreshed",
			setup: func(m *h1Mocks) {
				f := h1File(h1CustomerID)
				f.TMDownloadExpire = &future
				m.req.EXPECT().StorageV1FileGet(h1Ctx(), id).Return(f, nil)
				m.req.EXPECT().StorageV1FileDownloadURIRefresh(h1Ctx(), id).Return("https://example.com/refreshed", nil)
			},
			call:    func(ctx context.Context, h *serviceHandler) (any, error) { return call(ctx, h, h1Manager()) },
			wantRes: "https://example.com/refreshed",
		},
	)

	h1Run(t, tests)
}

func Test_StorageFileCreate_Guards(t *testing.T) {
	// The upload path needs a real GCS client, only the guards before it are testable.
	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.StorageFileCreate(ctx, h1Direct(), nil, smfile.TypeNone, "n", "d", "f")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "no permission",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.StorageFileCreate(ctx, h1NoPerm(), nil, smfile.TypeNone, "n", "d", "f")
			},
			wantErr: serviceerrors.ErrPermissionDenied,
		},
	})
}

func Test_storageFileCreate_Paths(t *testing.T) {
	cid := h1UUID(h1CustomerID)
	owner := h1UUID(h1AgentID)

	h1Run(t, []h1Case{
		{
			name: "rpc error",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().StorageV1FileCreate(h1Ctx(), cid, owner, smfile.ReferenceTypeNone, uuid.Nil, smfile.TypeNone, "n", "d", "f", "bucket", "storage/x", 60000).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.storageFileCreate(ctx, cid, owner, smfile.ReferenceTypeNone, uuid.Nil, smfile.TypeNone, "n", "d", "f", "bucket", "storage/x")
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().StorageV1FileCreate(h1Ctx(), cid, owner, smfile.ReferenceTypeNone, uuid.Nil, smfile.TypeNone, "n", "d", "f", "bucket", "storage/x", 60000).Return(h1File(h1CustomerID), nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.storageFileCreate(ctx, cid, owner, smfile.ReferenceTypeNone, uuid.Nil, smfile.TypeNone, "n", "d", "f", "bucket", "storage/x")
			},
			wantRes: h1File(h1CustomerID),
		},
	})
}

func Test_StorageFileList_Matrix(t *testing.T) {
	filters := map[smfile.Field]any{
		smfile.FieldCustomerID: h1UUID(h1CustomerID),
		smfile.FieldDeleted:    false,
	}
	h1Run(t, []h1Case{
		{
			name: "direct access",
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.StorageFileList(ctx, h1Direct(), 5, "t")
			},
			wantErr: serviceerrors.ErrDirectAccessNotSupported,
		},
		{
			name: "empty token and rpc error",
			setup: func(m *h1Mocks) {
				m.util.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.req.EXPECT().StorageV1FileList(h1Ctx(), "2026-01-01T00:00:00.000000Z", uint64(5), filters).Return(nil, errH1RPC)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.StorageFileList(ctx, h1Admin(), 5, "")
			},
			wantErr: errH1RPC,
		},
		{
			name: "success",
			setup: func(m *h1Mocks) {
				m.req.EXPECT().StorageV1FileList(h1Ctx(), "tok", uint64(5), filters).Return([]smfile.File{*h1File(h1CustomerID)}, nil)
			},
			call: func(ctx context.Context, h *serviceHandler) (any, error) {
				return h.StorageFileList(ctx, h1Admin(), 5, "tok")
			},
			wantRes: []*smfile.WebhookMessage{h1FileWebhook(h1CustomerID)},
		},
	})
}
