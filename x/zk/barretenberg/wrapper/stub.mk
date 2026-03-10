# Makefile.stub - Build the stub library for development/testing
#
# This builds a minimal stub implementation that allows the Go code
# to compile and run basic tests without the full Barretenberg library.
#
# Usage:
#   make -f Makefile.stub           # Build for current platform
#   make -f Makefile.stub clean     # Clean build artifacts

CXX := clang++
CXXFLAGS := -std=c++17 -fPIC -O2 -Wall -Wextra

# Detect platform
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Darwin)
    ifeq ($(UNAME_M),arm64)
        PLATFORM := darwin_arm64
        LDFLAGS := -lc++
    else
        PLATFORM := darwin_amd64
        LDFLAGS := -lc++
    endif
else ifeq ($(UNAME_S),Linux)
    ifeq ($(UNAME_M),aarch64)
        PLATFORM := linux_arm64
    else
        PLATFORM := linux_amd64
    endif
    LDFLAGS := -lstdc++
endif

LIB_DIR := ../lib/$(PLATFORM)
TARGET := $(LIB_DIR)/libbarretenberg.a

SRCS := barretenberg_stub.cpp
OBJS := $(SRCS:.cpp=.o)

.PHONY: all clean

all: $(TARGET)

$(TARGET): $(OBJS) | $(LIB_DIR)
	ar rcs $@ $^
	@echo "Built stub library: $@"

$(LIB_DIR):
	mkdir -p $(LIB_DIR)

%.o: %.cpp
	$(CXX) $(CXXFLAGS) -I../include -c $< -o $@

clean:
	rm -f $(OBJS)
	rm -f $(TARGET)
