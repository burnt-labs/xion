cmake_minimum_required(VERSION 3.16)

project(barretenberg_wrapper
    VERSION 0.1.0
    DESCRIPTION "C wrapper for Barretenberg UltraHonk verification"
    LANGUAGES CXX
)

# Require C++20 for Barretenberg
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# Build options
option(BUILD_SHARED_LIBS "Build shared libraries" OFF)

# Barretenberg git reference — pinned to aztec-packages monorepo stable release tag
set(BB_REF "v4.0.4" CACHE STRING "Barretenberg git reference (branch/tag/commit)")

# Detect target platform
if(NOT DEFINED TARGET_PLATFORM)
    if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
        if(CMAKE_SYSTEM_PROCESSOR MATCHES "aarch64|arm64")
            set(TARGET_PLATFORM "linux_arm64")
        else()
            set(TARGET_PLATFORM "linux_amd64")
        endif()
    elseif(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
        if(CMAKE_SYSTEM_PROCESSOR MATCHES "arm64")
            set(TARGET_PLATFORM "darwin_arm64")
        else()
            set(TARGET_PLATFORM "darwin_amd64")
        endif()
    else()
        message(FATAL_ERROR "Unsupported platform: ${CMAKE_SYSTEM_NAME}")
    endif()
endif()

message(STATUS "Building for platform: ${TARGET_PLATFORM}")

# macOS: Set SDK path for proper header resolution
if(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
    execute_process(
        COMMAND xcrun --show-sdk-path
        OUTPUT_VARIABLE MACOS_SDK_PATH
        OUTPUT_STRIP_TRAILING_WHITESPACE
    )
    if(MACOS_SDK_PATH)
        message(STATUS "macOS SDK: ${MACOS_SDK_PATH}")
        set(CMAKE_OSX_SYSROOT ${MACOS_SDK_PATH})
        # Ensure LMDB and other external projects can find system headers
        set(ENV{CFLAGS} "-isysroot ${MACOS_SDK_PATH}")
        set(ENV{CXXFLAGS} "-isysroot ${MACOS_SDK_PATH}")
    endif()
endif()

# Output directory
set(LIB_OUTPUT_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../lib/${TARGET_PLATFORM}")
file(MAKE_DIRECTORY ${LIB_OUTPUT_DIR})

# Barretenberg dependency
# The library can be built from source or fetched as a pre-built artifact
if(BARRETENBERG_SOURCE_DIR)
    # Use local Barretenberg source
    message(STATUS "Using local Barretenberg: ${BARRETENBERG_SOURCE_DIR}")
    add_subdirectory(${BARRETENBERG_SOURCE_DIR} barretenberg EXCLUDE_FROM_ALL)
else()
    # Fetch Barretenberg via FetchContent
    include(FetchContent)

    message(STATUS "Fetching Barretenberg @ ${BB_REF}")

    FetchContent_Declare(
        barretenberg
        GIT_REPOSITORY https://github.com/AztecProtocol/aztec-packages.git
        GIT_TAG ${BB_REF}
        GIT_SHALLOW TRUE
        GIT_PROGRESS TRUE
        GIT_SUBMODULES ""
        SOURCE_SUBDIR barretenberg/cpp
    )

    # Configure Barretenberg build options
    set(TESTING OFF CACHE BOOL "" FORCE)
    set(BENCHMARKS OFF CACHE BOOL "" FORCE)
    set(BUILD_SHARED_LIBS OFF CACHE BOOL "" FORCE)
    set(DISABLE_ASM OFF CACHE BOOL "" FORCE)
    set(MULTITHREADING ON CACHE BOOL "" FORCE)
    
    # Disable warnings-as-errors for Barretenberg to handle stricter compiler checks
    # GCC and Clang have different warning flag names, so we need to check the compiler
    if(CMAKE_CXX_COMPILER_ID STREQUAL "GNU")
        # GCC-specific flags for GCC 13+ stricter checks
        set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -Wno-error=stringop-overflow -Wno-error=missing-field-initializers -Wno-error=array-bounds" CACHE STRING "" FORCE)
    elseif(CMAKE_CXX_COMPILER_ID MATCHES "Clang")
        # Clang-specific flags (Apple Clang and LLVM Clang)
        set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -Wno-error=missing-field-initializers" CACHE STRING "" FORCE)
    endif()

    # Use FetchContent_Populate (instead of MakeAvailable) so we can patch
    # Barretenberg's compile flags before its CMakeLists.txt is processed.
    FetchContent_GetProperties(barretenberg)
    if(NOT barretenberg_POPULATED)
        FetchContent_Populate(barretenberg)

        # Apple Clang 17 triggers -Wvla-cxx-extension but reports version 17,
        # while Barretenberg's fix only applies for Clang >= 18 (LLVM versioning).
        # Patch the version check to cover Apple Clang as well.
        set(_bb_src_cmake "${barretenberg_SOURCE_DIR}/barretenberg/cpp/src/CMakeLists.txt")
        file(READ "${_bb_src_cmake}" _bb_src_cmake_content)
        string(REPLACE
            "CMAKE_CXX_COMPILER_VERSION VERSION_GREATER_EQUAL 18"
            "CMAKE_CXX_COMPILER_VERSION VERSION_GREATER_EQUAL 16"
            _bb_src_cmake_content "${_bb_src_cmake_content}")
        file(WRITE "${_bb_src_cmake}" "${_bb_src_cmake_content}")
        message(STATUS "Patched Barretenberg Clang version check for Apple Clang compatibility")

        # Disable nodejs_module which has CMake errors when NAPI_VERSION is not set
        set(_bb_nodejs_cmake "${barretenberg_SOURCE_DIR}/barretenberg/cpp/src/barretenberg/nodejs_module/CMakeLists.txt")
        if(EXISTS "${_bb_nodejs_cmake}")
            file(READ "${_bb_nodejs_cmake}" _bb_nodejs_cmake_content)
            # Wrap the entire file content in a condition that's always false
            file(WRITE "${_bb_nodejs_cmake}" "# Disabled for wrapper build - nodejs module not needed\nreturn()\n${_bb_nodejs_cmake_content}")
            message(STATUS "Disabled Barretenberg nodejs_module (not needed for wrapper)")
        endif()

        add_subdirectory(${barretenberg_SOURCE_DIR}/barretenberg/cpp ${barretenberg_BINARY_DIR} EXCLUDE_FROM_ALL)
    endif()
endif()

# Create the wrapper static library
add_library(barretenberg_wrapper STATIC
    barretenberg_wrapper.cpp
)

target_include_directories(barretenberg_wrapper
    PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR}/../include
    PRIVATE
        ${barretenberg_SOURCE_DIR}/barretenberg/cpp/src
        ${CMAKE_BINARY_DIR}/_deps/tracy-src/public
        ${CMAKE_BINARY_DIR}/_deps/msgpack-c/src/msgpack-c/include
)

# Link against Barretenberg libraries
target_link_libraries(barretenberg_wrapper
    PRIVATE
        barretenberg
)

# Compiler flags
target_compile_options(barretenberg_wrapper PRIVATE
    -fPIC
    -fvisibility=hidden
    $<$<CONFIG:Release>:-O3>
    $<$<CONFIG:Release>:-DNDEBUG>
)

# Define version (use git ref as version identifier)
target_compile_definitions(barretenberg_wrapper PRIVATE
    BB_VERSION="${BB_REF}"
)

# Set output properties
set_target_properties(barretenberg_wrapper PROPERTIES
    OUTPUT_NAME "barretenberg_wrapper"
    POSITION_INDEPENDENT_CODE ON
)

# Merge all static libraries (wrapper + barretenberg dependencies) into a single
# fat archive so that Go/CGo can link against one -lbarretenberg.
# Generate a merge script at configure time; find runs at build time via sh.
if(CMAKE_SYSTEM_NAME STREQUAL "Darwin")
    file(WRITE "${CMAKE_BINARY_DIR}/merge_libs.sh"
"#!/bin/sh
set -e
libtool -static -o '${LIB_OUTPUT_DIR}/libbarretenberg.a' $(find '${CMAKE_BINARY_DIR}' -name '*.a' ! -path '${LIB_OUTPUT_DIR}/*')
")
elseif(CMAKE_SYSTEM_NAME STREQUAL "Linux")
    file(WRITE "${CMAKE_BINARY_DIR}/merge_libs.sh"
"#!/bin/sh
set -e
tmp=$(mktemp -d)
cd \"$tmp\"
counter=0
for a in $(find '${CMAKE_BINARY_DIR}' -name '*.a' ! -path '${LIB_OUTPUT_DIR}/*'); do
    # Create unique subdirectory for each archive to avoid .o filename collisions
    mkdir -p \"ar_$counter\"
    cd \"ar_$counter\"
    '${CMAKE_AR}' x \"$a\"
    cd ..
    counter=$((counter + 1))
done
# Collect all .o files from all subdirectories
find . -name '*.o' -exec '${CMAKE_AR}' rcs '${LIB_OUTPUT_DIR}/libbarretenberg.a' {} +
cd /
rm -rf \"$tmp\"
")
endif()

add_custom_command(TARGET barretenberg_wrapper POST_BUILD
    COMMAND sh "${CMAKE_BINARY_DIR}/merge_libs.sh"
    COMMENT "Combining static libraries"
)

# Installation
install(TARGETS barretenberg_wrapper
    ARCHIVE DESTINATION lib/${TARGET_PLATFORM}
)

install(FILES
    ${CMAKE_CURRENT_SOURCE_DIR}/../include/barretenberg_wrapper.h
    DESTINATION include
)
