cmake_minimum_required(VERSION 3.9)
project(oneseismic LANGUAGES CXX)

include(CheckIncludeFile)
include(CTest)
include(GNUInstallDirs)
include(TestBigEndian)

option(BUILD_PYTHON  "Build Python library"                ON)
# Start and stop a redis instance specific for the integration tests. If you
# turn this option off, the integration tests will still build, but fail unless
# pointed to a running redis instance. Using an external redis instance can be
# useful when running tests while simultaneously monitoring traffic to and from
# the server, and looking at commands issued. Generally this option should be
# ON.
option(PRIVATE_REDIS "Use private and test-specific redis" ON)

# Usually, a randomly-generated 5-digit port should be fine, but in the case of
# collision or when public redis instances, a specific port can be used.
#
# Defaults to localhost:<random-port>, but either part or the full address can
# be specified.
string(RANDOM LENGTH 1 ALPHABET 12345     head-random-port)
string(RANDOM LENGTH 4 ALPHABET 012345689 tail-random-port)
string(CONCAT random-port ${head-random-port} ${tail-random-port})
set(REDIS_PORT ${random-port}                CACHE STRING "Redis port")
set(REDIS_HOST "localhost"                   CACHE STRING "Redis host")
set(REDIS_ADDR "${REDIS_HOST}:${REDIS_PORT}" CACHE STRING "Redis address")

add_library(json INTERFACE)
target_include_directories(json INTERFACE external/nlohmann)

add_library(clara INTERFACE)
target_include_directories(clara INTERFACE external/clara)

find_package(fmt        REQUIRED)
find_package(Threads    REQUIRED)
find_package(spdlog     REQUIRED)

find_package(PkgConfig REQUIRED)
pkg_check_modules(hiredis    IMPORTED_TARGET REQUIRED hiredis)
pkg_check_modules(zmq        IMPORTED_TARGET REQUIRED libzmq)

find_program(REDIS_SERVER_EXECUTABLE redis-server)
find_program(REDIS_CLI_EXECUTABLE    redis-cli)

if (NOT MSVC)
    # assuming gcc-style options
    # add warnings in debug mode
    list(APPEND warnings-c++ -Wall
                             -Wextra
                             -pedantic
                             -Wformat-nonliteral
                             -Wcast-align
                             -Wpointer-arith
                             -Wmissing-declarations
                             -Wcast-qual
                             -Wwrite-strings
                             -Wchar-subscripts
                             -Wredundant-decls
    )
endif ()

set(CMAKE_CXX_STANDARD 14)

add_library(oneseismic
    src/base64.cpp
    src/geometry.cpp
    src/transfer.cpp
    src/messages.cpp
    src/load_balancer.cpp
)
add_library(oneseismic::oneseismic ALIAS oneseismic)
target_include_directories(oneseismic
    PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include>
        ${CURL_INCLUDE_DIRS}
    PRIVATE
        include
        external/nlohmann
)
target_link_libraries(oneseismic
    PUBLIC
        fmt::fmt
        PkgConfig::hiredis
        PkgConfig::zmq
)

install(
    TARGETS
        oneseismic
    EXPORT
        oneseismic-export
    ARCHIVE DESTINATION
        ${CMAKE_INSTALL_LIBDIR}
    LIBRARY DESTINATION
        ${CMAKE_INSTALL_LIBDIR}
    RUNTIME DESTINATION
        ${CMAKE_INSTALL_BINDIR}
)
install(DIRECTORY include/ DESTINATION include)
install(
    EXPORT
        oneseismic-export
    DESTINATION
        ${CMAKE_INSTALL_DATAROOTDIR}/oneseismic/cmake
    FILE
        oneseismic-config.cmake
    NAMESPACE
        oneseismic::
)
export(
    TARGETS
        oneseismic
    FILE
        oneseismic-config.cmake
    NAMESPACE
        oneseismic::
)
set(ONESEISMIC_LIB_CMAKECONFIG_DIR ${CMAKE_CURRENT_BINARY_DIR} CACHE PATH "")

add_subdirectory(external/catch2)

add_executable(tests
    tests/testsuite.cpp
    tests/geometry.cpp
    tests/messages.cpp
    tests/load_balancer.cpp
)
target_link_libraries(tests
    PRIVATE
        catch2
        oneseismic::oneseismic
        fmt::fmt
        Threads::Threads
        json
)
add_test(NAME unit-tests COMMAND tests)

# Check if all dependencies for integration tests are available. It's written
# as a series of if () to give a complete report of missing dependencies, and
# to make it easy to add new dependencies without complicated list and
# exhaustiveness logic.
set(all-integration-test-dependencies TRUE)
if (PRIVATE_REDIS)
    if (NOT REDIS_SERVER_EXECUTABLE)
        set(all-integration-test-dependencies FALSE)
        message(WARNING
            "redis-server not found - not building integration tests."
            "Install redis-server or set with -DREDIS_SERVER_EXECUTABLE=<path>"
        )
    endif ()
    if (NOT REDIS_CLI_EXECUTABLE)
        set(all-integration-test-dependencies FALSE)
        message(WARNING
            "redis-cli not found - not building integration tests."
            "Install redis-cli or set with -DREDIS_CLI_EXECUTABLE=<path>"
        )
    endif ()
endif ()

if (all-integration-test-dependencies)
    # If a public redis instance is used, don't register the redis start-stop
    # jobs
    if (PRIVATE_REDIS)
        add_test(
            NAME
                redis-start
            COMMAND
                ${REDIS_SERVER_EXECUTABLE}
                --port ${REDIS_PORT}
                --daemonize yes
        )
        add_test(
            NAME
                redis-stop
            COMMAND
                ${REDIS_CLI_EXECUTABLE} -p ${REDIS_PORT} SHUTDOWN
        )
        add_test(
            NAME
                redis-try-stop
            COMMAND
                ${REDIS_CLI_EXECUTABLE} -p ${REDIS_PORT} SHUTDOWN
        )
        # "try stop" - try to stop a running redis instance and always pass the
        # test. This kills orphaned redis sessions from previous ctest
        # invocations before starting the tests again, for a better user
        # experience. This will not be sufficient or even mask other possible
        # errors, but they will (hopefully) show up elsewhere anyway. ctest is
        # a poor total process management tool, but this should cover 99% of
        # daily use.
        #
        # It's not too uncommon to abort ctest with C-c or SIGINT, which means
        # the cleanup never runs and the daemonized redis instance keeps
        # running. While starting a new daemonized process silently works,
        # there's now a risk of test data leaing across invocations, so we want
        # a fresh instance every time.
        #
        # Daemonized redis will still "leak" if ctest is killed or terminated
        # somehow, but that is largely outside our control. Ctest will only
        # clean up redis that runs at REDIS_PORT, which means that re-created
        # build directories are unable to clean up orphaned redis instances.
        # There really aren't too many problems that should arise from this, so
        # it's an acceptable defect.
        set_tests_properties(redis-try-stop PROPERTIES
            PASS_REGULAR_EXPRESSION
                ".*"
            FIXTURES_SETUP
                redis-try-stop
        )
        set_tests_properties(redis-start PROPERTIES
            FIXTURES_REQUIRED
                redis-try-stop
            FIXTURES_SETUP
                redis
        )
        set_tests_properties(redis-stop PROPERTIES
            FIXTURES_CLEANUP
                redis
        )
    endif ()
    target_compile_definitions(tests
        PRIVATE
            INTEGRATION_TESTS_REDIS
    )
    add_test(
        NAME
            integration-tests
        COMMAND
            tests --redis ${REDIS_ADDR} "[integration]"
    )
    # Always set the fixutre-requirement - if the fixture does not exist, it
    # silently gets ignored
    set_tests_properties(integration-tests PROPERTIES
        FIXTURES_REQUIRED
            redis
    )
endif()
