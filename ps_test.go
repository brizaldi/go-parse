package ps

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
	contentType   string
}{
	{name: "good json", json: `{"foo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "badly formatted json", json: `{"foo":"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type", json: `{"foo": 1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type", json: `{1: 1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json files", json: `{"foo": "bar"}{"alpha": "beta"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "syntax error in json", json: `{"foo": 1"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown field in json", json: `{"fooo": "bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type for field", json: `{"foo": 10.2}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "allow unknown field in json", json: `{"fooo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "missing field name", json: `{jack: "bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "file too large", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 5, allowUnknown: false},
	{name: "not json", json: `Hello, world`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "wrong header", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: false, contentType: "application/xml"},
}

func TestParser_ReadJSON(t *testing.T) {
	for _, e := range jsonTests {
		var testParser Parser
		// set max file size
		testParser.MaxJSONSize = e.maxSize

		// allow/disallow unknown fields.
		testParser.AllowUnknownFields = e.allowUnknown

		// declare a variable to read the decoded json into.
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		// create a request with the body.
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error", err)
		}
		if e.contentType != "" {
			req.Header.Add("Content-Type", e.contentType)
		} else {
			req.Header.Add("Content-Type", "application/json")
		}

		// create a test response recorder, which satisfies the requirements
		// for a ResponseWriter.
		rr := httptest.NewRecorder()

		// call ReadJSON and check for an error.
		err = testParser.ReadJSON(rr, req, &decodedJSON)

		// if we expect an error, but do not get one, something went wrong.
		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none received", e.name)
		}

		// if we do not expect an error, but get one, something went wrong.
		if !e.errorExpected && err != nil {
			t.Errorf("%s: error not expected, but one received: %s \n%s", e.name, err.Error(), e.json)
		}
		req.Body.Close()
	}
}

func TestParser_ReadJSONAndMarshal(t *testing.T) {
	// set max file size
	var testParser Parser

	// create a request with the body
	req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(`{"foo": "bar"}`)))
	if err != nil {
		t.Log("Error", err)
	}

	// create a test response recorder, which satisfies the requirements
	// for a ResponseWriter
	rr := httptest.NewRecorder()

	// call ReadJSON and check for an error; since we are using nil for the final parameter,
	// we should get an error
	err = testParser.ReadJSON(rr, req, nil)

	// we expect an error, but did not get one, so something went wrong
	if err == nil {
		t.Error("error expected, but none received")
	}

	req.Body.Close()
}

var WriteJSONTests = []struct {
	name          string
	payload       any
	errorExpected bool
}{
	{
		name: "valid",
		payload: JSONResponse{
			Error:   false,
			Message: "foo",
		},
		errorExpected: false,
	},
	{
		name:          "invalid",
		payload:       make(chan int),
		errorExpected: true,
	},
}

func TestParser_WriteJSON(t *testing.T) {
	for _, e := range WriteJSONTests {
		// create a variable of type ps.Parser, and just use the defaults.
		var testParser Parser

		rr := httptest.NewRecorder()

		headers := make(http.Header)
		headers.Add("FOO", "BAR")
		err := testParser.WriteJSON(rr, http.StatusOK, e.payload, headers)
		if err == nil && e.errorExpected {
			t.Errorf("%s: expected error, but did not get one", e.name)
		}
		if err != nil && !e.errorExpected {
			t.Errorf("%s: did not expect error, but got one: %v", e.name, err)
		}
	}
}

func TestParser_ErrorJSON(t *testing.T) {
	var testParser Parser

	rr := httptest.NewRecorder()
	err := testParser.ErrorJSON(rr, errors.New("some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var requestPayload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&requestPayload)
	if err != nil {
		t.Error("received error when decoding ErrorJSON payload:", err)
	}

	if !requestPayload.Error {
		t.Error("error set to false in response from ErrorJSON, and should be set to true")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code returned; expected 503, but got %d", rr.Code)
	}
}
