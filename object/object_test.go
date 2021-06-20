package object

import "testing"

func TestStringHashKey(t *testing.T) {
	key1a := &String{Value: "Hello World"}
	key1b := &String{Value: "Hello World"}
	key2a := &String{Value: "My name is johnny"}
	key2b := &String{Value: "My name is johnny"}

	if key1a.HashKey() != key1b.HashKey() {
		t.Errorf("strings with same content have different hash keys")
	}
	if key2a.HashKey() != key2b.HashKey() {
		t.Errorf("strings with same content have different hash keys")
	}
	if key1a.HashKey() == key2a.HashKey() {
		t.Errorf("strings with different content have same hash keys")
	}
}

func TestIntHashKey(t *testing.T) {
	key1a := &Integer{Value: 69}
	key1b := &Integer{Value: 69}
	key2a := &Integer{Value: 42}
	key2b := &Integer{Value: 42}

	if key1a.HashKey() != key1b.HashKey() {
		t.Errorf("integers with same value have different hash keys")
	}
	if key2a.HashKey() != key2b.HashKey() {
		t.Errorf("integers with same value have different hash keys")
	}
	if key1a.HashKey() == key2a.HashKey() {
		t.Errorf("integers with different value have same hash keys")
	}
}
