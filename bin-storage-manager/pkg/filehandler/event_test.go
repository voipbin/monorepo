package filehandler

// func Test_EventCustomerDeleted(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		customer *cucustomer.Customer

// 		expectFilters map[string]string
// 		responseFiles []*file.File
// 	}{
// 		{
// 			name: "normal",

// 			customer: &cucustomer.Customer{
// 				ID: uuid.FromStringOrNil("cfd3058e-1b6c-11ef-bd2d-037b38fdc758"),
// 			},

// 			expectFilters: map[string]string{
// 				"customer_id": "cfd3058e-1b6c-11ef-bd2d-037b38fdc758",
// 				"deleted":     "false",
// 			},
// 			responseFiles: []*file.File{
// 				{
// 					ID:       uuid.FromStringOrNil("d00ae0e4-1b6c-11ef-87a1-17beb6c9a6bd"),
// 					TMDelete: dbhandler.DefaultTimeStamp,
// 				},
// 				{
// 					ID:       uuid.FromStringOrNil("d038a59c-1b6c-11ef-b359-4314ea110157"),
// 					TMDelete: dbhandler.DefaultTimeStamp,
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockUtil := utilhandler.NewMockUtilHandler(mc)

// 			h := &fileHandler{
// 				db:            mockDB,
// 				notifyHandler: mockNotify,
// 				utilHandler:   mockUtil,
// 			}
// 			ctx := context.Background()

// 			mockUtil.EXPECT().TimeGetCurTime(.Return(utilhandler.TimeGetCurTime())
// 			mockDB.EXPECT().FileGets(ctx, gomock.Any(), uint64(10000), tt.expectFilters.Return(tt.responseFiles, nil)

// 			// delete
// 			for _, f := range tt.responseFiles {

// 				mockDB.EXPECT().FileDelete(ctx, f.ID.Return(nil)
// 				mockDB.EXPECT().FileGet(ctx, f.ID.Return(f, nil)
// 				mockNotify.EXPECT().PublishWebhookEvent(ctx, f.CustomerID, file.EventTypeFileDeleted, f)
// 			}

// 			if err := h.EventCustomerDeleted(ctx, tt.customer); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }
