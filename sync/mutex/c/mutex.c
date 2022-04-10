#include <stdio.h>
#include "c11threads.h"

// Locks a mutex, blocking if necessary until it becomes free.
// int mtx_lock(mtx_t *mutex);

// Unlocks a mutex.
// int mtx_unlock(mtx_t *mutex);

int tfunc(void *arg);

mtx_t mtx;
mtx_t startup_mtx;
int the_counter;

int main(void) {
	int i;
	thrd_t threads[4];

	printf("start thread test\n");

	for(i=0; i<4; i++) {
		thrd_create(threads + i, tfunc, (void*)(long)i);
	}

	for(i=0; i<4; i++) {
		thrd_join(threads[i], 0);
	}

	printf("end thread test\n\n");

	return 0;
}

// Code to initialize the_mutex omitted.
int increment_the_counter() {
    int r = mtx_lock(&mtx);
    if (r != 0) return r;

    // With the mutex locked we can safely poke the counter.
    the_counter++;

    return mtx_unlock(&mtx);
}

int read_the_counter(int *value) {
    int r = mtx_lock(&mtx);
    if (r != 0) return r;

    // With the mutex locked we can safely read the counter.
    *value = the_counter;

    return mtx_unlock(&mtx);
}

int tfunc(void *arg)
{
	int num = (long)arg;
    int value = 0;
	struct timespec dur;

    mtx_init(&mtx, mtx_timed);
	mtx_init(&mtx, mtx_plain);
	
    increment_the_counter();
	printf("hello from thread %d\n", num);

    read_the_counter(&value);
    printf("the_counter: %d\n", value);

	dur.tv_sec = 1;
	dur.tv_nsec = 0;
	thrd_sleep(&dur, 0);

	printf("thread %d done\n", num);

	return 0;
}
