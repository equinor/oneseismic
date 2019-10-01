import segyio
import segyio._segyio

textheader_size = 3200
binary_size = 400

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
