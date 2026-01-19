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

# Barretenberg git reference (branch, tag, or commit hash)
# Note: Barretenberg doesn't use semantic version tags - use branch name or commit
set(BB_REF "master" CACHE STRING "Barretenberg git reference (branch/tag/commit)")

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
        GIT_REPOSITORY https://github.com/AztecProtocol/barretenberg.git
        GIT_TAG ${BB_REF}
        GIT_SHALLOW FALSE
        GIT_PROGRESS TRUE
        GIT_SUBMODULES ""
        SOURCE_SUBDIR cpp
    )

    # Configure Barretenberg build options
    set(TESTING OFF CACHE BOOL "" FORCE)
    set(BENCHMARKS OFF CACHE BOOL "" FORCE)
    set(BUILD_SHARED_LIBS OFF CACHE BOOL "" FORCE)
    set(DISABLE_ASM OFF CACHE BOOL "" FORCE)
    set(MULTITHREADING ON CACHE BOOL "" FORCE)

    FetchContent_MakeAvailable(barretenberg)
endif()

# Create the wrapper static library
add_library(barretenberg_wrapper STATIC
    barretenberg_wrapper.cpp
)

target_include_directories(barretenberg_wrapper
    PUBLIC
        ${CMAKE_CURRENT_SOURCE_DIR}/../include
    PRIVATE
        ${barretenberg_SOURCE_DIR}/cpp/src
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
    ARCHIVE_OUTPUT_DIRECTORY ${LIB_OUTPUT_DIR}
    OUTPUT_NAME "barretenberg"
    POSITION_INDEPENDENT_CODE ON
)

# On Linux, we need to create an archive that includes all dependencies
# This creates a "fat" static library
if(CMAKE_SYSTEM_NAME STREQUAL "Linux" OR CMAKE_SYSTEM_NAME STREQUAL "Darwin")
    # After building, merge all static libraries into one
    add_custom_command(TARGET barretenberg_wrapper POST_BUILD
        COMMAND ${CMAKE_COMMAND} -E echo "Creating combined static library..."
        COMMAND ${CMAKE_AR} -x $<TARGET_FILE:barretenberg_wrapper>
        COMMAND ${CMAKE_AR} -rcs ${LIB_OUTPUT_DIR}/libbarretenberg.a *.o
        COMMAND ${CMAKE_COMMAND} -E remove *.o
        WORKING_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}
        COMMENT "Combining static libraries"
        VERBATIM
    )
endif()

# Installation
install(TARGETS barretenberg_wrapper
    ARCHIVE DESTINATION lib/${TARGET_PLATFORM}
)

install(FILES
    ${CMAKE_CURRENT_SOURCE_DIR}/../include/barretenberg_wrapper.h
    DESTINATION include
)
