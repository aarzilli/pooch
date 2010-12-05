#include <scheme.h>
#include <stdlib.h>
#include <stdio.h>

pointer gocall(scheme *sc, pointer args) {
 printf("prova prova prova!\n");
 return 0;
}

void custom_init(scheme *s) {
 foreign_func f = gocall;
 scheme_set_output_port_file(s, stdout);
}

int main(void) {
  scheme *s = scheme_init_new();
  custom_init(s);
}
