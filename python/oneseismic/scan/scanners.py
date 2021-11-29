import segyio

import numpy as np

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


class Scanner:
    """Base class and interface for scanners

    A scanner can be plugged into the scan.__main__ logic and scan.scan
    function as an action, which implements the indexing and reporting logic
    for a type of scan. All scans involve parsing the binary/trace headers - or
    in some cases the trace itself, and write some report.
    """
    def __init__(self, endian):
        self.endian = endian
        self.intp = parseint(endian = endian, default_length = 4)

    def scan_binary_header(self, binary):
        """Scan a SEG-Y binary header

        Parameters
        ----------
        binary : segyio.Field or dict_like
        """
        pass

    def scan_trace_header(self, header):
        """Scan a SEG-Y trace header

        Parameters
        ----------
        header : segyio.Field or dict_like
        """
        pass

    def scan_trace_samples(self, trace):
        """Scan a SEG-Y trace

        Parameters
        ----------
        trace : segyio.Trace or array_like
        """
        pass

    def report(self):
        """Report the result of a scan

        Returns
        -------
        report : dict
        """
        pass


class BasicScanner(Scanner):
    def __init__(self, endian):
        super().__init__(endian)
        self.observed = {}

    def scan_binary_header(self, binary):
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

    def report(self):
        return dict(self.observed)


class LineScanner(Scanner):
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
        self.samples = None
        self.interval = None

    def scan_binary_header(self, header):
        self.interval = self.intp.parse(
                header[segyio.su.hdt],
                length = 2
        )
        self.samples = self.intp.parse(
            header[segyio.su.hns],
            length = 2,
            signed = False,
        )

    def scan_trace_header(self, header):
        key1 = self.intp.parse(header[self.key1])
        key2 = self.intp.parse(header[self.key2])
        self.key1s.add(key1)
        self.key2s.add(key2)
        self.last1s[key1] = self.traceno
        self.traceno += 1
        # TODO: detect key-pair duplicates?
        # TODO: check that the sample-interval is consistent across the file?

    def report(self):
        key3s = map(int,
            np.arange(0, self.samples * self.interval, self.interval)
        )

        return {
            'key1-last-trace' : self.last1s,
            'key-words'       : [self.key1, self.key2],
            'dimensions'      : [
                sorted(self.key1s),
                sorted(self.key2s),
                list(key3s)
            ]
        }


class StatisticsScanner(Scanner):
    """Gather statistical information from the traces

    This information is not strictly necessary, but it can be quite handy
    for applications to have it easily accessible through the manifest.
    This method assumes that the trace is parsed as native floats.
    """
    def __init__(self, endian):
        super().__init__(endian)

        self.minsample = None
        self.maxsample = None

    def scan_trace_samples(self, trace):
        trmin = float(np.amin(trace))
        trmax = float(np.amax(trace))

        if self.minsample is None or trmin < self.minsample:
            self.minsample = trmin

        if self.maxsample is None or trmax > self.maxsample:
            self.maxsample = trmax

    def report(self):
        return {
            'sample-value-min' : self.minsample,
            'sample-value-max' : self.maxsample
        }

