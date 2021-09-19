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

struct query_result {
    char* err;
    /*
     * The body is the already json-encoded response. Accessing this if err is
     * set is undefined behaviour.
     */
    char* body;
    int size;
};

void plan_delete(struct plan*);
void query_result_delete(struct query_result*);

struct session;
struct session* session_new();
const char* session_init(struct session*, const char* doc, int len);
struct plan session_plan_query(
    struct session*,
    const char* doc,
    int len,
    int task_size);

struct query_result session_query_manifest(
    struct session*,
    const char* path,
    int len);

#ifdef __cplusplus
}
#endif //__cplusplus
#endif //ONESEISMIC_CGO_QUERY_H
