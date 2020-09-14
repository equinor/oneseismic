from .scan import parseint

class segmenter:
    def __init__(self, primary, secondary, endian):
        self.primary = primary
        self.secondary = secondary
        self.primaries = []
        self.secondaries = []
        self.previous_secondary = None
        self.traceno = 0
        self.intp = parseint(endian = endian, default_length = 4)

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
