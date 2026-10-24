package filehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/dbhandler"
)

func Test_EventCustomerDeleted_paths(t *testing.T) {
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("3c4d5e6f-0000-0000-0000-000000000001")
	cu := &cmcustomer.Customer{ID: customerID}
	filters := map[file.Field]any{
		file.FieldDeleted:    false,
		file.FieldCustomerID: customerID,
	}

	f1 := &file.File{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("3c4d5e6f-0000-0000-0000-000000000011")}, BucketName: "media", Filepath: "bin/one", Filesize: 10}
	f2 := &file.File{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("3c4d5e6f-0000-0000-0000-000000000012")}, BucketName: "media", Filepath: "bin/two", Filesize: 20}

	t.Run("listing fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &fileHandler{db: mockDB}
		mockDB.EXPECT().FileList(ctx, "", uint64(10000), filters).Return(nil, errTest)

		if err := h.EventCustomerDeleted(ctx, cu); !errors.Is(err, errTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errTest, err)
		}
	})

	t.Run("a customer without files is a no-op", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &fileHandler{db: mockDB}
		mockDB.EXPECT().FileList(ctx, "", uint64(10000), filters).Return([]*file.File{}, nil)

		if err := h.EventCustomerDeleted(ctx, cu); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("one failing file does not stop the others", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		mockAccount := accounthandler.NewMockAccountHandler(mc)

		fake := newFakeGCS(t, map[string]int64{"media/bin/one": 10, "media/bin/two": 20})
		h := &fileHandler{db: mockDB, notifyHandler: mockNotify, accountHandler: mockAccount, client: fake.client(t)}

		mockDB.EXPECT().FileList(ctx, "", uint64(10000), filters).Return([]*file.File{f1, f2}, nil)

		// the first file cannot be deleted from the db.
		mockDB.EXPECT().FileDelete(ctx, f1.ID).Return(errTest)

		// the second file goes through the full delete path.
		mockDB.EXPECT().FileDelete(ctx, f2.ID).Return(nil)
		mockDB.EXPECT().FileGet(ctx, f2.ID).Return(f2, nil)
		mockNotify.EXPECT().PublishEvent(ctx, file.EventTypeFileDeleted, f2)
		mockAccount.EXPECT().DecreaseFileInfo(ctx, f2.AccountID, int64(1), int64(20)).Return(&account.Account{}, nil)

		if err := h.EventCustomerDeleted(ctx, cu); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
		if got := fake.deletedKeys(); !reflect.DeepEqual(got, []string{"media/bin/two"}) {
			t.Errorf("Wrong match. expect only the second object to be deleted, got: %v", got)
		}
	})
}
