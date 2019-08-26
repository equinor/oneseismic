if (SKBUILD)
    # invoked as a part of scikit-build, so this is just a proxy for the python
    # extension cmake. this works around the fundamental limitation in cmake
    # that it looks only for directories with a CMakeLists.txt in it, not for a
    # named file
    include(setup-CMakeLists.txt)
    return ()
endif ()

cmake_minimum_required(VERSION 3.5)
project(core-server LANGUAGES CXX)

find_package(PythonInterp REQUIRED)

if (NOT PYTHON_EXECUTABLE)
    message(SEND_ERROR "Could not find python executable")
    return()
endif()

set(setup.py ${CMAKE_CURRENT_SOURCE_DIR}/setup.py)

add_custom_target(
    core-server ALL
    COMMENT "Building python with setup.py"
    SOURCES ${setup.py}
    DEPENDS ${setup.py}
    VERBATIM
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}

    COMMAND ${PYTHON_EXECUTABLE} ${setup.py}
        # build the extension inplace (really, once its built, copy it to the
        # source tree) so that post-build, the directory can be used to run
        # tests against
        build_ext
            --build-type=${CMAKE_BUILD_TYPE}
        build # setup.py build args
            --cmake-executable ${CMAKE_COMMAND}
            --generator ${CMAKE_GENERATOR}
        -- -Dseismic-cloud_DIR=${CMAKE_CURRENT_BINARY_DIR}/..
)

add_dependencies(core-server seismic-cloud)

# If a cmake install prefix is set, the python module should be installed to
# this path.
if(CMAKE_INSTALL_PREFIX)
    set(prefix --prefix=${CMAKE_INSTALL_PREFIX} --root=/)
endif()

install(CODE "
    execute_process(
        COMMAND ${PYTHON_EXECUTABLE} ${setup.py}
            install
                ${prefix}
                --cmake-executable \"${CMAKE_COMMAND}\"
                --generator \"${CMAKE_GENERATOR}\"
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    )"
)
