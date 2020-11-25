package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBadGUIDParam(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{
		gin.Param{Key: "guid",      Value: ""},
		gin.Param{Key: "dimension", Value: "0"},
		gin.Param{Key: "lineno",    Value: "0"},
	}

	expected := "guid empty";
	_, err := parseSliceParams(c)
	if err == nil {
		t.Errorf("parseSliceParams didn't fail on empty guid")
	} else if !strings.HasPrefix(err.Error(), expected) {
		t.Errorf("expected error with prefix '%s', was %v", expected, err)
	}
}

func TestBadDimensionParam(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{
		gin.Param{Key: "guid",      Value: "some-guid"},
		gin.Param{Key: "dimension", Value: "foo"},
		gin.Param{Key: "lineno",    Value: "0"},
	}

	expected := "error parsing dimension"
	_, err := parseSliceParams(c)
	if err == nil {
		t.Errorf("parseSliceParams didn't fail on bad dimension")
	} else if !strings.HasPrefix(err.Error(), expected) {
		t.Errorf("expected error with prefix '%s', was %v", expected, err)
	}
}

func TestBadLinenoParam(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{
		gin.Param{Key: "guid",      Value: "some-guid"},
		gin.Param{Key: "dimension", Value: "0"},
		gin.Param{Key: "lineno",    Value: "foo"},
	}

	expected := "error parsing lineno"
	_, err := parseSliceParams(c)
	if err == nil {
		t.Errorf("parseSliceParams didn't fail on bad lineno")
	} else if !strings.HasPrefix(err.Error(), expected) {
		t.Errorf("expected error with prefix '%s', was %v", expected, err)
	}
}
