# stub.mk — Build libbarretenberg_stub.a for CI/testing without the real library.
#
# The stub is pure C (barretenberg_stub.c), so it compiles with plain gcc.
# No clang, no libc++, no libstdc++ required — no C++ stdlib flags at all.
#
# Usage:
#   make -f stub.mk           # Build for current platform
#   make -f stub.mk clean     # Clean build artifacts

CC := gcc
CFLAGS := -std=c11 -fPIC -O2 -Wall -Wextra

# Detect platform
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Darwin)
    ifeq ($(UNAME_M),arm64)
        PLATFORM := darwin_arm64
    else
        PLATFORM := darwin_amd64
    endif
else ifeq ($(UNAME_S),Linux)
    ifeq ($(UNAME_M),aarch64)
        PLATFORM := linux_arm64
    else
        PLATFORM := linux_amd64
    endif
endif

LIB_DIR := ../lib/$(PLATFORM)
TARGET := $(LIB_DIR)/libbarretenberg_stub.a

SRCS := barretenberg_stub.c
OBJS := $(SRCS:.c=.o)

.PHONY: all clean

all: $(TARGET)

$(TARGET): $(OBJS) | $(LIB_DIR)
	ar rcs $@ $^
	ranlib $@
	@echo "Built stub library: $@"

$(LIB_DIR):
	mkdir -p $(LIB_DIR)

%.o: %.c
	$(CC) $(CFLAGS) -I../include -c $< -o $@

clean:
	rm -f $(OBJS)
	rm -f $(TARGET)
