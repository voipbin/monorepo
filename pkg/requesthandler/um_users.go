package requesthandler

// // UMV1UserGets sends a request to user-manager
// // to get users.
// // it returns user if it succeed.
// func (r *requestHandler) UMV1UserGets(ctx context.Context, pageToken string, pageSize uint64) ([]umuser.User, error) {
// 	uri := fmt.Sprintf("/v1/users?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

// 	tmp, err := r.sendRequestUM(uri, rabbitmqhandler.RequestMethodGet, resourceUMUsers, requestTimeoutDefault, 0, ContentTypeJSON, nil)
// 	switch {
// 	case err != nil:
// 		return nil, err
// 	case tmp == nil:
// 		// not found
// 		return nil, fmt.Errorf("response code: %d", 404)
// 	case tmp.StatusCode > 299:
// 		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
// 	}

// 	var res []umuser.User
// 	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
// 		return nil, err
// 	}

// 	return res, nil
// }

// // UMV1UserGet sends a request to user-manager
// // to get user.
// // it returns user if it succeed.
// func (r *requestHandler) UMV1UserGet(ctx context.Context, id uint64) (*umuser.User, error) {
// 	uri := fmt.Sprintf("/v1/users/%d", id)

// 	tmp, err := r.sendRequestUM(uri, rabbitmqhandler.RequestMethodGet, resourceUMUsers, requestTimeoutDefault, 0, ContentTypeJSON, nil)
// 	switch {
// 	case err != nil:
// 		return nil, err
// 	case tmp == nil:
// 		// not found
// 		return nil, fmt.Errorf("response code: %d", 404)
// 	case tmp.StatusCode > 299:
// 		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
// 	}

// 	var res umuser.User
// 	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
// 		return nil, err
// 	}

// 	return &res, nil
// }

// // UMV1UserDelete sends a request to user-manager
// // to delete the user.
// // it returns error if it failed.
// func (r *requestHandler) UMV1UserDelete(ctx context.Context, id uint64) error {
// 	uri := fmt.Sprintf("/v1/users/%d", id)

// 	tmp, err := r.sendRequestUM(uri, rabbitmqhandler.RequestMethodDelete, resourceUMUsers, requestTimeoutDefault, 0, ContentTypeJSON, nil)
// 	switch {
// 	case err != nil:
// 		return err
// 	case tmp == nil:
// 		// not found
// 		return fmt.Errorf("response code: %d", 404)
// 	case tmp.StatusCode > 299:
// 		return fmt.Errorf("response code: %d", tmp.StatusCode)
// 	}

// 	return nil
// }

// // UMV1UserCreate sends a request to user-manager
// // to create a new user.
// // it returns user if it succeed.
// // timeout: seconds
// func (r *requestHandler) UMV1UserCreate(ctx context.Context, timeout int, username, password, name, detail string, permission umuser.Permission) (*umuser.User, error) {
// 	uri := "/v1/users"

// 	req := &umrequest.V1DataUsersPost{
// 		Username: username,
// 		Password: password,

// 		Name:   name,
// 		Detail: detail,

// 		Permission: uint64(permission),
// 	}

// 	m, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	res, err := r.sendRequestUM(uri, rabbitmqhandler.RequestMethodPost, resourceUMUsers, timeout, 0, ContentTypeJSON, m)
// 	switch {
// 	case err != nil:
// 		return nil, err
// 	case res == nil:
// 		// not found
// 		return nil, fmt.Errorf("response code: %d", 404)
// 	case res.StatusCode > 299:
// 		return nil, fmt.Errorf("response code: %d", res.StatusCode)
// 	}

// 	var data umuser.User
// 	if err := json.Unmarshal([]byte(res.Data), &data); err != nil {
// 		return nil, err
// 	}

// 	return &data, nil
// }

// // UMV1UserLogin sends a request to user-manager
// // to user login.
// // it returns user if it succeed.
// // timeout: seconds
// func (r *requestHandler) UMV1UserLogin(ctx context.Context, timeout int, username, password string) (*umuser.User, error) {
// 	uri := fmt.Sprintf("/v1/users/%s/login", username)

// 	req := &umrequest.V1DataUsersUsernameLoginPost{
// 		Password: password,
// 	}

// 	m, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	tmp, err := r.sendRequestUM(uri, rabbitmqhandler.RequestMethodPost, resourceUMUsers, timeout, 0, ContentTypeJSON, m)
// 	switch {
// 	case err != nil:
// 		return nil, err
// 	case tmp == nil:
// 		// not found
// 		return nil, fmt.Errorf("response code: %d", 404)
// 	case tmp.StatusCode > 299:
// 		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
// 	}

// 	var res umuser.User
// 	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
// 		return nil, err
// 	}

// 	return &res, nil
// }

// // UMV1UserUpdate sends a request to user-manager
// // to update the user's basic info.
// // it returns error if it failed.
// func (r *requestHandler) UMV1UserUpdateBasicInfo(ctx context.Context, userID uint64, name, detail string) error {
// 	uri := fmt.Sprintf("/v1/users/%d", userID)

// 	req := &umrequest.V1DataUsersIDPut{
// 		Name:   name,
// 		Detail: detail,
// 	}

// 	m, err := json.Marshal(req)
// 	if err != nil {
// 		return err
// 	}

// 	res, err := r.sendRequestUM(uri, rabbitmqhandler.RequestMethodPut, resourceUMUsers, requestTimeoutDefault, 0, ContentTypeJSON, m)
// 	switch {
// 	case err != nil:
// 		return err
// 	case res == nil:
// 		// not found
// 		return fmt.Errorf("response code: %d", 404)
// 	case res.StatusCode > 299:
// 		return fmt.Errorf("response code: %d", res.StatusCode)
// 	}

// 	return nil
// }

// // UMV1UserUpdatePermission sends a request to user-manager
// // to update the user's permission.
// // it returns error if it failed.
// func (r *requestHandler) UMV1UserUpdatePermission(ctx context.Context, userID uint64, permission umuser.Permission) error {
// 	uri := fmt.Sprintf("/v1/users/%d/permission", userID)

// 	req := &umrequest.V1DataUsersIDPermissionPut{
// 		Permission: uint64(permission),
// 	}

// 	m, err := json.Marshal(req)
// 	if err != nil {
// 		return err
// 	}

// 	res, err := r.sendRequestUM(uri, rabbitmqhandler.RequestMethodPut, resourceUMUsers, requestTimeoutDefault, 0, ContentTypeJSON, m)
// 	switch {
// 	case err != nil:
// 		return err
// 	case res == nil:
// 		// not found
// 		return fmt.Errorf("response code: %d", 404)
// 	case res.StatusCode > 299:
// 		return fmt.Errorf("response code: %d", res.StatusCode)
// 	}

// 	return nil
// }

// // UMV1UserUpdatePassword sends a request to user-manager
// // to update the user's password.
// // it returns error if it failed.
// // timeout: seconds
// func (r *requestHandler) UMV1UserUpdatePassword(ctx context.Context, timeout int, userID uint64, password string) error {
// 	uri := fmt.Sprintf("/v1/users/%d/password", userID)

// 	req := &umrequest.V1DataUsersIDPasswordPut{
// 		Password: password,
// 	}

// 	m, err := json.Marshal(req)
// 	if err != nil {
// 		return err
// 	}

// 	res, err := r.sendRequestUM(uri, rabbitmqhandler.RequestMethodPut, resourceUMUsers, timeout, 0, ContentTypeJSON, m)
// 	switch {
// 	case err != nil:
// 		return err
// 	case res == nil:
// 		// not found
// 		return fmt.Errorf("response code: %d", 404)
// 	case res.StatusCode > 299:
// 		return fmt.Errorf("response code: %d", res.StatusCode)
// 	}

// 	return nil
// }
