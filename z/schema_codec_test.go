package z

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestCodecStringNumberRoundTrip(t *testing.T) {
	c := Codec(String(), Number(), CodecTx{
		Decode: func(v any, _ *RefinementCtx) (any, error) {
			f, err := strconv.ParseFloat(v.(string), 64)
			return f, err
		},
		Encode: func(v any, _ *RefinementCtx) (any, error) {
			f, _ := ToFloat(v)
			return strconv.FormatFloat(f, 'f', -1, 64), nil
		},
	})
	got, err := Decode(c, "42.5")
	if err != nil || got != 42.5 {
		t.Fatalf("decode: %v %v", got, err)
	}
	back, err := Encode(c, 42.5)
	if err != nil || back != "42.5" {
		t.Fatalf("encode: %v %v", back, err)
	}
}

func TestCodecISODate(t *testing.T) {
	c := Codec(String().ISODateTime(), Time(), CodecTx{
		Decode: func(v any, _ *RefinementCtx) (any, error) {
			return time.Parse(time.RFC3339Nano, v.(string))
		},
		Encode: func(v any, _ *RefinementCtx) (any, error) {
			return v.(time.Time).UTC().Format("2006-01-02T15:04:05.000Z"), nil
		},
	})
	const iso = "2024-01-15T10:30:00.000Z"
	decoded, err := Decode(c, iso)
	if err != nil {
		t.Fatal(err)
	}
	tm, ok := decoded.(time.Time)
	if !ok {
		t.Fatalf("got %T", decoded)
	}
	want := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !tm.UTC().Equal(want) {
		t.Fatalf("time = %v", tm)
	}
	encoded, err := Encode(c, tm)
	if err != nil || encoded != iso {
		t.Fatalf("encode: %v %v", encoded, err)
	}
}

func TestInvertCodec(t *testing.T) {
	c := Codec(String(), Number(), CodecTx{
		Decode: func(v any, _ *RefinementCtx) (any, error) {
			return strconv.ParseFloat(v.(string), 64)
		},
		Encode: func(v any, _ *RefinementCtx) (any, error) {
			f, _ := ToFloat(v)
			return strconv.FormatFloat(f, 'f', -1, 64), nil
		},
	})
	inv := InvertCodec(c)
	got, err := Decode(inv, 7.0)
	if err != nil || got != "7" {
		t.Fatalf("invert decode: %v %v", got, err)
	}
}

func TestJSONStringCodec(t *testing.T) {
	c := JSONStringCodec(Object(Shape{"a": Number()}))
	got, err := Decode(c, `{"a":1}`)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := got.(map[string]any)
	if !ok || m["a"] != 1.0 {
		t.Fatalf("%#v", got)
	}
	enc, err := Encode(c, map[string]any{"a": 1.0})
	if err != nil {
		t.Fatal(err)
	}
	if enc != `{"a":1}` {
		t.Fatalf("enc = %q", enc)
	}
}

func TestSafeDecodeEncode(t *testing.T) {
	c := Codec(String(), Number(), CodecTx{
		Decode: func(v any, _ *RefinementCtx) (any, error) {
			return strconv.ParseFloat(fmt.Sprint(v), 64)
		},
		Encode: func(v any, _ *RefinementCtx) (any, error) {
			f, _ := ToFloat(v)
			return strconv.FormatFloat(f, 'f', -1, 64), nil
		},
	})
	bad := SafeDecode(c, 123) // not a string → input schema fails
	if bad.Success {
		t.Fatal("expected failure")
	}
	good := SafeEncode(c, 3.0)
	if !good.Success || good.Data != "3" {
		t.Fatalf("%+v", good)
	}
}
