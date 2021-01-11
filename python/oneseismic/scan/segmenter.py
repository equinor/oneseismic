import numpy as np
import segyio

from .scan import parseint
from .scan import format_size
from .scan import binary_size

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

    def scan_binary(self, stream):
        """Read info from the binary header

        Read the necessary information from the binary header, and, if necessary,
        seek past the extended textual headers.

        Parameters
        ----------
        stream : stream_like
            an open io.IOBase like stream

        Notes
        -----
        If this function succeeds, stream will have read past the binary header and
        the extended text headers.

        If an exception is raised in this function, the stream is left at an
        unspecified position, and must be seeked or reset to behave well defined.
        """
        chunk = stream.read(binary_size)
        binary = segyio.field.Field(buf = chunk, kind = 'binary')

        # skip extra textual headers
        exth = self.intp.parse(binary[segyio.su.exth], length = 2)
        for _ in range(exth):
            stream.seek(textheader_size, whence = io.SEEK_CUR)

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
            'byteoffset-first-trace': 3600 + exth * 3200,
        })

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
        return self.observed

    def add(self, header):
        """Add a new header to the index

        This is mandatory to implement for scanners
        """
        raise NotImplementedError

class segmenter(scanner):
    """Scan, check, and report geometry

    Scan and report on the volume geometry, and verify that the file satisfies
    oneseismic's requirements for regularity and structure.
    """
    def __init__(self, primary, secondary, endian):
        super().__init__(endian)
        self.primary = primary
        self.secondary = secondary
        self.primaries = []
        self.secondaries = []
        self.previous_secondary = None
        self.traceno = 0

    def report(self):
        """Report the result

        This function should be called after the full file has been scanned.

        The report contains these keys:
        + byteorder
        + format
        + samples
        + sampleinterval
        + byteoffset-first-trace
        + dimensions
        """
        r = dict(super().report())
        interval = r['sampleinterval']
        samples = map(int, np.arange(0, r['samples'] * interval, interval))
        r['dimensions'] = [
            self.primaries,
            self.secondaries,
            list(samples),
        ]
        return r

    def add(self, header):
        """Add a record

        This function should be called once for every header, see the examples
        for intended use.

        This function assumes the primary key only changes on new segments
        (usually inlines), and that a primary key is never repeated outside its
        segment. Furthermore, it assumes that the secondary key follows the same
        pattern in all segments.

        This function assumes the header keys are integers in big-endian.

        Parameters
        ----------
        header : dict_like
            A dict like header, like the trace header object from segyio

        Examples
        --------
        Intended use:
        >>> seg = segmenter(...)
        >>> for header in headers:
        ...     seg.add(header)
        """

        key1 = self.intp.parse(header[self.primary])
        key2 = self.intp.parse(header[self.secondary])

        def increment():
            self.traceno += 1
            self.previous_secondary = key2

        # This is the first trace
        if not self.primaries and not self.secondaries:
            self.scan_first_header(header)
            self.primaries = [key1]
            self.secondaries = [key2]
            increment()
            return

        # The secondary wraps
        if key2 == self.secondaries[0]:

            if self.previous_secondary != self.secondaries[-1]:
                msg = "The secondary key wrapped before reaching end of line "\
                      "at trace: {} (primary: {}, secondary: {})."
                raise RuntimeError(msg.format(self.traceno, key1, key2))

            self.primaries.append(key1)
            increment()
            return

        # We should be encountering a new secondary
        if self.previous_secondary == self.secondaries[-1]:

            if key1 in self.primaries and key2 in self.secondaries:
                msg = "Duplicate (primary, secondary) pair detected at "\
                      "trace: {} (primary: {}, secondary: {})."
                raise RuntimeError(msg.format(self.traceno, key1, key2))

            if key2 in self.secondaries:
                msg = "Unexpected secondary key encountered at "\
                      "trace: {} (primary: {}, secondary: {})."
                raise RuntimeError(msg.format(self.traceno, key1, key2))

            if key1 != self.primaries[-1]:
                msg = "Primary key increment is expected to be accompanied by "\
                      "wrapping of secondary key. At trace: {} (primary: {}, "\
                      "secondary: {})."
                raise RuntimeError(msg.format(self.traceno, key1, key2))

            if len(self.primaries) > 1:
                msg = "Encountered a unseen secondary key after the first "\
                      "line at trace: {} (primary: {}, secondary: {})."
                raise RuntimeError(msg.format(self.traceno, key1, key2))

            self.secondaries.append(key2)
            increment()
            return

        # By now we know that we are encountering a known secondary that does
        # not wrap. Check that it follows the pattern of previous segments
        key2_index = self.secondaries.index(key2)
        if  self.secondaries[key2_index - 1] != self.previous_secondary:
            msg = "Encountered secondary key pattern deviating from that of "\
                  "previous lines at trace: {} (primary: {}, secondary: {})."
            raise RuntimeError(msg.format(self.traceno, key1, key2))
        increment()
