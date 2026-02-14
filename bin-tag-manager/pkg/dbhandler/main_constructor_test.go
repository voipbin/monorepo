package dbhandler

import (
	"testing"

	"monorepo/bin-tag-manager/pkg/cachehandler"

	gomock "go.uber.org/mock/gomock"
)

func TestNewHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := NewHandler(dbTest, mockCache)

	if h == nil {
		t.Errorf("Expected handler, got nil")
	}
}
