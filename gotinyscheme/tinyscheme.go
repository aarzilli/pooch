package gotinyscheme

/*
#include <scheme.h>
#include <stdlib.h>
*/
import "C"

type Scheme struct {
	scheme *C.scheme
}

func NewScheme() *Scheme {
	s := C.scheme_init_new();
	return &Scheme{s}
}
