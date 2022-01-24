import math
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


class GeoScanner(Scanner):
    """ Find the mapping parameters from UTM coordinates to line numbers

    This scanner finds the 3 points in the cube closest to the 3 corners
    (il_min, xl_min), (il_max, xl_min) and (il_min, xl_max). This is done
    because a spread in the line numbers gave considerably better precision
    than taking adjacent traces. Since the actual corner traces might be missing
    from the file, we find the nearest.

    These points are used to compute the UTM-to-lineno affine matrix:

        a b c     X     inline
        k m n  *  Y  =  crossline
                  1

    Note that this assumes uniform inline and crossline positional increments.

    Attributes
    ----------
        p0, p1, p2
            UTM coordinates of points at 3 "corners" of system, with p0
            being the point of trace0 and p1/p2 far enough from it to
            achieve good calculation precision.
        p0_linenos, p1_linenos, p2_linenos
            iline/xline coordinates of corresponding points

    """
    def __init__(self, il_word, xl_word, endian):
        super().__init__(endian)
        self.xl_word = xl_word
        self.il_word = il_word

        self.iline_min = math.inf
        self.iline_max = -math.inf
        self.xline_min = math.inf
        self.xline_max = -math.inf

        self.p0_linenos = None
        self.p1_linenos = None
        self.p2_linenos = None

        self.p0 = None
        self.p1 = None
        self.p2 = None

    @staticmethod
    def is_closer(target, point, other):
        def manhattan(x, y): return sum(map(lambda i, j: abs(i - j), x, y))

        return manhattan(other, target) > manhattan(point, target)

    @staticmethod
    def get_cdp(header, word):
        scale = header[segyio.su.scalco]
        # SEG-Y specifies that a scaling of 0 should be interpreted as identity
        if scale == 0:
            scale = 1

        cdp = header[word]
        if scale > 0:
            return cdp * scale
        else:
            return cdp / -scale

    def scan_trace_header(self, header):
        iline = self.intp.parse(header[self.il_word])
        xline = self.intp.parse(header[self.xl_word])
        cdpx = self.get_cdp(header, segyio.su.cdpx)
        cdpy = self.get_cdp(header, segyio.su.cdpy)

        if self.iline_min > iline:
            self.iline_min = iline

        if self.iline_max < iline:
            self.iline_max = iline

        if self.xline_min > xline:
            self.xline_min = xline

        if self.xline_max < xline:
            self.xline_max = xline

        closer_to_origo = \
            self.p0_linenos is not None and \
            self.is_closer(
                (self.iline_min, self.xline_min),
                (iline, xline),
                self.p0_linenos
            )

        if self.p0_linenos is None or closer_to_origo:
            self.p0_linenos = (iline, xline)
            self.p0 = (cdpx, cdpy)

        closer_to_il_corner = \
            self.p1_linenos is not None and \
            self.is_closer(
                (self.iline_max, self.xline_min),
                (iline, xline),
                self.p1_linenos
            )

        if self.p1_linenos is None or closer_to_il_corner:
            self.p1_linenos = (iline, xline)
            self.p1 = (cdpx, cdpy)

        closer_to_xl_corner = \
            self.p2_linenos is not None and \
            self.is_closer(
                (self.iline_min, self.xline_max),
                (iline, xline),
                self.p2_linenos
            )

        if self.p2_linenos is None or closer_to_xl_corner:
            self.p2_linenos = (iline, xline)
            self.p2 = (cdpx, cdpy)


    def report(self):
        # To represent a point UTM (x, y) as (xline, iline) coordinate only a
        # combination of linear transformations is needed inside one UTM zone
        # grid, because converting from one cartesian grid to another is always
        # a linear transformation.
        #
        # Thus a problem of changing coordinate system from UTM to iline/xline
        # coordinates presents a standard 2D affine transformation:
        #   a b c     p.x     p.inline
        #   k m n  x  p.y  =  p.xline
        #   0 0 1     1       1
        #
        # As we have 3 variables for each equation we need to obtain 3 points
        # with known coordinates in each of the coordinate systems. Then we can
        # solve the system with regards to coefficient matrix.
        #
        # With 3 known points system can be represented as
        #   a b c     p0.x  p1.x  p2.x     p0.iline  p1.iline  p2.iline
        #   k m n  x  p0.y  p1.y  p2.y  =  p0.xline  p1.xline  p2.xline
        #   0 0 1     1     1     1        1         1         1
        #
        # This can be transformed to the system M x = b, where
        #
        #       p0.x  p0.y  1       a k       p0.inline  p0.xline
        #   M = p1.x  p1.y  1   x = b m   b = p1.inline  p1.xline
        #       p2.x  p2.y  1 ,     c n,      p2.inline  p2.xline
        #
        # We solve this system using the inverse of M: x = M^-1 b

        M = np.array([
            [self.p0[0], self.p0[1], 1],
            [self.p1[0], self.p1[1], 1],
            [self.p2[0], self.p2[1], 1]
        ])

        try:
            utm_to_inline = np.linalg.inv(M).dot(
                np.array([
                    self.p0_linenos[0],
                    self.p1_linenos[0],
                    self.p2_linenos[0]
                ])
            )
            utm_to_crossline = np.linalg.inv(M).dot(
                np.array([
                    self.p0_linenos[1],
                    self.p1_linenos[1],
                    self.p2_linenos[1]
                ])
            )
        except np.linalg.LinAlgError:
            # This exception is raised if M is not invertible
            # TODO: Raise exception? Log?
            return {}

        # Check that the lineno-to-utm mappings are not zero. This will for
        # instance happen if the UTM coordinate headers are set to zero
        # (happens if they are unset).
        # TODO: Raise exception? Log?
        if (utm_to_inline == 0).all() or (utm_to_crossline == 0).all():
            return {}

        # We expect inlines and crosslines to be perpendicular to each other.
        # If that is the case, the vectors (a, b) and (k, m) will also be
        # perpendicular, a, b, k and m being coefficients on the UTM-to-lineno
        # affine matrix (see the comment at the beginning of function). We
        # normalize the result to distinguish between floating point errors and
        # small dot product due to small vectors
        dot_product = utm_to_inline[:-1].dot(utm_to_crossline[:-1])
        moduli_product = np.linalg.norm(utm_to_inline[:-1]) \
                       * np.linalg.norm(utm_to_crossline[:-1])
        if abs(dot_product / moduli_product) > 1e-4:
            # TODO: Raise exception? Log?
            return {}

        return {
            'utm-to-lineno': [utm_to_inline.tolist(), utm_to_crossline.tolist()]
        }
