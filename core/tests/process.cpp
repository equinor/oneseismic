#include <catch/catch.hpp>

#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>
#include <oneseismic/process.hpp>

one::slice_fetch default_slice_fetch() {
    one::slice_fetch input;
    input.pid   = "some-pid";
    input.token = "some-token";
    input.guid  = "some-guid";

    input.storage_endpoint = "some-endpoint";
    input.shape      = { 64, 64, 64 };
    input.shape_cube = { 720, 860, 251 };

    input.dim    = 0;
    input.lineno = 0;
    return input;
}
