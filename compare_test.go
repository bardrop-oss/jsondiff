package jsondiff

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func Test_marshalUnmarshal_invalid_JSON(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		e := errors.New("marshal")
		_, _, err := marshalUnmarshal(nil, options{
			// Override marshal func to simulate error.
			marshal:   func(any) ([]byte, error) { return nil, e },
			unmarshal: json.Unmarshal,
		})
		if !errors.Is(err, e) {
			t.Errorf("expected non-nil error")
		}
	})
	t.Run("unmarshal", func(t *testing.T) {
		e := errors.New("unmarshal")
		_, _, err := marshalUnmarshal(nil, options{
			// Override unmarshal func to simulate error.
			marshal:   json.Marshal,
			unmarshal: func([]byte, any) error { return e },
		})
		if !errors.Is(err, e) {
			t.Errorf("expected non-nil error")
		}
	})
}

func TestCompareWithoutMarshal_error(t *testing.T) {
	type custom struct {
		foo string
		bar int
	}
	_, err := CompareWithoutMarshal(custom{"a", 1}, custom{"b", 2})
	if err == nil {
		t.Errorf("expected non-nil error")
	}
	t.Log(err)
}

type skip struct{}

var skipBytes = []byte("skip")

func Test_compare_marshaling_error(t *testing.T) {
	e := errors.New("")

	d := Differ{opts: options{
		marshal: func(a any) ([]byte, error) {
			if _, ok := a.(skip); ok {
				return skipBytes, nil
			}
			return nil, e
		},
		unmarshal: func(b []byte, _ any) error {
			if bytes.Equal(b, skipBytes) {
				return nil
			}
			return e
		},
	}}
	if _, err := compare(&d, nil, nil); !errors.Is(err, e) {
		t.Errorf("expected non-nil error")
	}
	if _, err := compare(&d, skip{}, nil); !errors.Is(err, e) {
		t.Errorf("expected non-nil error")
	}
}

func Test_compareJSON_marshaling_error(t *testing.T) {
	e := errors.New("")

	unmarshalFn := func(b []byte, _ any) error {
		if bytes.Equal(b, skipBytes) {
			return nil
		}
		return e
	}
	if _, err := compareJSON(nil, nil, nil, unmarshalFn); !errors.Is(err, e) {
		t.Errorf("expected non-nil error")
	}
	if _, err := compareJSON(nil, []byte("skip"), nil, unmarshalFn); !errors.Is(err, e) {
		t.Errorf("expected non-nil error")
	}
}

// Hashing with the Equivalent option sorts arrays to make the digest
// order-independent. It must not reorder the documents themselves:
// the differ walks them by index afterwards. The hash seed is random,
// so run the comparison many times.
func TestCompareJSON_equivalentDoesNotReorderDocuments(t *testing.T) {
	src := []byte(`[["a","b","c"],0]`)
	tgt := []byte(`[["a","b","d"],1]`)

	var want interface{}
	if err := json.Unmarshal(tgt, &want); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		patch, err := CompareJSON(src, tgt, Equivalent())
		if err != nil {
			t.Fatal(err)
		}
		b, err := patch.apply(src, false)
		if err != nil {
			t.Fatalf("apply %s: %v", patch, err)
		}
		var got interface{}
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("patch %s gives %s, want %s", patch, b, tgt)
		}
	}
}
