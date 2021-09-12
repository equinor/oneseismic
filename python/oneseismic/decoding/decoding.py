import numpy as np

from . import decoder

def splitshapes(xs):
    while len(xs) > 0:
        n, xs = xs[0], xs[1:]
        yield xs[:n]
        xs = xs[n:]

def allocarrays(header):
    shapes = splitshapes(header.shapes)

    d = {}
    for attr, shape in zip(header.attrs, shapes):
        a = np.zeros(shape = shape, dtype = 'f4')
        d[attr] = a

    return d

class process_header:
    """Make python-native header object

    The backing pybind11 object will initialize fresh lists every time a field
    is accessed, which is not expected python behaviour. This functions
    transforms the C++ backed object into an equivalent python-native object
    with regular reference mechanics.
    """
    def __init__(self, h):
        self.attrs    = h.attrs
        self.ndims    = h.ndims
        self.index    = h.index
        self.function = h.function
        self.shapes   = h.shapes
        self.labels   = h.labels

def decode_stream(stream, dec = None):
    """Decode a stream

    Decode a stream into python-native objects. This function can be called on
    any iterable byte stream.

    Pass your own decoder instance to re-use its buffers, otherwise a fresh
    instance is created for you.

    Parameters
    ----------
    stream : iterable of memory_views
    dec : oneseismic.decoding.decoder

    Returns
    -------
    decoded
        The decoded response, which is suited for passing to decoding.numpy()
        and friends

    Examples
    --------
    >>> p = get_process()
    >>> r = requests.get(p.stream(), headers = p.headers(), stream = True)
    >>> decoded = decode_stream(r.iter_content(None))
    >>> decoding.numpy(decoded)
    [[0.7283162   0.00633962  0.02923059 ...  1.3414259   0.39454985]
    [ 2.471489    0.5148768  -0.28820574 ...  3.3378954   1.1355114 ]
    [ 1.9840803  -0.05085047 -0.47386587 ...  4.450244    2.9178839 ]]
    """
    if dec is None:
        dec = decoder.decoder()
    else:
        dec.reset()

    for chunk in stream:
        status = dec.buffer_and_process(chunk)
        if status != dec.status.paused:
            raise RuntimeError(f'expected status == paused; was {status}')

        head = dec.header()
        if head is not None:
            head = process_header(head)
            break
    else: #no-break
        raise RuntimeError('end-of-stream, but header is not decoded yet')

    d = allocarrays(head)
    for name, a in d.items():
        dec.register_writer(name, a)

    done = dec.status.done
    for chunk in stream:
        status = dec.buffer_and_process(chunk)
        if status == done:
            break
    else: #no-break
        status = dec.process()

    # assert len(stream) == 0

    if status != done:
        raise RuntimeError('end-of-stream, but message is not complete')

    return head, d

def numpy(decoded):
    head, d = decoded
    return d[head.attrs[0]].squeeze()
