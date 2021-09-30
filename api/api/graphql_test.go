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

func TestSampleValueMinMaxSucceeds(t *testing.T) {
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
		"sample-value-min" : 1.2100000381469727,
		"sample-value-max" : 5.240489959716797,
		"line-labels": ["inline", "crossline", "time"]
	} `
	qctx := queryContext {
		session: setupSession(t, doc),
	}
	ctx := setQueryContext(context.Background(), &qctx)
	c := cube {}

	sampleValueMin, err := c.SampleValueMin(ctx)
	expected := float64(1.2100000381469727)

	if err != nil {
		t.Errorf("expected success; got %v", err)
	}
	if !reflect.DeepEqual(*sampleValueMin, expected) {
		t.Errorf("expected %v; got %v", expected, *sampleValueMin)
	}

	sampleValueMax, err := c.SampleValueMax(ctx)
	expected = 5.240489959716797

	if err != nil {
		t.Errorf("expected success; got %v", err)
	}
	if *sampleValueMax != expected {
		t.Errorf("expected %v; got %v", expected, *sampleValueMax)
	}
}

func TestSampleValueMinMaxSucceedsOnKeyMiss(t *testing.T) {
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

	sampleValueMin, err := c.SampleValueMin(ctx)
	if err != nil {
		t.Errorf("expected success; got %v", err)
	}
	if sampleValueMin != nil {
		t.Errorf("expected sample-value-min = <nil>; got %v", *sampleValueMin)
	}

	sampleValueMax, err := c.SampleValueMax(ctx)
	if err != nil {
		t.Errorf("expected success; got %v", err)
	}
	if sampleValueMax != nil {
		t.Errorf("expected sample-value-max = <nil>; got %v", *sampleValueMax)
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
