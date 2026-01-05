package binary

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// FixtureBasic covers all primitive types, standard slices, and basic logic.
type FixtureBasic struct {
	Name      string   // UTF-8, empty, large strings
	Timestamp int64    // UnixNano (Time/Audit fields)
	Payload   []byte   // Binary data
	Tags      []uint32 // Slice of primitives
	Count     int16    // Small integers
	Active    bool     // Booleans
	Score     float64  // Floating point
}

// FixtureComplex covers nesting, pointers, and composition patterns.
type FixtureComplex struct {
	ID        uint64
	Primary   FixtureBasic   // Embedded/Nested struct
	Secondary *FixtureBasic  // Pointer to struct (nil/non-nil)
	List      []FixtureBasic // Slice of structs
	Matrix    [3]int         // Fixed array
}

func TestFixtureBasic_Cases(t *testing.T) {
	runTest := func(t *testing.T, original *FixtureBasic) {
		t.Helper()

		var encoded []byte
		err := Encode(original, &encoded)
		if err != nil {
			t.Fatalf("Encode failed for original %#v: %v", original, err)
		}

		decoded := &FixtureBasic{}
		err = Decode(encoded, decoded)
		if err != nil {
			t.Fatalf("Decode failed: %v", err)
		}

		expected := *original
		if len(original.Payload) == 0 {
			expected.Payload = nil
		}
		if len(original.Tags) == 0 {
			expected.Tags = nil
		}

		if !reflect.DeepEqual(&expected, decoded) {
			t.Errorf("Decoded struct does not match expected.\nExpected: %#v\nDecoded:  %#v", &expected, decoded)
		}
	}

	// TC-001: String Handling
	t.Run("StringHandling", func(t *testing.T) {
		runTest(t, &FixtureBasic{Name: ""})
		runTest(t, &FixtureBasic{Name: "Hello World"})
		runTest(t, &FixtureBasic{Name: "ñandú 🚀"})
		runTest(t, &FixtureBasic{Name: strings.Repeat("a", 1025)})
	})

	// TC-002: Timestamp/Int64 Precision
	t.Run("TimestampPrecision", func(t *testing.T) {
		runTest(t, &FixtureBasic{Timestamp: time.Now().UnixNano()})
		runTest(t, &FixtureBasic{Timestamp: 0})
		runTest(t, &FixtureBasic{Timestamp: -1})
		runTest(t, &FixtureBasic{Timestamp: 9223372036854775807}) // math.MaxInt64
	})

	// TC-003: Binary Data
	t.Run("BinaryData", func(t *testing.T) {
		runTest(t, &FixtureBasic{Payload: nil})
		runTest(t, &FixtureBasic{Payload: []byte{}})
		runTest(t, &FixtureBasic{Payload: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}})
	})

	// TC-004: Slice of Primitives
	t.Run("SliceOfPrimitives", func(t *testing.T) {
		runTest(t, &FixtureBasic{Tags: nil})
		runTest(t, &FixtureBasic{Tags: []uint32{}})
		runTest(t, &FixtureBasic{Tags: []uint32{1, 2, 3, 4, 5}})
	})
}

func TestFixtureComplex_Cases(t *testing.T) {
	runTest := func(t *testing.T, original *FixtureComplex) {
		t.Helper()

		var encoded []byte
		err := Encode(original, &encoded)
		if err != nil {
			t.Fatalf("Encode failed for original %#v: %v", original, err)
		}

		decoded := &FixtureComplex{}
		err = Decode(encoded, decoded)
		if err != nil {
			t.Fatalf("Decode failed: %v", err)
		}

		// Create a deep copy of original to normalize for comparison.
		// This prevents side effects on the original test data.
		expected := *original
		if original.Secondary != nil {
			secondaryCopy := *original.Secondary
			expected.Secondary = &secondaryCopy
		}

		// Normalize slices in Primary and the copied Secondary.
		if len(expected.Primary.Payload) == 0 {
			expected.Primary.Payload = nil
		}
		if len(expected.Primary.Tags) == 0 {
			expected.Primary.Tags = nil
		}
		if expected.Secondary != nil {
			if len(expected.Secondary.Payload) == 0 {
				expected.Secondary.Payload = nil
			}
			if len(expected.Secondary.Tags) == 0 {
				expected.Secondary.Tags = nil
			}
		}

		// Normalize the List itself and its nested contents.
		if len(original.List) == 0 {
			expected.List = nil
		} else {
			expected.List = make([]FixtureBasic, len(original.List))
			for i, item := range original.List {
				newItem := item
				if len(item.Payload) == 0 {
					newItem.Payload = nil
				}
				if len(item.Tags) == 0 {
					newItem.Tags = nil
				}
				expected.List[i] = newItem
			}
		}

		if !reflect.DeepEqual(&expected, decoded) {
			t.Errorf("Decoded struct does not match expected.\nExpected: %#v\nDecoded:  %#v", &expected, decoded)
		}
	}

	// TC-005: Nested Structs
	t.Run("NestedStructs", func(t *testing.T) {
		runTest(t, &FixtureComplex{
			Primary: FixtureBasic{Name: "Nested", Count: 42},
		})
	})

	// TC-006: Pointer Handling (Nil)
	t.Run("PointerHandling_Nil", func(t *testing.T) {
		runTest(t, &FixtureComplex{Secondary: nil})
	})

	// TC-007: Pointer Handling (Populated)
	t.Run("PointerHandling_Populated", func(t *testing.T) {
		runTest(t, &FixtureComplex{
			Secondary: &FixtureBasic{Name: "Secondary", Active: true},
		})
	})

	// TC-008: Slice of Structs
	t.Run("SliceOfStructs", func(t *testing.T) {
		runTest(t, &FixtureComplex{List: nil})
		runTest(t, &FixtureComplex{List: []FixtureBasic{}})
		runTest(t, &FixtureComplex{
			List: []FixtureBasic{
				{Name: "Item 1", Score: 1.1},
				{Name: "Item 2", Score: 2.2},
			},
		})
	})

	// TC-009: Fixed Arrays
	t.Run("FixedArrays", func(t *testing.T) {
		runTest(t, &FixtureComplex{Matrix: [3]int{0, 0, 0}})
		runTest(t, &FixtureComplex{Matrix: [3]int{1, 2, 3}})
	})

	// TC-010: Zero Values
	t.Run("ZeroValues", func(t *testing.T) {
		runTest(t, &FixtureComplex{})
	})

	// TC-011: Large Collections
	t.Run("LargeCollections", func(t *testing.T) {
		largeList := make([]FixtureBasic, 10000)
		for i := 0; i < 10000; i++ {
			largeList[i] = FixtureBasic{
				Name:  "Item",
				Count: int16(i),
			}
		}
		runTest(t, &FixtureComplex{List: largeList})
	})

}

type testCustom string

// GetBinarycodec retrieves a custom binary codec.
func (s *testCustom) GetBinarycodec() codec {
	return new(stringcodec)
}
