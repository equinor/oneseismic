import logging
import warnings

import segyio

from .scan import parseint


class segmenter:
    def __init__(self, primary, secondary, endian):
        self.primary = primary
        self.secondary = secondary
        self.primaries = []
        self.secondaries = []
        self.previous_secondary = None
        self.traceno = 0
        self.intp = parseint(endian=endian, default_length=4)

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

        def increment():
            self.traceno += 1
            self.previous_secondary = key2

        # This is the first trace
        if not self.primaries and not self.secondaries:
            self.primaries = [key1]
            self.secondaries = [key2]
            increment()
            return

        # The secondary wraps
        if key2 == self.secondaries[0]:

            if key1 in self.primaries:
                raise RuntimeError("The file ...")

            if self.previous_secondary != self.secondaries[-1]:
                raise RuntimeError("The file ...")

            self.primaries.append(key1)
            increment()
            return

        # We are encountering a new secondary
        if self.previous_secondary == self.secondaries[-1]:

            if key2 in self.secondaries:
                raise RuntimeError("The file ...")

            if key1 != self.primaries[-1]:
                raise RuntimeError("The file ...")

            if len(self.primaries) > 1:
                raise RuntimeError("The file ...")

            self.secondaries.append(key2)
            increment()
            return

        # By now we know that we are encountering a known secondary that does
        # not wrap. Check that it follows the pattern of previous segments
        key2_index = self.secondaries.index(key2)
        if self.secondaries[key2_index - 1] != self.previous_secondary:
            raise RuntimeError("The file ...")
        increment()
