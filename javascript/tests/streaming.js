const one = require('oneseismic');
const fs  = require('fs');

/*
 * Tests are written targeting node for easier running from CI and command
 * line, but could just as well target chrome headless. Testing can be annoying
 * since node and browser support different APIs for files etc., so a test
 * suite that target both just seems annyoing.
 */
function equals(got, want, name) {
    // stringify to get cheap-and-dirty array-equals
    // https://stackoverflow.com/questions/7837456/how-to-compare-arrays-in-javascript
    got  = JSON.stringify(got);
    want = JSON.stringify(want);
    if (got !== want) {
        const msg = (
            "assertion failed: " +
            "got " + name + " = " + got + "; " +
            "want " + want
        );
        throw new Error(msg);
    }
}

function assert(check, msg) {
    if (!check) {
        throw new Error("assertion failed: " + msg);
    }
}

function approx(got, want, name, epsilon) {
    if (epsilon === undefined) epsilon = 1e-5;
    const diff = Math.abs(got - want);
    if (diff > epsilon) {
        const msg = (
            "assertion failed: " +
            "got " + name + " = " + got + "; " +
            "want ~" + want + 
            " (epsilon " + epsilon + ")"

        );
        throw new Error(msg);
    }
}

function sum(xs) {
    return xs.reduce((acc, v) => acc + v);
};

one().then(one => {
    (function single_buffer_decoding() {
        const decode  = one.stream_decoder();
        fs.readFile('curtain-1.bin', null, (err, chunk) => {
            assert(!err, "reading file");

            decode(chunk);
            const res = decode(null);
            const head = res[0];
            const body = res[1];
            const data = body['data']
            equals(head.function, one.functionid.curtain, "functionid");
            equals(head.ndims, 3, "ndims");
            equals(data['shape'], [5, 850], "shape");
            approx(sum(data['data']), -1.589, "data.sum", 1e-2);
        });
    })();

    (function small_chunk_decoding() {
        const decode  = one.stream_decoder();
        fs.readFile('curtain-1.bin', null, (err, contents) => {
            assert(!err, "reading file");

            var res;
            const len = contents.byteLength;
            for (let i = 0; i < len; i += 5) {
                const end = Math.min(i + 5, len);
                res = decode(contents.slice(i, end));
            }

            const head = res[0];
            const body = res[1];
            const data = body['data']
            equals(head.function, one.functionid.curtain, "functionid");
            equals(head.ndims, 3, "ndims");
            equals(data['shape'], [5, 850], "shape");
            approx(sum(data['data']), -1.589, "data.sum", 1e-2);
        });
    })();
})
