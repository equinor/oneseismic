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

class scanner:
    """Base class and interface for scanner

    A scanner can be plugged into the scan.__main__ logic and scan.scan
    function as an action, which implements the indexing and reporting logic
    for a type of scan. All scans involve parsing the trace headers, and write
    some report.
    """
    def __init__(self, endian):
        self.endian = endian
        self.intp = parseint(endian = endian, default_length = 4)
        self.observed = {}

    def scan_binary(self, binary):
        """Scan a SEG-Y binary header

        Parameters
        ----------
        binary : segyio.Field or dict_like

        Returns
        -------
        skip : int
            Number of external headers to skip
        """
        skip = self.intp.parse(binary[segyio.su.exth], length = 2)
        fmt = self.intp.parse(binary[segyio.su.format], length = 2)
        if fmt not in [1, 5]:
            msg = 'only IBM float and 4-byte IEEE float supported, was {}'
            raise NotImplementedError(msg.format(fmt))

        samples = self.intp.parse(
            binary[segyio.su.hns],
            length = 2,
            signed = False,
        )
        interval = self.intp.parse(binary[segyio.su.hdt], length = 2)
        self.observed.update({
            'byteorder': self.endian,
            'format': fmt,
            'samples': samples,
            'sampleinterval': interval,
            'byteoffset-first-trace': 3600 + skip * 3200,
        })
        return skip

    def scan_first_header(self, header):
        """Update metadata with (first) header information

        Some metadata is not necessarily well-defined or set in the binary
        header, but instead inferred from parsing the first trace header.

        Generally, scanners themselves should call this function, not users.

        Parameters
        ----------
        header : segyio.header or dict_like
            dict_like trace header
        """
        intp = parseint(endian = self.endian, default_length = 2)
        if self.observed.get('samples', 0) == 0:
            self.observed['samples'] = intp.parse(header[segyio.su.ns])

        if self.observed.get('sampleinterval', 0) == 0:
            self.observed['sampleinterval'] = intp.parse(header[segyio.su.dt])

    def tracelen(self):
        return self.observed['samples'] * format_size[self.observed['format']]

    def report(self):
        """Report the result of a scan

        The default implementation really only deals with file-specific
        geometry. Implement this method for custom scanners.

        Returns
        -------
        report : dict
        """
        return dict(self.observed)

    def add(self, header):
        """Add a new header to the index

        This is mandatory to implement for scanners
        """
        raise NotImplementedError

class lineset(scanner):
    """Scan the lineset

    Scan the set of lines in the survey, i.e. set of in- and crossline pairs.
    The report after a completed scan is a suitable input for the upload
    program.
    """
    def __init__(self, primary, secondary, endian):
        super().__init__(endian)
        # the header words for the in/crossline
        # usually this will be 189 and 193
        self.key1 = primary
        self.key2 = secondary

        self.key1s = set()
        self.key2s = set()

        # keep track of the last trace with a given inline (key1). This allows
        # the upload program to track if all traces that belong to a line have
        # been read, and buffers can be flushed
        self.last1s = {}
        self.traceno = 0

    def add(self, header):
        key1 = self.intp.parse(header[self.key1])
        key2 = self.intp.parse(header[self.key2])
        self.key1s.add(key1)
        self.key2s.add(key2)
        self.last1s[key1] = self.traceno
        self.traceno += 1
        # TODO: detect key-pair duplicates?
        # TODO: check that the sample-interval is consistent across the file?

    def report(self):
        r = super().report()
        interval = r['sampleinterval']
        samples = map(int, np.arange(0, r['samples'] * interval, interval))
        r['dimensions'] = [
            sorted(self.key1s),
            sorted(self.key2s),
            list(samples),
        ]
        r['key1-last-trace'] = self.last1s
        r['key-words'] = [self.key1, self.key2]
        return r

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
    chunk = stream.read(binary_size)
    binary = segyio.field.Field(buf = chunk, kind = 'binary')
    skip = action.scan_binary(binary)
    if skip > 0:
        stream.seek(skip * textheader_size, io.SEEK_CUR)

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
