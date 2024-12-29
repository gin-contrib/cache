package utils

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"
)

func TestSerialize(t *testing.T) {
	tests := []struct {
		input    any
		expected []byte
	}{
		{input: []byte("test"), expected: []byte("test")},
		{input: int(123), expected: []byte("123")},
		{input: int8(123), expected: []byte("123")},
		{input: int16(123), expected: []byte("123")},
		{input: int32(123), expected: []byte("123")},
		{input: int64(123), expected: []byte("123")},
		{input: uint(123), expected: []byte("123")},
		{input: uint8(123), expected: []byte("123")},
		{input: uint16(123), expected: []byte("123")},
		{input: uint32(123), expected: []byte("123")},
		{input: uint64(123), expected: []byte("123")},
	}

	for _, test := range tests {
		result, err := Serialize(test.input)
		if err != nil {
			t.Errorf("Serialize(%v) returned error: %v", test.input, err)
		}
		if !bytes.Equal(result, test.expected) {
			t.Errorf("Serialize(%v) = %v; want %v", test.input, result, test.expected)
		}
	}

	// Test for gob encoding
	type TestStruct struct {
		Field1 string
		Field2 int
	}
	input := TestStruct{"test", 123}
	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)
	err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("Failed to encode input: %v", err)
	}
	expected := b.Bytes()

	result, err := Serialize(input)
	if err != nil {
		t.Errorf("Serialize(%v) returned error: %v", input, err)
	}
	if !bytes.Equal(result, expected) {
		t.Errorf("Serialize(%v) = %v; want %v", input, result, expected)
	}
}

func TestDeserialize(t *testing.T) {
	tests := []struct {
		input    []byte
		ptr      any
		expected any
	}{
		{input: []byte("test"), ptr: new([]byte), expected: []byte("test")},
		{input: []byte("123"), ptr: new(int), expected: 123},
		{input: []byte("123"), ptr: new(int8), expected: int8(123)},
		{input: []byte("123"), ptr: new(int16), expected: int16(123)},
		{input: []byte("123"), ptr: new(int32), expected: int32(123)},
		{input: []byte("123"), ptr: new(int64), expected: int64(123)},
		{input: []byte("123"), ptr: new(uint), expected: uint(123)},
		{input: []byte("123"), ptr: new(uint8), expected: uint8(123)},
		{input: []byte("123"), ptr: new(uint16), expected: uint16(123)},
		{input: []byte("123"), ptr: new(uint32), expected: uint32(123)},
		{input: []byte("123"), ptr: new(uint64), expected: uint64(123)},
	}

	for _, test := range tests {
		err := Deserialize(test.input, test.ptr)
		if err != nil {
			t.Errorf("Deserialize(%v, %v) returned error: %v", test.input, test.ptr, err)
		}
		if !reflect.DeepEqual(reflect.ValueOf(test.ptr).Elem().Interface(), test.expected) {
			t.Errorf("Deserialize(%v, %v) = %v; want %v", test.input, test.ptr, reflect.ValueOf(test.ptr).Elem().Interface(), test.expected)
		}
	}

	// Test for gob decoding
	type TestStruct struct {
		Field1 string
		Field2 int
	}
	inputStruct := TestStruct{"test", 123}
	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)
	err := encoder.Encode(inputStruct)
	if err != nil {
		t.Fatalf("Failed to encode input: %v", err)
	}
	input := b.Bytes()

	var outputStruct TestStruct
	err = Deserialize(input, &outputStruct)
	if err != nil {
		t.Errorf("Deserialize(%v, %v) returned error: %v", input, &outputStruct, err)
	}
	if !reflect.DeepEqual(outputStruct, inputStruct) {
		t.Errorf("Deserialize(%v, %v) = %v; want %v", input, &outputStruct, outputStruct, inputStruct)
	}
}
