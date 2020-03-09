cmake_minimum_required(VERSION 3.5.0)
project(oneseismic-python)

if (NOT BUILD_PYTHON)
    return()
endif ()

find_package(PythonInterp 3 REQUIRED)

if (NOT PYTHON_EXECUTABLE)
    message(WARNING "Could not find python - skipping python bindings")
    message(WARNING "Select specific python distribution with "
                    "-DPYTHON_EXECUTABLE=bin/python")
    return()
endif()

set(setup.py ${CMAKE_CURRENT_SOURCE_DIR}/setup.py)

if (CMAKE_BUILD_TYPE)
    # use the cmake_build_type of the source project, unless it has been
    # specifically overriden
    set(ONESEISMIC_PYTHON_BUILD_TYPE
        --build-type=${CMAKE_BUILD_TYPE}
        CACHE STRING "override CMAKE_BUILD_TYPE in python extension"
    )
endif ()

add_custom_target(oneseismic-python
    ALL
    COMMENT "Building python library with setup.py"
    SOURCES ${setup.py}
    VERBATIM
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
	
    COMMAND ${PYTHON_EXECUTABLE} ${setup.py}
        # build the extension inplace (really, once its built, copy it to the
        # source tree) so that post-build, the directory can be used to run
        # tests against
        build_ext --inplace
        build # setup.py build args
	    --cmake-executable ${CMAKE_COMMAND}
	    --generator ${CMAKE_GENERATOR}
	    ${ONESEISMIC_PYTHON_BUILD_TYPE}
        -- # cmake to the extension
	    -Doneseismic_DIR=${ONESEISMIC_LIB_CMAKECONFIG_DIR}
            # "install" to the python/oneseismic dir with rpath, so there's no need
            # to fiddle with environment in ctest to load the core library from
            # the build tree
	    -DCMAKE_INSTALL_RPATH_USE_LINK_PATH=ON
	    -DCMAKE_INSTALL_RPATH=$<TARGET_FILE_DIR:oneseismic::oneseismic>
	    -DCMAKE_INSTALL_NAME_DIR=$<TARGET_FILE_DIR:oneseismic::oneseismic>
)


add_dependencies(oneseismic-python oneseismic::oneseismic)

install(CODE "
    if (DEFINED ENV{DESTDIR})
        get_filename_component(abs-destdir \"\$ENV{DESTDIR}\" ABSOLUTE)
        set(root_destdir --root \${abs-destdir})
    endif()
    execute_process(
        COMMAND ${PYTHON_EXECUTABLE} ${setup.py}
            install
                \${root_destdir}
                --cmake-executable \"${CMAKE_COMMAND}\"
                --generator \"${CMAKE_GENERATOR}\"
		${ONESEISMIC_PYTHON_BUILD_TYPE}
            --
                -DCMAKE_INSTALL_RPATH_USE_LINK_PATH=OFF
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    )"
)

# run tests with setup.py test
# this is very slow compared to invoking pytest directly, but setuptools will
# copy the built extension into the tree as it sees fit
#
# use --skip-cmake, otherwise running the tests would trigger a build with
# different args to setup.py, rebuilding the python lib (and wrongly so as it
# either won't find oneseismic or picked up on a system installed one)
add_test(
    NAME python.unit
    COMMAND ${PYTHON_EXECUTABLE} ${setup.py} --skip-cmake test
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
)
