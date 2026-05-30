package cache

import (
	"fmt"
	"reflect"
)

func UnsafeAssign(dst interface{}, src interface{}) error {
	refDst := reflect.ValueOf(dst)
	refSrc := reflect.ValueOf(src)
	if refDst.Kind() == reflect.Ptr {
		refDst = refDst.Elem()
	}
	if refSrc.Kind() == reflect.Ptr {
		refSrc = refSrc.Elem()
	}
	if !refDst.CanSet() {
		return fmt.Errorf("%s cannot be set", refDst.Type())
	}
	if !refSrc.Type().AssignableTo(refDst.Type()) {
		return fmt.Errorf("cannot assign %s to %s", refSrc.Type(), refDst.Type())
	}
	refDst.Set(refSrc)
	return nil
}
