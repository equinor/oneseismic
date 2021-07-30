cmake_minimum_required(VERSION 3.13)
project(oneseismic-python LANGUAGES C CXX)

set(CMAKE_CXX_STANDARD 14)
set(CMAKE_CXX_VISIBILITY_PRESET "hidden")
set(CMAKE_C_VISIBILITY_PRESET   "hidden")

if (SKBUILD)
  # scikit-build does not add your site-packages to the search path automatically,
  # so we need to add it _or_ the pybind11 specific directory here.
  execute_process(
    COMMAND
      "${PYTHON_EXECUTABLE}" -c
      "import pybind11; print(pybind11.get_cmake_dir())"
    OUTPUT_VARIABLE _tmp_dir
    OUTPUT_STRIP_TRAILING_WHITESPACE
  )
  list(APPEND CMAKE_PREFIX_PATH "${_tmp_dir}")
endif()

find_package(PythonExtensions REQUIRED)
find_package(pybind11 REQUIRED)
find_package(oneseismic REQUIRED)

# fmt is not required - oneseismic supports building a slim core (only the
# assembler) , which makes it easier to build for python.
# dev-builds usually link against the full package however, which has more
# dependencies, so looking and accepting missing packages is a good balance
# between use cases
find_package(fmt)

add_library(decoder MODULE oneseismic/decoder.cpp)
python_extension_module(decoder)
target_link_libraries(decoder
    pybind11::headers
    oneseismic::oneseismic
)

install(
    TARGETS decoder
    LIBRARY DESTINATION oneseismic
    RUNTIME DESTINATION oneseismic
)
