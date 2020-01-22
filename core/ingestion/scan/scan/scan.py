import hashlib
import io

import segyio
import segyio._segyio
import numpy as np

textheader_size = 3200
binary_size = 400
header_size = 240

class parseint:
    def __init__(self, endian, default_length = 2):
        self.default_length  = default_length
        self.endian = endian

    def parse(self, i, length = None, signed = True):
        """ Parse endian-sourced integer when read from bytes as big-endian

        segyio reads from the byte buffer as if it was big endian always. However,
        since the regular segyio logic is circumvented, segyio never had the chance
        to byte-swap the buffer so that it behaves as if it was big-endian.

        To work around this, this function assumes the integer was read by segyio,
        and that the file was big-endian.

        Parameters
        ----------
        i : int
            integer read from bytes as big-endian
        length : int, optional, default = None
            size of the source integer in bytes. If no length is passed,
            self.default_length is used

        Returns
        -------
        i : int
        """
        if length is None:
            length = self.default_length

        chunk = i.to_bytes(length = length, byteorder = 'big', signed = signed)
        return int.from_bytes(chunk, byteorder = self.endian, signed = signed)

def scan_binary(stream, endian):
    """ Read info from the binary header

    Read the necessary information from the binary header, and, if necessary,
    seek past the extended textual headers.

    Parameters
    ----------
    stream : stream_like
        an open io.IOBase like stream
    endian : { 'big', 'little' }

    Returns
    -------
    out : dict
        dict of the keys format, samples, sampleinterval and
        byteoffset-first-trace

    Notes
    -----
    If this function succeeds, stream will have read past the binary header and
    the extended text headers.

    If an exception is raised in this function, the stream is left at an
    unspecified position, and must be seeked or reset to behave well defined.
    """
    chunk = stream.read(binary_size)
    binary = segyio.field.Field(buf = chunk, kind = 'binary')

    intp = parseint(endian = endian, default_length = 2)
    # skip extra textual headers
    exth = intp.parse(binary[segyio.su.exth])
    for _ in range(exth):
        stream.seek(textheader_size, whence = io.SEEK_CUR)

    fmt = intp.parse(binary[segyio.su.format])
    if fmt not in [1, 5]:
        msg = 'only IBM float and 4-byte IEEE float supported, was {}'
        raise NotImplementedError(msg.format(fmt))

    return {
        'format': fmt,
        'samples': intp.parse(binary[segyio.su.hns], signed=False),
        'sampleinterval': intp.parse(binary[segyio.su.hdt]),
        'byteoffset-first-trace': 3600 + exth * 3200,
    }

def updated_count_interval(header, d, endian):
    """ Return a dict to updated sample/interval from trace header

    Get a dict of the missing sample-count and sample-interval, suitable as
    a parameter to dict.update.

    Use the output of this function to set the values missing from the binary
    header.

    Parameters
    ----------
    header : segyio.header or dict_like
        dict_like trace header
    d : dict
        OpenVDS metadata from the run and binary header
    endian : { 'little', 'big' }

    Returns
    -------
    updated : dict
        A dict of the sample-count and sample-interval not already present in
        the d parameter
    """
    updated = {}

    intp = parseint(endian = endian, default_length = 2)
    if d.get('samples', 0) == 0:
        updated['samples'] = intp.parse(header[segyio.su.ns])

    if d.get('sampleinterval', 0) == 0:
        updated['sampleinterval'] = intp.parse(header[segyio.su.dt])

    return updated

format_size = {
    1:  4,
    2:  4,
    3:  2,
    5:  4,
    6:  8,
    8:  1,
    9:  8,
    10: 4,
    11: 2,
    12: 8,
    16: 1,
}

def resolve_endianness(big, little):
    if big is None and little is None:
        return 'big'

    if big and not little:
        return 'big'

    if little and not big:
        return 'little'

    msg = 'big and little endian specified, but options are exclusive'
    raise ValueError(msg)

class hashio:
    """Read stream, and calculate running hash

    The sha1 of the file is used as a global, unique identifier, but has to
    be computed. Since scanning for geometry is necessary anyway, and not
    really parallelisable, so it is a good time to also compute the hash.

    Proxies the io.IOBase family of interfaces, and disallows seeking
    backwards. Every read and seek operation is intercepted to compute the hash
    value, but downstream users not made aware of this.
    """
    def __init__(self, stream):
        self.stream = stream
        self.sha1 = hashlib.sha1()

    def read(self, *args):
        chunk = self.stream.read(*args)
        self.sha1.update(chunk)
        return chunk

    def seek(self, offset, whence = io.SEEK_SET):
        if whence != io.SEEK_CUR:
            raise NotImplementedError
        _ = self.read(offset)

    def hexdigest(self):
        return self.sha1.hexdigest()


def scan(stream, primary_word=189, secondary_word=193, little_endian=None, big_endian=None):
    """Scan a file and create an index

    Scan a stream, and produce an index for building a job schedule in further
    ingestion.

    Parameters
    ----------
    args
        for expected members, see main in __main__.py
    stream :
        io.IOBase compatible interface

    Returns
    -------
    d : dict
    """
    from .segmenter import segmenter

    endian = resolve_endianness(big_endian, little_endian)
    out = {
        'byteorder': endian,
    }

    stream = hashio(stream)
    stream.seek(textheader_size, whence = io.SEEK_CUR)

    out.update(scan_binary(stream, endian))

    chunk = stream.read(header_size)
    header = segyio.field.Field(buf = chunk, kind = 'trace')

    out.update(updated_count_interval(header, out, endian))

    tracelen = out['samples'] * format_size[out['format']]

    seg = segmenter(
        primary = primary_word,
        secondary = secondary_word,
        endian = endian,
    )

    seg.add(header)
    stream.seek(tracelen, whence = io.SEEK_CUR)

    trace_count = 0
    while True:
        chunk = stream.read(header_size)
        if len(chunk) == 0:
            break

        if len(chunk) != header_size:
            msg = 'file truncated at trace {}'.format(trace_count)
            raise RuntimeError(msg)

        header = segyio.field.Field(buf = chunk, kind = 'trace')
        seg.add(header)
        stream.seek(tracelen, whence = io.SEEK_CUR)
        trace_count += 1

    out['guid'] = stream.hexdigest()
    interval = out['sampleinterval']
    samples = map(int, np.arange(0, out['samples'] * interval, interval))
    out['dimensions'] = [
        seg.primaries,
        seg.secondaries,
        list(samples),
    ]
    return out
