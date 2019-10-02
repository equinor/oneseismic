import logging
import warnings
import segyio

from .scan import parseint

class segmenter:
    def __init__(self, primary, secondary, endian):
        self.primary = primary
        self.secondary = secondary
        self.segments = []
        self.current = None
        self.traceno = 0
        self.intp = parseint(endian = endian, default_length = 4)

    def add(self, header):
        """Add a record, possibly finalizing the current segment

        This function should be called once for every header, see the examples
        for intended use.

        This function assumes the primary key only changes on new segments
        (usually inlines), and that a primary key is never repeated outside its
        segment. When a new primary key is encountered, the current segment is
        finalized and commited, and the next segment begins. This makes it
        necessary to call commit() after the last header has been added.

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
        >>> seg.commit()
        """
        key1 = self.intp.parse(header[self.primary])
        key2 = self.intp.parse(header[self.secondary])
        cdpx = float(self.intp.parse(header[segyio.su.cdpx]))
        cdpy = float(self.intp.parse(header[segyio.su.cdpy]))

        if self.current is not None and key1 != self.current['primaryKey']:
            self.commit()

        if self.current is None:
            logging.debug('starting new segment from trace %', self.traceno)
            self.current = {
                'primaryKey': key1,
                'traceStart': self.traceno,
                'binInfoStart': {
                    'inlineNumber': key1,
                    'crosslineNumber': key2,
                    'ensembleXCoordinate': cdpx,
                    'ensembleYCoordinate': cdpy,
                },
                'binInfoStop': {
                    'inlineNumber': key1,
                    'crosslineNumber': key2,
                    'ensembleXCoordinate': cdpx,
                    'ensembleYCoordinate': cdpy,
                },
            }

        stop = self.current['binInfoStop']
        stop['inlineNumber'] = key1
        stop['crosslineNumber'] = key2
        self.traceno += 1

    def commit(self):
        """Commit the current segment

        Close and commit the current segment, and add it to the list of
        segments. This method *must* be called when the file is exhausted, to
        properly close the last segment.

        Warns
        -----
        UserWarning
            If commit is called without a current object, usually from a
            missing add()

        Notes
        -----
        add is aware of this function, and will commit and close all other
        segments than the last automatically from the headers.

        Examples
        --------
        Typical use:
        >>> seg = segmenter(...)
        >>> for header in headers:
        ...     seg.add(header)
        >>> seg.commit()
        """

        if self.current is None:
            msg = 'commit called without current object, did you forget add()?'
            warnings.warn(msg)
            return

        logging.debug('committing segment [%:%]',
            self.current['traceStart'],
            self.traceno,
        )
        # traceno has already been incremented, because a segment is closed and
        # committed *after* the next segment has been added, and scan is
        # expected to produce closed intervals
        self.current['traceStop'] = self.traceno - 1
        self.segments.append(self.current)
        self.current = None
