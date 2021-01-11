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


def scan(stream, action):
    """Scan a file and build an index from action

    Scan a stream, and produce an index for building a job schedule in further
    ingestion. The actual indexing is handled by the action interface, as it
    turns out a lot of useful tasks boil down to scanning the headers of a
    SEG-Y file.

    Parameters
    ----------
    stream
        io.IOBase compatible interface
    action : scanner
        An object with a scanner compatible interface

    Returns
    -------
    d : dict
    """
    stream.seek(textheader_size, io.SEEK_CUR)
    action.scan_binary(stream)

    chunk = stream.read(header_size)
    header = segyio.field.Field(buf = chunk, kind = 'trace')
    action.add(header)
    tracelen = action.tracelen()
    stream.seek(tracelen, io.SEEK_CUR)

    trace_count = 1
    while True:
        chunk = stream.read(header_size)
        if len(chunk) == 0:
            break

        if len(chunk) != header_size:
            msg = 'file truncated at trace {}'.format(trace_count)
            raise RuntimeError(msg)

        header = segyio.field.Field(buf = chunk, kind = 'trace')
        action.add(header)
        stream.seek(tracelen, io.SEEK_CUR)
        trace_count += 1

    return action.report()
