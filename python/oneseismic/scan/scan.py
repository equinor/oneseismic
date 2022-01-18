import hashlib
import io

import segyio
import segyio._segyio
import numpy as np

from segyio.tools import native
from .scanners import parseint

textheader_size = 3200
binary_size = 400
header_size = 240

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

def tonative(trace, format, endian):
    if endian == 'little': trace.byteswap(inplace = True)
    return native(trace, format = format, copy = False)

def scan(stream, scanners, endian):
    """Scan a file and build an index from action

    Scan a stream, and produce an index for building a job schedule in further
    ingestion. The actual indexing is handled by the scanner interface, as it
    turns out a lot of useful tasks boil down to scanning the headers of a
    SEG-Y file.

    Parameters
    ----------
    stream
        io.IOBase compatible interface
    scanner : list of scanners
        Objects with a scanner compatible interface
    endian : str
        endianness of segy-file

    Returns
    -------
    d : dict

    Notes
    -----
    If multiple scanners include the same key in their report, the first
    scanner will take precedence. Ideally this should not happen, and it's the
    callers responsibility to pass compatible scanners.
    """
    stream.seek(textheader_size, io.SEEK_CUR)
    chunk = stream.read(binary_size)
    binary = segyio.field.Field(buf = chunk, kind = 'binary')

    # Information needed by the scan program
    intp      = parseint(endian)
    extheader = intp.parse(binary[segyio.su.exth],   length = 2)
    fmt       = intp.parse(binary[segyio.su.format], length = 2)
    samples   = intp.parse(binary[segyio.su.hns],    length = 2, signed = False)

    for scanner in scanners:
        scanner.scan_binary_header(binary)

    if extheader > 0:
        stream.seek(exthead * textheader_size, io.SEEK_CUR)

    tracelen  = samples * format_size[fmt]
    tracesize = header_size + tracelen

    trace_count = 0
    while True:
        chunk = stream.read(tracesize)
        if len(chunk) == 0:
            break

        if len(chunk) != tracesize:
            msg = 'file truncated at trace {}'.format(trace_count)
            raise RuntimeError(msg)

        header  = segyio.field.Field(buf = chunk[:header_size], kind = 'trace')
        # dtype must match fmt size. Endianess and data type are matched later
        dt = "i{}".format(format_size[fmt])
        trace = np.frombuffer(chunk[header_size:], dtype=dt)
        trace = tonative(trace.copy(), fmt, endian)

        for scanner in scanners:
            scanner.scan_trace_header(header)
            scanner.scan_trace_samples(trace)

        trace_count += 1

    report = {}
    for scanner in reversed(scanners):
        report.update(scanner.report())

    return report
