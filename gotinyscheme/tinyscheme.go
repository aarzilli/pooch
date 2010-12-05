package gotinyscheme

/*
#include <scheme.h>
#include <stdlib.h>
#include <stdio.h>

pointer gocall(scheme *sc, pointer args) {
 GoCall(sc, args);
 return scheme_nil(sc);
}

void custom_init(scheme *s) {
 scheme_set_output_port_file(s, stdout);
 scheme_global_define(s, mk_symbol(s, "gocall"), mk_foreign_func(s, gocall));
}

*/
import "C"

import (
	"unsafe"
	"fmt"
	"strconv"
)

//export GoCall
func GoCall(scheme *C.scheme, args C.pointer) {
	fmt.Printf("ciao\n")
}

type Scheme struct {
	scheme *C.scheme
}

type Value struct {
	scheme *Scheme
	pointer C.pointer
}

func NewScheme() *Scheme {
	s := C.scheme_init_new();
	C.custom_init(s)
	return &Scheme{s}
}

func (s *Scheme) evalEx(code string) C.pointer {
	ccode := C.CString(code)
	defer C.free(unsafe.Pointer(ccode))
	//TODO: intercettare output port, salvarlo come errore
	C.scheme_load_string(s.scheme, ccode)
	fmt.Printf("Error code: %d\n", int(C.scheme_retcode(s.scheme)))
	return C.scheme_value(s.scheme)
}

func (s *Scheme) pointerToInterface(p C.pointer) interface{} {
	if C.scheme_isinteger(s.scheme, p) != 0 {
		return int64(C.scheme_ivalue(s.scheme, p))
	} else if C.scheme_isreal(s.scheme, p) != 0 {
		return float64(C.scheme_rvalue(s.scheme, p))
	} else if C.scheme_ischar(s.scheme, p) != 0 {
		return int64(C.scheme_charvalue(s.scheme, p))
	} else if C.scheme_isstring(s.scheme, p) != 0 {
		return C.GoString(C.scheme_string_value(s.scheme, p))
	}

	return nil
}

func (s *Scheme) pointerToValue(p C.pointer) *Value {
	return &Value{s, p}
}

func (s *Scheme) EvalEx(code string) interface{} {
	return s.pointerToInterface(s.evalEx(code))
}

func (s *Scheme) Eval(code string) *Value {
	return s.pointerToValue(s.evalEx(code))
}

func (s *Scheme) Close() {
	C.free(unsafe.Pointer(s.scheme))
}

func (v *Value) String() string {
	return fmt.Sprintf("%v", v.scheme.pointerToInterface(v.pointer))
}

func (v *Value) Int() int64 {
	if C.scheme_isinteger(v.scheme.scheme, v.pointer) != 0 {
		return int64(C.scheme_ivalue(v.scheme.scheme, v.pointer))
	} else if C.scheme_isreal(v.scheme.scheme, v.pointer) != 0 {
		return int64(float64(C.scheme_rvalue(v.scheme.scheme, v.pointer)))
	} else if C.scheme_ischar(v.scheme.scheme, v.pointer) != 0 {
		return int64(C.scheme_charvalue(v.scheme.scheme, v.pointer))
	} else if C.scheme_isstring(v.scheme.scheme, v.pointer) != 0 {
		s := C.GoString(C.scheme_string_value(v.scheme.scheme, v.pointer))
		n, _ := strconv.Atoi64(s)
		return n
	}

	return 0
}

func (v *Value) Float() float64 {
	if C.scheme_isinteger(v.scheme.scheme, v.pointer) != 0 {
		return float64(int64(C.scheme_ivalue(v.scheme.scheme, v.pointer)))
	} else if C.scheme_isreal(v.scheme.scheme, v.pointer) != 0 {
		return float64(C.scheme_rvalue(v.scheme.scheme, v.pointer))
	} else if C.scheme_ischar(v.scheme.scheme, v.pointer) != 0 {
		return float64(int64(C.scheme_charvalue(v.scheme.scheme, v.pointer)))
	} else if C.scheme_isstring(v.scheme.scheme, v.pointer) != 0 {
		s := C.GoString(C.scheme_string_value(v.scheme.scheme, v.pointer))
		n, _ := strconv.Atof64(s)
		return n
	}

	return 0
}

