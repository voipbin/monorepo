package api

// func setupServer() *gin.Engine {
// 	app := gin.Default()

// 	ApplyRoutes(app)
// 	return app
// }

// func TestPing(t *testing.T) {

// 	ts := httptest.NewServer(setupServer())
// 	defer ts.Close()

// 	res, err := http.Get(fmt.Sprintf("%s/ping", ts.URL))
// 	if err != nil {
// 		t.Errorf("Wrong match. expect: ok, got: %v", err)
// 	}

// 	if res.StatusCode != 200 {
// 		t.Errorf("Wrong match. expect: 200, got: %d", res.StatusCode)
// 	}
// }
