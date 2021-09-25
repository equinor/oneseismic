function f32Array(size) {
    const offset = Module._malloc(size * 4);
    Module.HEAPF32.set(new Float32Array(size), offset / 4);
    return {
        "data": Module.HEAPF32.subarray(offset / 4, offset / 4 + size),
        "offset": offset
    }
}

function splitshapes(shapes) {
    /*
     * This function is a bit awkward, but unpacks the tuples
     *     [3 201 720 1 2 20 15] => [[201 720 1], [20, 15]]
     * for all the attributes in the response
     */
    let i = 0
    const xs = new Array()
    while (i < shapes.length) {
        const n = shapes[i];
        i += 1;
        xs.push(shapes.slice(i, i + n));
        i += n;
    }
    return xs;
}

function emvector_to_array(xs) {
    return new Array(xs.size()).fill(0).map((_, id) => xs.get(id))
}

function native_process_header(primitive) {
    /*
     * Using the C++-backed process-header is pretty awkward, since arrays
     * aren't really arrays (they're embind VectorInt etc). Convert into a
     * js-native object.
     */
    const head = {}
    head.attrs    = emvector_to_array(primitive.attrs)
    head.index    = emvector_to_array(primitive.index)
    head.shapes   = emvector_to_array(primitive.shapes)
    head.labels   = emvector_to_array(primitive.labels)
    head.ndims    = primitive.ndims
    head.function = primitive.function
    return head
}

Module.stream_decoder = function() {
    // TODO: maybe this should make an object proper, rather than just a very
    // stateful function
    const decoder = new Module.decoder()
    decoder.reset();
    const d = {};
    let header = null;
    let done = false;

    return (chunk) => {
        if (done) {
            return [header, d];
        }

        if (chunk) {
            decoder.buffer(chunk);
        }

        const ok = decoder.process();
        if (ok === Module.status.done) {
            // TODO: This delete must be guarded so it always runs; otherwise
            // it will leak on errors or incomplete decoding
            decoder.delete();
            done = true;
            return [header, d];
        }

        if (header === null) {
            // not enough bytes to read the header - suspend and wait for more 
            if (!decoder.header_ready()) {
                return null;
            }

            const cppheader = decoder.header_get();
            header = native_process_header(cppheader);
            cppheader.delete()
            const shapes = splitshapes(header.shapes);
            const attrs  = header.attrs;
            /*
             * allocate arrays and give to the decoder. When process returns
             * done(), all data has been read and is available in the d dict
             */
            for (let i = 0; i < shapes.length; i++) {
                const attr  = attrs[i];
                const shape = shapes[i];
                const fdata = f32Array(shape.reduce((acc, v) => acc * v));
                decoder.register_writer(attr, fdata.offset);
                d[attr] = { shape: shape, data: fdata.data };
            }
        }

        return [header, d];
    };
}
