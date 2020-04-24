package cache

import "testing"

func Test_Convert(t *testing.T) {

	var a int = 1000
	var b *int
	UnsafeConvert(&b, &a)
	t.Log(a, *b, a == *b)

	var foo_from [2]string
	var foo_to *[2]string
	foo_from[0] = "abc"
	foo_from[1] = "123"
	UnsafeConvert(&foo_to, &foo_from)
	t.Log(foo_from, foo_to)

}
