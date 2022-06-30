Building the javascript library can be daunting

# Using Docker
The easiest way to get started, and probably easiest to automate for packaging
purposes, is to build using docker.

Upstream publish the emscripten/emsdk image, but that image does not include
msgpack.

The emscripten.dockerfile can be used to retrieve dependencies and build the 
javascript library. The resulting image will have the complete WASM based library in
/wasm

To build the image:

    docker build -f javascript/emscripten.dockerfile -t oneseismic/emscripten .

Buildkit can be used to have the build artifacts copied to local path after build
has completed:

    DOCKER_BUILDKIT=1 docker build --output type=local,dest=<local_path> \
        -f javascript/emscripten.dockerfile .

The options set in build.sh are not authorative, but a very good starting
point.

# Using system emscripten + cmake
If you already have emscripten installed on your system, you can run build.sh
(or manually run the commands in it) directly. To run build.sh directly there are
three positional arguments that need to be supplied:

    ./buid.sh <build_path> <include_path> <toolchain_file>

where:
    <build_path> - Path to write build artifacts
    <include_path> - Path to msgpack-c development library. Note this should not be system
        installed version, a seperate version should be set up
        (for example by cloning https://github.com/msgpack/msgpack-c)
    <toolchain_file> - Path to Emscripten.cmake file, this will vary depending on how
        emscripten was installed


# Demos
There are a couple of demos on how to use the oneseismic decoder from
javascript in the demo/ directory - one for node.js, and one for vanilla
browser javascript.


# Tests and test-data
Test-data includes stored binary messages, to avoid bringing up a bunch of
infrastructure and to make the tests fast & robust to run. This may lead to
some churn and repo ballooning since message formats can get outdated fast, so
a better long-term strategy is necessary.
