Building the javascript library can be daunting

# Using Docker
The easiest way to get started, and probably easiest to automate for packaging
purposes, is to build using docker.

Upstream publish the emscripten/emsdk image, but that image does not include
msgpack. You can build a custom image with msgpack included using the
emscripten.dockerfile

    cd javascript/
    docker build -f emscripten.dockerfile -t oneseismic/emscripten

Then, in order to build, run the build.sh in the docker container. Since it
needs the core/ directory it must be run from the oneseismic root:

    docker run --rm -v $(pwd):/src -u $(id -u):$(id -g) \
        oneseismic/emscripten javascript/build.sh /src/js

The options set in build.sh are not authorative, but a very good starting
point.

# Using system emscripten + cmake
If you already have emscripten installed on your system, you can run build.sh
(or manually run the commands in it) directly. Please note that emscripten does
not like system installed headers, so msgpack (the only real dependency) should
be available at some non-system path and explicitly included with
`-DCMAKE_CXX_FLAGS=-Ipath/msgpack`

# Demos
There are a couple of demos on how to use the oneseismic decoder from
javascript in the demo/ directory - one for node.js, and one for vanilla
browser javascript.

Please note that you libraries probably should be built modularized for
node.js.
