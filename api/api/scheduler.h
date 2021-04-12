#ifndef ONESEISMIC_CGO_SLICE_H
#define ONESEISMIC_CGO_SLICE_H

#ifdef __cplusplus
extern "C" {
#endif //__cplusplus

struct plan {
    const char* err;
    int len;
    int* sizes;
    char* tasks;
};

struct plan mkschedule(const char* doc, int len, int task_size);
void cleanup(struct plan*);

#ifdef __cplusplus
}
#endif //__cplusplus
#endif //ONESEISMIC_CGO_SLICE_H
