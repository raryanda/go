package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type (
	bindTestStruct struct {
		I           int
		PtrI        *int
		I8          int8
		PtrI8       *int8
		I16         int16
		PtrI16      *int16
		I32         int32
		PtrI32      *int32
		I64         int64
		PtrI64      *int64
		UI          uint
		PtrUI       *uint
		UI8         uint8
		PtrUI8      *uint8
		UI16        uint16
		PtrUI16     *uint16
		UI32        uint32
		PtrUI32     *uint32
		UI64        uint64
		PtrUI64     *uint64
		B           bool
		PtrB        *bool
		F32         float32
		PtrF32      *float32
		F64         float64
		PtrF64      *float64
		S           string
		PtrS        *string
		cantSet     string
		DoesntExist string
		T           Timestamp
		Tptr        *Timestamp
		SA          StringArray
	}
	Timestamp   time.Time
	TA          []Timestamp
	StringArray []string
	Struct      struct {
		Foo string
	}
)

func (t *Timestamp) UnmarshalParam(src string) error {
	ts, err := time.Parse(time.RFC3339, src)
	*t = Timestamp(ts)
	return err
}

func (a *StringArray) UnmarshalParam(src string) error {
	*a = StringArray(strings.Split(src, ","))
	return nil
}

func (s *Struct) UnmarshalParam(src string) error {
	*s = Struct{
		Foo: src,
	}
	return nil
}

func (t bindTestStruct) GetCantSet() string {
	return t.cantSet
}

var values = map[string][]string{
	"I":       {"0"},
	"PtrI":    {"0"},
	"I8":      {"8"},
	"PtrI8":   {"8"},
	"I16":     {"16"},
	"PtrI16":  {"16"},
	"I32":     {"32"},
	"PtrI32":  {"32"},
	"I64":     {"64"},
	"PtrI64":  {"64"},
	"UI":      {"0"},
	"PtrUI":   {"0"},
	"UI8":     {"8"},
	"PtrUI8":  {"8"},
	"UI16":    {"16"},
	"PtrUI16": {"16"},
	"UI32":    {"32"},
	"PtrUI32": {"32"},
	"UI64":    {"64"},
	"PtrUI64": {"64"},
	"B":       {"true"},
	"PtrB":    {"true"},
	"F32":     {"32.5"},
	"PtrF32":  {"32.5"},
	"F64":     {"64.5"},
	"PtrF64":  {"64.5"},
	"S":       {"test"},
	"PtrS":    {"test"},
	"cantSet": {"test"},
	"T":       {"2016-12-06T19:09:05+01:00"},
	"Tptr":    {"2016-12-06T19:09:05+01:00"},
	"ST":      {"bar"},
}

func TestBindJSON(t *testing.T) {
	assert := assert.New(t)
	testBindOkay(assert, strings.NewReader(userJSON), MIMEApplicationJSON)
	testBindError(assert, strings.NewReader(invalidContent), MIMEApplicationJSON, &json.SyntaxError{})
	testBindError(assert, strings.NewReader(userJSONInvalidType), MIMEApplicationJSON, &json.UnmarshalTypeError{})
}

func TestBindQueryParams(t *testing.T) {
	e := New()
	req := httptest.NewRequest(http.MethodGet, "/?id=1&name=Jon+Snow", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	u := new(user)
	err := c.Bind(u)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, u.ID)
		assert.Equal(t, "Jon Snow", u.Name)
	}
}

func TestBindQueryParamsCaseInsensitive(t *testing.T) {
	e := New()
	req := httptest.NewRequest(http.MethodGet, "/?ID=1&NAME=Jon+Snow", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	u := new(user)
	err := c.Bind(u)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, u.ID)
		assert.Equal(t, "Jon Snow", u.Name)
	}
}

func TestBindQueryParamsCaseSensitivePrioritized(t *testing.T) {
	e := New()
	req := httptest.NewRequest(http.MethodGet, "/?id=1&ID=2&NAME=Jon+Snow&name=Jon+Doe", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	u := new(user)
	err := c.Bind(u)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, u.ID)
		assert.Equal(t, "Jon Doe", u.Name)
	}
}

func TestBindUnmarshalParam(t *testing.T) {
	e := New()
	req := httptest.NewRequest(http.MethodGet, "/?ts=2016-12-06T19:09:05Z&sa=one,two,three&ta=2016-12-06T19:09:05Z&ta=2016-12-06T19:09:05Z&ST=baz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	result := struct {
		T  Timestamp   `query:"ts"`
		TA []Timestamp `query:"ta"`
		SA StringArray `query:"sa"`
		ST Struct
	}{}
	err := c.Bind(&result)
	ts := Timestamp(time.Date(2016, 12, 6, 19, 9, 5, 0, time.UTC))

	assert := assert.New(t)
	if assert.NoError(err) {
		//		assert.Equal( Timestamp(reflect.TypeOf(&Timestamp{}), time.Date(2016, 12, 6, 19, 9, 5, 0, time.UTC)), result.T)
		assert.Equal(ts, result.T)
		assert.Equal(StringArray([]string{"one", "two", "three"}), result.SA)
		assert.Equal([]Timestamp{ts, ts}, result.TA)
		assert.Equal(Struct{"baz"}, result.ST)
	}
}

func TestBindUnmarshalParamPtr(t *testing.T) {
	e := New()
	req := httptest.NewRequest(http.MethodGet, "/?ts=2016-12-06T19:09:05Z", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	result := struct {
		Tptr *Timestamp `query:"ts"`
	}{}
	err := c.Bind(&result)
	if assert.NoError(t, err) {
		assert.Equal(t, Timestamp(time.Date(2016, 12, 6, 19, 9, 5, 0, time.UTC)), *result.Tptr)
	}
}

func TestBindUnsupportedMediaType(t *testing.T) {
	assert := assert.New(t)
	testBindError(assert, strings.NewReader(invalidContent), MIMEApplicationJSON, &json.SyntaxError{})
}

func TestBindbindData(t *testing.T) {
	assert := assert.New(t)
	ts := new(bindTestStruct)
	b := new(DefaultBinder)
	b.bindData(ts, values, "form")
	assertBindTestStruct(assert, ts)
}

func TestBindUnmarshalTypeError(t *testing.T) {
	body := bytes.NewBufferString(`{ "id": "text" }`)
	e := New()
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set(HeaderContentType, MIMEApplicationJSON)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	u := new(user)

	err := c.Bind(u)

	he := NewHTTPError(http.StatusBadRequest, "Incorrect data structure")

	assert.Equal(t, he, err)
}

func TestBindSetWithProperType(t *testing.T) {
	assert := assert.New(t)
	ts := new(bindTestStruct)
	typ := reflect.TypeOf(ts).Elem()
	val := reflect.ValueOf(ts).Elem()
	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			continue
		}
		if len(values[typeField.Name]) == 0 {
			continue
		}
		val := values[typeField.Name][0]
		err := setWithProperType(typeField.Type.Kind(), val, structField)
		assert.NoError(err)
	}
	assertBindTestStruct(assert, ts)

	type foo struct {
		Bar bytes.Buffer
	}
	v := &foo{}
	typ = reflect.TypeOf(v).Elem()
	val = reflect.ValueOf(v).Elem()
	assert.Error(setWithProperType(typ.Field(0).Type.Kind(), "5", val.Field(0)))
}

func TestBindSetFields(t *testing.T) {
	assert := assert.New(t)

	ts := new(bindTestStruct)
	val := reflect.ValueOf(ts).Elem()
	// Int
	if assert.NoError(setIntField("5", 0, val.FieldByName("I"))) {
		assert.Equal(5, ts.I)
	}
	if assert.NoError(setIntField("", 0, val.FieldByName("I"))) {
		assert.Equal(0, ts.I)
	}

	// Uint
	if assert.NoError(setUintField("10", 0, val.FieldByName("UI"))) {
		assert.Equal(uint(10), ts.UI)
	}
	if assert.NoError(setUintField("", 0, val.FieldByName("UI"))) {
		assert.Equal(uint(0), ts.UI)
	}

	// Float
	if assert.NoError(setFloatField("15.5", 0, val.FieldByName("F32"))) {
		assert.Equal(float32(15.5), ts.F32)
	}
	if assert.NoError(setFloatField("", 0, val.FieldByName("F32"))) {
		assert.Equal(float32(0.0), ts.F32)
	}

	// Bool
	if assert.NoError(setBoolField("true", val.FieldByName("B"))) {
		assert.Equal(true, ts.B)
	}
	if assert.NoError(setBoolField("", val.FieldByName("B"))) {
		assert.Equal(false, ts.B)
	}

	ok, err := unmarshalFieldNonPtr("2016-12-06T19:09:05Z", val.FieldByName("T"))
	if assert.NoError(err) {
		assert.Equal(ok, true)
		assert.Equal(Timestamp(time.Date(2016, 12, 6, 19, 9, 5, 0, time.UTC)), ts.T)
	}
}

func assertBindTestStruct(a *assert.Assertions, ts *bindTestStruct) {
	a.Equal(0, ts.I)
	a.Equal(int8(8), ts.I8)
	a.Equal(int16(16), ts.I16)
	a.Equal(int32(32), ts.I32)
	a.Equal(int64(64), ts.I64)
	a.Equal(uint(0), ts.UI)
	a.Equal(uint8(8), ts.UI8)
	a.Equal(uint16(16), ts.UI16)
	a.Equal(uint32(32), ts.UI32)
	a.Equal(uint64(64), ts.UI64)
	a.Equal(true, ts.B)
	a.Equal(float32(32.5), ts.F32)
	a.Equal(float64(64.5), ts.F64)
	a.Equal("test", ts.S)
	a.Equal("", ts.GetCantSet())
}

func testBindOkay(assert *assert.Assertions, r io.Reader, ctype string) {
	e := New()
	req := httptest.NewRequest(http.MethodPost, "/", r)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	req.Header.Set(HeaderContentType, ctype)
	u := new(user)
	err := c.Bind(u)
	if assert.NoError(err) {
		assert.Equal(1, u.ID)
		assert.Equal("Jon Snow", u.Name)
	}
}

func testBindError(assert *assert.Assertions, r io.Reader, ctype string, expectedInternal error) {
	e := New()
	req := httptest.NewRequest(http.MethodPost, "/", r)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	req.Header.Set(HeaderContentType, ctype)
	u := new(user)
	err := c.Bind(u)

	switch {
	case strings.HasPrefix(ctype, MIMEApplicationJSON):
		if assert.IsType(new(HTTPError), err) {
			assert.Equal(http.StatusBadRequest, err.(*HTTPError).Code)
		}
	default:
		if assert.IsType(new(HTTPError), err) {
			assert.Equal(ErrUnsupportedMediaType, err)
			assert.IsType(expectedInternal, err.(*HTTPError).Internal)
		}
	}
}
