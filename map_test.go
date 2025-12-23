package binary

import (
	"bytes"
	"testing"
)

func TestMapCodec(t *testing.T) {
	t.Run("MapStringInt", func(t *testing.T) {
		m := map[string]int{
			"one": 1,
			"two": 2,
		}

		var buf bytes.Buffer
		err := Encode(m, &buf)
		if err != nil {
			t.Fatalf("Encode failed: %v", err)
		}

		var m2 map[string]int
		// We use a reader but Decode also supports bytes
		err = Decode(buf.Bytes(), &m2)
		if err != nil {
			t.Fatalf("Decode failed: %v", err)
		}

		if len(m2) != len(m) {
			t.Errorf("Expected length %d, got %d", len(m), len(m2))
		}

		for k, v := range m {
			if v2, ok := m2[k]; !ok || v2 != v {
				t.Errorf("Expected m2[%s] = %d, got %d", k, v, v2)
			}
		}
	})

	t.Run("EmptyMap", func(t *testing.T) {
		m := map[string]string{}
		var buf bytes.Buffer
		if err := Encode(m, &buf); err != nil {
			t.Fatal(err)
		}

		var m2 map[string]string
		if err := Decode(buf.Bytes(), &m2); err != nil {
			t.Fatal(err)
		}

		if len(m2) != 0 {
			t.Errorf("Expected empty map, got length %d", len(m2))
		}
	})

	t.Run("MapWithPointers", func(t *testing.T) {
		val := 42
		m := map[int]*int{
			1: &val,
			2: nil,
		}

		var buf bytes.Buffer
		if err := Encode(m, &buf); err != nil {
			t.Fatal(err)
		}

		var m2 map[int]*int
		if err := Decode(buf.Bytes(), &m2); err != nil {
			t.Fatal(err)
		}

		if len(m2) != 2 {
			t.Fatalf("Expected length 2, got %d", len(m2))
		}

		if m2[1] == nil || *m2[1] != 42 {
			t.Errorf("Expected *m2[1] = 42, got %v", m2[1])
		}
		if m2[2] != nil {
			t.Errorf("Expected m2[2] = nil, got %v", m2[2])
		}
	})

	t.Run("SliceOfMaps", func(t *testing.T) {
		m1 := map[string]int{"a": 1, "b": 2}
		m2 := map[string]int{"c": 3}
		s := []map[string]int{m1, m2}

		var buf bytes.Buffer
		if err := Encode(s, &buf); err != nil {
			t.Fatalf("Encode failed: %v", err)
		}

		var s2 []map[string]int
		if err := Decode(buf.Bytes(), &s2); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}

		if len(s2) != 2 {
			t.Fatalf("Expected length 2, got %d", len(s2))
		}

		if s2[0]["a"] != 1 || s2[0]["b"] != 2 || s2[1]["c"] != 3 {
			t.Errorf("Data mismatch in slice of maps: %v", s2)
		}
	})
}
