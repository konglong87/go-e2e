#ifndef GO_E2E_COMPUTER_POINTER_DARWIN_H
#define GO_E2E_COMPUTER_POINTER_DARWIN_H

#ifdef __cplusplus
extern "C" {
#endif

typedef struct goe2e_computer_pointer goe2e_computer_pointer;

goe2e_computer_pointer *goe2e_computer_pointer_create(void);
void goe2e_computer_pointer_show(goe2e_computer_pointer *pointer, const char *json_utf8);
void goe2e_computer_pointer_hide(goe2e_computer_pointer *pointer);
void goe2e_computer_pointer_close(goe2e_computer_pointer *pointer);

#ifdef __cplusplus
}
#endif

#endif
