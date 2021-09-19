package api

import (
	"context"
	"reflect"
	"testing"
)

func setupSession(t *testing.T, doc string) *QuerySession {
	session := NewQuerySession()
	err := session.InitWithManifest([]byte(doc))
	if err != nil {
		t.Logf("Could not init session")
		t.Logf("Has manifest definition changed?")
		t.Fatalf("%v", err)
	}
	return session
}

/*
 * This test is subtly more valuable than it lets on.
 *
 * There is a chance that the communication between C++ and go is brittle,
 * sending json.RawString (alias for []byte) about and praying that it decodes
 * into the right thing.
 */
func TestLinenumbersReturnsExpected(t *testing.T) {
	doc := `{
		"format-version": 1,
		"guid": "<some-id>",
		"data": [{
				"file-extension": "f32",
				"filters": [],
				"shapes": [[3, 3, 3]],
				"prefix": "src",
				"resolution": "source"
			}],
		"attributes": [],
		"line-numbers": [
				[9961, 9963, 9965],
				[1961, 1962, 1963],
				[0, 4000, 8000]
			],
		"line-labels": ["inline", "crossline", "time"]
	} `
	qctx := queryContext {
		session: setupSession(t, doc),
	}
	ctx := setQueryContext(context.Background(), &qctx)
	c := cube {}
	numbers, err := c.Linenumbers(ctx)

	expected := [][]int32{
		{9961, 9963, 9965},
		{1961, 1962, 1963},
		{   0, 4000, 8000},
	}
	if err != nil {
		t.Errorf("expected success; got %v", err)
	}
	if !reflect.DeepEqual(numbers, expected) {
		t.Errorf("expected %v; got %v", expected, numbers)
	}
}

func TestFilenameOnUploadSucceedsOnKeyMiss(t *testing.T) {
	doc := `{
		"format-version": 1,
		"guid": "<some-id>",
		"data": [{
				"file-extension": "f32",
				"filters": [],
				"shapes": [[3, 3, 3]],
				"prefix": "src",
				"resolution": "source"
			}],
		"attributes": [],
		"line-numbers": [
				[9961, 9963, 9965],
				[1961, 1962, 1963],
				[0, 4000, 8000]
			],
		"line-labels": ["inline", "crossline", "time"]
	}`
	qctx := queryContext {
		session: setupSession(t, doc),
	}
	ctx := setQueryContext(context.Background(), &qctx)
	c := cube {}
	fname, err := c.FilenameOnUpload(ctx)
	if err != nil {
		t.Errorf("expected success; got %v", err)
	}
	if fname != nil {
		t.Errorf("expected fname = <nil>; got %v", fname)
	}
}

func TestFilenameOnUploadSucceeds(t *testing.T) {
	doc := `{
		"format-version": 1,
		"guid": "<some-id>",
		"upload-filename": "some-filename",
		"data": [{
				"file-extension": "f32",
				"filters": [],
				"shapes": [[3, 3, 3]],
				"prefix": "src",
				"resolution": "source"
			}],
		"attributes": [],
		"line-numbers": [
				[9961, 9963, 9965],
				[1961, 1962, 1963],
				[0, 4000, 8000]
			],
		"line-labels": ["inline", "crossline", "time"]
	}`
	qctx := queryContext {
		session: setupSession(t, doc),
	}
	ctx := setQueryContext(context.Background(), &qctx)
	c := cube {}
	fname, err := c.FilenameOnUpload(ctx)
	if err != nil {
		t.Errorf("expected success; got %v", err)
	}
	if fname == nil {
		t.Errorf("expected fname; got <nil>")
	} else if *fname != "some-filename" {
		t.Errorf("expected fname = 'some-filename'; got %v", *fname)
	}
}
