obj = mutex.o
bin = mutex

CFLAGS = -std=gnu99 -pedantic -Wall -g
LDFLAGS = -lpthread

$(bin): $(obj)
	$(CC) -o $@ $(obj) $(LDFLAGS) && make run

mutex.o: mutex.c c11threads.h

.PHONY: run
run: 
	./mutex && make clean

.PHONY: clean
clean:
	rm -f $(obj) $(bin)
