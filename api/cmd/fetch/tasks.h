#ifndef ONESEISMIC_CGO_TASK_H
#define ONESEISMIC_CGO_TASK_H

#include <stdbool.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif //__cplusplus

/*
 * The process handle. Must be explicitly free'd from go by calling cleanup().
 * It is implemented under the assumption that there is a fresh, newly-created
 * proc for every request, although this can in the future be pooled and
 * instances re-used.
 *
 * The process has a "staging area" (like in git) where downloaded fragments
 * are added.
 */
struct proc;

/*
 * Create a new process of kind. The kind is the function key in the task
 * message (see: message.go). Returns a nullptr if the kind is wrong or if it
 * can not be allocated. Newly-created processes must be init()'d before use.
 */
struct proc* newproc(const char* kind);
/*
 * Clean up (delete) the proc. If proc is a nullptr, this is a no-op.
 */
void cleanup(struct proc*);

/*
 * Get the last-set error message. This should be called immediately after the
 * error occured, like errno. Error messages are not cleared, so the presence
 * of an error message does not mean the last called failed.
 */
const char* errmsg(struct proc*);

/*
 * Init a process. msg of len is a task as bytes (see: messages.hpp)
 *
 * This function only up the process, it does not spawn off a thread.
 */
bool init(struct proc*, const void* msg, int len);

/*
 * Add a downloaded fragment to the staging area. This function is not thread
 * safe, and must not be called simultaneously from multiple goroutines.
 *
 * The index argument can be considered a key, and should correspond to the
 * index (in fragments()) of the fragment ID that triggered the download.
 */
bool add(struct proc*, int index, const void* chunk, int len);

/*
 * Get the list of fragments from proc. This function is not thread safe. The
 * list of fragments is returned as a single string, with list elements
 * separated by ';'. The char array is owned by C++ and *must not* be free'd.
 *
 * Returning the list-of-fragments as a single string means only a single round
 * trip go <-> C++, at the cost of parsing a string in go.
 */
const char* fragments(struct proc*);

/*
 * The result of the pack(), which packs all add()ed objects into a response
 * that can be written to redis. The actual redis write is expected to be
 * handled by go, C++ only arranges the bytes.
 *
 * The memory is owned by C++ and *must not* be free'd by go. This function is
 * not thread safe.
 *
 * If pack fails (packed.err == true) then accessing the other fields is
 * undefined behaviour.
 */
struct packed {
    bool err;
    int size;
    const void* body;
};
struct packed pack(struct proc*);


#ifdef __cplusplus
}
#endif //__cplusplus
#endif // ONESEISMIC_CGO_TASK_H
