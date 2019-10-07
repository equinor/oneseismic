import hashlib
import io

import segyio
import segyio._segyio

textheader_size = 3200
binary_size = 400
header_size = 240

class parseint:
    def __init__(self, endian, default_length = 2):
        self.default_length  = default_length
        self.endian = endian

    def parse(self, i, length = None):
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

        chunk = i.to_bytes(length = length, byteorder = 'big', signed = True)
        return int.from_bytes(chunk, byteorder = self.endian, signed = True)

def scan_binary(stream, endian):
    """ Read OpenVDS info from the binary header

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
        dict of the OpenVDS keys dataSampleFormatCode, sampleCount, and
        sampleInterval

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
        'dataSampleFormatCode': fmt,
        'sampleCount': intp.parse(binary[segyio.su.hns]),
        'sampleInterval': intp.parse(binary[segyio.su.hdt]),
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
    if d.get('sampleCount', 0) == 0:
        updated['sampleCount'] = intp.parse(header[segyio.su.ns])

    if d.get('sampleInterval', 0) == 0:
        updated['sampleInterval'] = intp.parse(header[segyio.su.dt])

    return updated

word_widths = {
    1:   'FourByte',
    5:   'FourByte',
    17:  'FourByte',
    21:  'FourByte',
    25:  'FourByte',
    29:  'TwoByte',
    71:  'TwoByte',
    73:  'FourByte',
    77:  'FourByte',
    81:  'FourByte',
    84:  'FourByte',
    89:  'TwoByte',
    109: 'TwoByte',
    115: 'TwoByte',
    117: 'TwoByte',
    181: 'FourByte',
    185: 'FourByte',
    189: 'FourByte',
    193: 'FourByte',
}

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

word_numerals = {
    'TraceSequenceNumber': 1,
    'TraceSequenceNumberWithinFile': 5,
    'EnergySourcePointNumber': 17,
    'EnsembleNumber': 21,
    'TraceNumberWithinEnsemble': 25,
    'TraceIdentificationCode': 29,

    'scalar':          71,
    'Scalar':          71,
    'CoordinateScale': 71,

    'source-x':          73,
    'Source-X':          73,
    'sourcex':           73,
    'SourceX':           73,
    'SourceXCoordinate': 73,

    'source-y':          77,
    'Source-Y':          77,
    'sourcey':           77,
    'SourceY':           77,
    'SourceYCoordinate': 77,

    'Group-X':             81,
    'group-x':             81,
    'GroupX':              81,
    'groupx':              81,
    'ReceiverXCoordinate': 81,
    'Receiver-X':          81,
    'ReceiverX':           81,
    'receiver-x':          81,
    'receiverx':           81,
    'GroupXCoordinate':    81,

    'Group-Y':             84,
    'group-y':             84,
    'GroupY':              84,
    'groupy':              84,
    'ReceiverYCoordinate': 84,
    'Receiver-Y':          84,
    'ReceiverY':           84,
    'receiver-y':          84,
    'receivery':           84,
    'GroupyCoordinate':    84,

    'CoordinateUnits': 89,
    'StartTime': 109,
    'NumSamples': 115,
    'SampleInterval': 117,

    'cdp-x':               181,
    'cdpx':                181,
    'CDP-X':               181,
    'CDPXCoordinate':      181,
    'easting':             181,
    'Easting':             181,
    'EnsembleXCoordinate': 181,

    'cdp-y':               185,
    'cdpy':                185,
    'CDP-Y':               185,
    'CDPYCoordinate':      185,
    'northing':            185,
    'Northing':            185,
    'EnsembleYCoordinate': 185,

    'inline':       189,
    'Inline':       189,
    'InLine':       189,
    'InlineNumber': 189,
    'InLineNumber': 189,

    'crossline':       193,
    'Crossline':       193,
    'CrossLine':       193,
    'CrosslineNumber': 193,
    'CrossLineNumber': 193,
}

def resolve_endianness(big, little):
    if big is None and little is None:
        return 'BigEndian'

    if big and not little:
        return 'BigEndian'

    if little and not big:
        return 'LittleEndian'

    msg = 'big and little endian specified, but options are exclusive'
    raise ValueError(msg)

class hashio:
    """Read stream, and calculate running hash

    The sha256 of the file is used as a global, unique identifier, but has to
    be computed. Since scanning for geometry is necessary anyway, and not
    really parallelisable, so it is a good time to also compute the hash.

    Proxies the io.IOBase family of interfaces, and disallows seeking
    backwards. Every read and seek operation is intercepted to compute the hash
    value, but downstream users not made aware of this.
    """
    def __init__(self, stream):
        self.stream = stream
        self.sha256 = hashlib.sha256()

    def read(self, *args):
        chunk = self.stream.read(*args)
        self.sha256.update(chunk)
        return chunk

    def seek(self, offset, whence = io.SEEK_SET):
        if whence != io.SEEK_CUR:
            raise NotImplementedError
        _ = self.read(offset)

    def hexdigest(self):
        return self.sha256.hexdigest()

from .segmenter import segmenter

def scan(args, stream):
    """Scan a file and create an index

    Scan a stream, and produce an index for building a job schedule in further
    ingestion.

    The output is compatible with what openvds' SEGYScan outputs, except the
    persistentID does not describe the run, but rather the input file (sha256).

    Parameters
    ----------
    args
        for expected members, see main in __main__.py
    stream :
        io.IOBase compatible interface

    Returns
    -------
    d : dict
        openvds compatible dict describing the structure of the file
    """
    primary = args.primary_word or word_numerals[args.primary]
    secondary = args.secondary_word or word_numerals[args.secondary]

    endianness = resolve_endianness(args.big_endian, args.little_endian)
    out = {
        'primaryKey': [primary, word_widths[primary]],
        'secondaryKey': [secondary, word_widths[secondary]],
        'headerEndianness': endianness,
    }

    # openvds uses the values Little/BigEndian, but all python functions take
    # little and big
    endian = {
        'LittleEndian': 'little',
        'BigEndian': 'big',
    }[endianness]

    stream = hashio(stream)
    stream.seek(textheader_size, whence = io.SEEK_CUR)

    out.update(scan_binary(stream, endian))

    chunk = stream.read(header_size)
    header = segyio.field.Field(buf = chunk, kind = 'trace')

    out.update(updated_count_interval(header, out, endian))
    # convert to milliseconds
    out['sampleInterval'] /= 1000.0
    tracelen = out['sampleCount'] * format_size[out['dataSampleFormatCode']]

    seg = segmenter(
            primary = primary,
            secondary = secondary,
            endian = endian,
        )

    seg.add(header)
    stream.seek(tracelen, whence = io.SEEK_CUR)

    while True:
        chunk = stream.read(header_size)
        if len(chunk) == 0:
            seg.commit()
            break

        if len(chunk) != header_size:
            msg = 'file truncated at trace {}'.format(trace_count)
            raise RuntimeError(msg)

        header = segyio.field.Field(buf = chunk, kind = 'trace')
        seg.add(header)
        stream.seek(tracelen, whence = io.SEEK_CUR)

    out['persistentID'] = stream.hexdigest()
    out['segmentInfo'] = seg.segments
    out['traceCount'] = seg.traceno
    return out
