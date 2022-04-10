#include <stdio.h>
#include "c11threads.h"

int tfunc(void *arg);
void run_timed_test();
int hold_mutex_three_seconds(void* arg);

mtx_t mtx;
mtx_t startup_mtx;

int main(void)
{
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

	printf("start timed mutex test\n");
	run_timed_test();
	printf("stop timed mutex test\n");

	return 0;
}

int tfunc(void *arg)
{
	int num = (long)arg;
	struct timespec dur;

	printf("hello from thread %d\n", num);

	dur.tv_sec = 4;
	dur.tv_nsec = 0;
	thrd_sleep(&dur, 0);

	printf("thread %d done\n", num);
	return 0;
}

int hold_mutex_three_seconds(void* arg) {
	struct timespec dur;
	mtx_lock(&mtx);
	mtx_unlock(&startup_mtx);
	dur.tv_sec = 3;
	dur.tv_nsec = 0;
	thrd_sleep(&dur, 0);
	mtx_unlock(&mtx);

	return 0;
}

void run_timed_test()
{
	thrd_t thread;
	struct timespec ts;
	struct timespec dur;

	mtx_init(&mtx, mtx_timed);
	mtx_init(&mtx, mtx_plain);

	mtx_lock(&startup_mtx);
	thrd_create(&thread, hold_mutex_three_seconds, &mtx);

	mtx_lock(&startup_mtx);
	timespec_get(&ts, 0);
	ts.tv_sec = ts.tv_sec + 2;
	ts.tv_nsec = 0;
	if (mtx_timedlock(&mtx,&ts)==thrd_timedout) {
		printf("thread has locked mutex & we timed out waiting for it\n");
	}

	dur.tv_sec = 4;
	dur.tv_nsec = 0;
	thrd_sleep(&dur, 0);

	if (mtx_timedlock(&mtx,&ts)==thrd_success) {
		printf("thread no longer has mutex & we grabbed it\n");
	}

	mtx_destroy(&mtx);
}