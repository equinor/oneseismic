#ifndef ONESEISMIC_CGO_QUERY_H
#define ONESEISMIC_CGO_QUERY_H

#ifdef __cplusplus
extern "C" {
#endif //__cplusplus

struct plan {
    /*
     * The error message. On success, this is a nullptr. Checking err for null
     * is the only correct way to determine if the function succeeded or not.
     */
    const char* err;

    /*
     * The number of task groups/chunks in this plan, including the header.
     * This denotes the length of the sizes array.
     */
    int len;

    /*
     * The plan, the already-partitioned task descriptions, suitable for
     * consumption by the workers, including the header.
     *
     * The sizes array stores the size of every chunk, and the tasks array
     * stores the marshalled chunks, contiguously. To partition into individual
     * chunks:
     *
     *     chunks = []
     *     for size in sizes:
     *         chunks.append(copy(tasks, tasks + size))
     *         tasks.advance(size)
     */
    int* sizes;
    char* tasks;
};

struct plan mkschedule(const char* doc, int len, int task_size);
void cleanup(struct plan*);

#ifdef __cplusplus
}
#endif //__cplusplus
#endif //ONESEISMIC_CGO_QUERY_H
