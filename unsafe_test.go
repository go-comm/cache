package cache

import (
	"testing"
)

func TestUnsafeAssignSameType(t *testing.T) {
	var dst int
	src := 42
	if err := UnsafeAssign(&dst, src); err != nil {
		t.Fatal(err)
	}
	if dst != 42 {
		t.Fatalf("expected 42, got %d", dst)
	}
}

func TestUnsafeAssignPtrSrc(t *testing.T) {
	var dst int
	src := 100
	if err := UnsafeAssign(&dst, &src); err != nil {
		t.Fatal(err)
	}
	if dst != 100 {
		t.Fatalf("expected 100, got %d", dst)
	}
}

func TestUnsafeAssignString(t *testing.T) {
	var dst string
	src := "hello"
	if err := UnsafeAssign(&dst, src); err != nil {
		t.Fatal(err)
	}
	if dst != "hello" {
		t.Fatalf("expected hello, got %s", dst)
	}
}

func TestUnsafeAssignStruct(t *testing.T) {
	type S struct {
		Name string
		Age  int
	}
	var dst S
	src := S{Name: "test", Age: 25}
	if err := UnsafeAssign(&dst, src); err != nil {
		t.Fatal(err)
	}
	if dst.Name != "test" || dst.Age != 25 {
		t.Fatalf("expected {test 25}, got %+v", dst)
	}
}

func TestUnsafeAssignPtrDst(t *testing.T) {
	type S struct {
		Name string
	}
	var dst S
	src := &S{Name: "p"}
	if err := UnsafeAssign(dst, src); err == nil {
		t.Fatal("expected error for non-settable dst")
	}
}

func TestUnsafeAssignIncompatible(t *testing.T) {
	var dst string
	src := 42
	err := UnsafeAssign(&dst, src)
	if err == nil {
		t.Fatal("expected error for incompatible types")
	}
}

func TestUnsafeAssignFloat64(t *testing.T) {
	var dst float64
	src := 3.14
	if err := UnsafeAssign(&dst, src); err != nil {
		t.Fatal(err)
	}
	if dst != 3.14 {
		t.Fatalf("expected 3.14, got %f", dst)
	}
}

func TestUnsafeAssignBool(t *testing.T) {
	var dst bool
	src := true
	if err := UnsafeAssign(&dst, src); err != nil {
		t.Fatal(err)
	}
	if !dst {
		t.Fatal("expected true")
	}
}
