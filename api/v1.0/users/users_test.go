package users

// func setupServer(app *gin.Engine) {
// 	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
// 	ApplyRoutes(v1)
// }

// func TestUsersPOST(t *testing.T) {

// 	// create mock
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockSvc := servicehandler.NewMockServiceHandler(mc)

// 	type test struct {
// 		name string
// 		user user.User
// 		req  request.BodyUsersPOST
// 	}

// 	tests := []test{
// 		{
// 			"admin permission user",
// 			user.User{
// 				ID:         1,
// 				Permission: user.PermissionAdmin,
// 			},
// 			request.BodyUsersPOST{
// 				Username:   "username-0a790e7a-f15c-11ea-9582-7f1f242cd6f8",
// 				Password:   "password-0d6e5248-f15c-11ea-b916-379c5bd787a4",
// 				Name:       "name1",
// 				Detail:     "detail1",
// 				Permission: uint64(user.PermissionAdmin),
// 			},
// 		},
// 		{
// 			"none permission user",
// 			user.User{
// 				ID:         1,
// 				Permission: user.PermissionAdmin,
// 			},
// 			request.BodyUsersPOST{
// 				Username:   "username-4880078e-f15f-11ea-afe3-ebf4eb79cf50",
// 				Password:   "password-4bb9412c-f15f-11ea-b37b-533f30888631",
// 				Name:       "",
// 				Detail:     "",
// 				Permission: uint64(user.PermissionNone),
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			w := httptest.NewRecorder()
// 			_, r := gin.CreateTestContext(w)

// 			r.Use(func(c *gin.Context) {
// 				c.Set(common.OBJServiceHandler, mockSvc)
// 				c.Set("customer", tt.user)
// 			})
// 			setupServer(r)

// 			// create body
// 			body, err := json.Marshal(tt.req)
// 			if err != nil {
// 				t.Errorf("Could not marshal the request. err: %v", err)
// 			}

// 			mockSvc.EXPECT().UserCreate(&tt.user, tt.req.Username, tt.req.Password, tt.req.Name, tt.req.Detail, user.Permission(tt.req.Permission)).Return(&user.User{}, nil)
// 			req, _ := http.NewRequest("POST", "/v1.0/users", bytes.NewBuffer(body))
// 			req.Header.Set("Content-Type", "application/json")

// 			r.ServeHTTP(w, req)
// 			if w.Code != http.StatusOK {
// 				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
// 			}
// 		})
// 	}
// }

// func TestUsersGET(t *testing.T) {

// 	// create mock
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockSvc := servicehandler.NewMockServiceHandler(mc)

// 	type test struct {
// 		name      string
// 		user      user.User
// 		pageSize  uint64
// 		pageToken string

// 		res []*user.User
// 	}

// 	tests := []test{
// 		{
// 			"admin permission user",
// 			user.User{
// 				ID:         1,
// 				Permission: user.PermissionAdmin,
// 			},
// 			10,
// 			"2021-01-29 03:18:22.131000",
// 			[]*user.User{
// 				{
// 					ID:         1,
// 					Username:   "test1",
// 					Name:       "test1",
// 					Detail:     "test1",
// 					Permission: 1,
// 					TMCreate:   "",
// 					TMUpdate:   "",
// 					TMDelete:   "",
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			w := httptest.NewRecorder()
// 			_, r := gin.CreateTestContext(w)

// 			r.Use(func(c *gin.Context) {
// 				c.Set(common.OBJServiceHandler, mockSvc)
// 				c.Set("customer", tt.user)
// 			})
// 			setupServer(r)

// 			mockSvc.EXPECT().UserGets(&tt.user, tt.pageSize, tt.pageToken).Return(tt.res, nil)

// 			reqQuery := fmt.Sprintf("/v1.0/users?page_size=%d&page_token=%s", tt.pageSize, tt.pageToken)
// 			req, _ := http.NewRequest("GET", reqQuery, nil)
// 			req.Header.Set("Content-Type", "application/json")

// 			r.ServeHTTP(w, req)
// 			if w.Code != http.StatusOK {
// 				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
// 			}
// 		})
// 	}
// }

// func TestUsersIDGET(t *testing.T) {

// 	// create mock
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockSvc := servicehandler.NewMockServiceHandler(mc)

// 	tests := []struct {
// 		name string
// 		user user.User

// 		reqQuery string
// 		id       uint64

// 		res *user.User
// 	}{
// 		{
// 			"admin permission user",
// 			user.User{
// 				ID:         1,
// 				Permission: user.PermissionAdmin,
// 			},

// 			"/v1.0/users/1",
// 			1,

// 			&user.User{
// 				ID:         1,
// 				Username:   "test1",
// 				Name:       "test1",
// 				Detail:     "test1",
// 				Permission: 1,
// 				TMCreate:   "",
// 				TMUpdate:   "",
// 				TMDelete:   "",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			w := httptest.NewRecorder()
// 			_, r := gin.CreateTestContext(w)

// 			r.Use(func(c *gin.Context) {
// 				c.Set(common.OBJServiceHandler, mockSvc)
// 				c.Set("customer", tt.user)
// 			})
// 			setupServer(r)

// 			mockSvc.EXPECT().UserGet(&tt.user, tt.id).Return(tt.res, nil)

// 			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
// 			req.Header.Set("Content-Type", "application/json")

// 			r.ServeHTTP(w, req)
// 			if w.Code != http.StatusOK {
// 				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
// 			}
// 		})
// 	}
// }

// func TestUsersIDPUT(t *testing.T) {

// 	// create mock
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockSvc := servicehandler.NewMockServiceHandler(mc)

// 	tests := []struct {
// 		name string
// 		user user.User

// 		reqQuery string
// 		reqBody  []byte
// 		id       uint64
// 		userName string
// 		detail   string

// 		res *user.User
// 	}{
// 		{
// 			"admin permission user",
// 			user.User{
// 				ID:         2,
// 				Permission: user.PermissionAdmin,
// 			},

// 			"/v1.0/users/2",
// 			[]byte(`{"name":"name2","detail":"detail2"}`),
// 			2,
// 			"name2",
// 			"detail2",

// 			&user.User{
// 				ID:         2,
// 				Username:   "test1",
// 				Name:       "name2",
// 				Detail:     "detail2",
// 				Permission: 1,
// 				TMCreate:   "",
// 				TMUpdate:   "",
// 				TMDelete:   "",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			w := httptest.NewRecorder()
// 			_, r := gin.CreateTestContext(w)

// 			r.Use(func(c *gin.Context) {
// 				c.Set(common.OBJServiceHandler, mockSvc)
// 				c.Set("customer", tt.user)
// 			})
// 			setupServer(r)

// 			mockSvc.EXPECT().UserUpdate(&tt.user, uint64(tt.id), tt.userName, tt.detail).Return(nil)

// 			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
// 			req.Header.Set("Content-Type", "application/json")

// 			r.ServeHTTP(w, req)
// 			if w.Code != http.StatusOK {
// 				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
// 			}
// 		})
// 	}
// }

// func TestUsersIDPasswordPUT(t *testing.T) {

// 	// create mock
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockSvc := servicehandler.NewMockServiceHandler(mc)

// 	tests := []struct {
// 		name string
// 		user user.User

// 		reqQuery string
// 		reqBody  []byte
// 		id       uint64
// 		password string
// 	}{
// 		{
// 			"admin permission user",
// 			user.User{
// 				ID:         2,
// 				Permission: user.PermissionAdmin,
// 			},

// 			"/v1.0/users/2/password",
// 			[]byte(`{"password":"password2"}`),
// 			2,
// 			"password2",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			w := httptest.NewRecorder()
// 			_, r := gin.CreateTestContext(w)

// 			r.Use(func(c *gin.Context) {
// 				c.Set(common.OBJServiceHandler, mockSvc)
// 				c.Set("customer", tt.user)
// 			})
// 			setupServer(r)

// 			mockSvc.EXPECT().UserUpdatePassword(&tt.user, uint64(tt.id), tt.password).Return(nil)

// 			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
// 			req.Header.Set("Content-Type", "application/json")

// 			r.ServeHTTP(w, req)
// 			if w.Code != http.StatusOK {
// 				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
// 			}
// 		})
// 	}
// }

// func TestUsersIDPermissionPUT(t *testing.T) {

// 	// create mock
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockSvc := servicehandler.NewMockServiceHandler(mc)

// 	tests := []struct {
// 		name string
// 		user user.User

// 		reqQuery   string
// 		reqBody    []byte
// 		id         uint64
// 		permission user.Permission
// 	}{
// 		{
// 			"admin permission user",
// 			user.User{
// 				ID:         2,
// 				Permission: user.PermissionAdmin,
// 			},

// 			"/v1.0/users/2/permission",
// 			[]byte(`{"permission":0}`),
// 			2,
// 			0,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			w := httptest.NewRecorder()
// 			_, r := gin.CreateTestContext(w)

// 			r.Use(func(c *gin.Context) {
// 				c.Set(common.OBJServiceHandler, mockSvc)
// 				c.Set("customer", tt.user)
// 			})
// 			setupServer(r)

// 			mockSvc.EXPECT().UserUpdatePermission(&tt.user, uint64(tt.id), tt.permission).Return(nil)

// 			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
// 			req.Header.Set("Content-Type", "application/json")

// 			r.ServeHTTP(w, req)
// 			if w.Code != http.StatusOK {
// 				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
// 			}
// 		})
// 	}
// }
