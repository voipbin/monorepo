package dbhandler

import (
	context "context"
	"fmt"
	reflect "reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/cachehandler"
)

func TestQueuecallReferenceCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		data      *queuecallreference.QueuecallReference
		expectRes *queuecallreference.QueuecallReference
	}{
		{
			"normal",
			&queuecallreference.QueuecallReference{
				ID:           uuid.FromStringOrNil("93aeb602-6016-11ec-909b-d30e283ea6cb"),
				CustomerID:   uuid.FromStringOrNil("2d2c0a14-7ffc-11ec-a6f5-ef525309e5e6"),
				Type:         queuecall.ReferenceTypeCall,
				QueuecallIDs: []uuid.UUID{},
				TMCreate:     DefaultTimeStamp,
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
			},
			&queuecallreference.QueuecallReference{
				ID:           uuid.FromStringOrNil("93aeb602-6016-11ec-909b-d30e283ea6cb"),
				CustomerID:   uuid.FromStringOrNil("2d2c0a14-7ffc-11ec-a6f5-ef525309e5e6"),
				Type:         queuecall.ReferenceTypeCall,
				QueuecallIDs: []uuid.UUID{},
				TMCreate:     DefaultTimeStamp,
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().QueuecallReferenceSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallReferenceGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf("")).AnyTimes()

			if err := h.QueuecallReferenceCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallReferenceGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestQueuecallReferenceDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name string

		id uuid.UUID

		data      *queuecallreference.QueuecallReference
		expectRes *queuecallreference.QueuecallReference
	}{
		{
			"test normal",

			uuid.FromStringOrNil("447196b0-762b-11ec-9010-375c278aee23"),

			&queuecallreference.QueuecallReference{
				ID:           uuid.FromStringOrNil("447196b0-762b-11ec-9010-375c278aee23"),
				CustomerID:   uuid.FromStringOrNil("378a2662-7ffc-11ec-8b4b-9b0bb8c4670a"),
				Type:         queuecall.ReferenceTypeCall,
				QueuecallIDs: []uuid.UUID{},
				TMCreate:     DefaultTimeStamp,
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
			},
			&queuecallreference.QueuecallReference{
				ID:           uuid.FromStringOrNil("447196b0-762b-11ec-9010-375c278aee23"),
				CustomerID:   uuid.FromStringOrNil("378a2662-7ffc-11ec-8b4b-9b0bb8c4670a"),
				Type:         queuecall.ReferenceTypeCall,
				QueuecallIDs: []uuid.UUID{},
				TMCreate:     DefaultTimeStamp,
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().QueuecallReferenceSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallReferenceGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallReferenceCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			err := h.QueuecallReferenceDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallReferenceGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}
			if res.TMDelete == "" {
				t.Errorf("Wrong match. expect: not empty, got: empty")
			}

			tt.expectRes.TMCreate = res.TMCreate
			tt.expectRes.TMUpdate = res.TMUpdate
			tt.expectRes.TMDelete = res.TMDelete
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
