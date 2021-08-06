import collections
import io
import json
import math
import numpy as np
import segyio
import segyio._segyio
import pathlib

from segyio.tools import native

def splitarray(a, n):
    """Split array into chunks of size N

    Split the array into chunks of size N. The last chunk may be smaller.

    Parameters
    ----------
    a : array_like
    n : int

    Yields
    ------
    a : array_like

    Examples
    --------
    >>> list(splitarray([1, 2, 3, 4, 5], 2))
    [[1, 2], [3, 4], [5]]
    >>> list(splitarray([1, 2, 3, 4, 5], 3))
    [[1, 2, 3], [4, 5]]
    >>> list(splitarray([1, 2, 3, 4, 5, 6], 3))
    [[1, 2, 3], [4, 5, 6]]
    """
    a = np.asarray(a)
    for i in range(0, len(a), n):
        yield a[i:i+n]

class fileset:
    """Files of a volume

    The fileset class implements a simple virtual write-only file system. It
    internally handles the addresing from line numbers to fragment IDs.

    A "file" in this sense is just a numpy array (as a buffer) that can be
    uploaded as-is as a fragment. The parameters are generally obtained by
    parsing the output of the oneseismic scan program.

    Parameters
    ----------
    key1s : list of int
    key2s : list of int
    key3s : list of int
    shape : tuple of int

    Notes
    -----
    Right now, oneseismic is not mature enough to properly handle "holes" in
    the volumes, and require explicit padding.  The fileset class is aware of
    this and will generate padding fragments when necessary. This may change in
    the future and should not be relied on.
    """
    def __init__(self, key1s, key2s, key3s, shape, prefix):
        mkfile = lambda: np.zeros(shape = shape, dtype = np.float32)
        # fileset is built on the files dict, which is a mapping from (i,j,k)
        # fragment IDs to arrays. When a new fragment is accessed, a full
        # zero-cube will be created, so there is no need to keep track of what
        # needs padding.
        self.files = collections.defaultdict(mkfile)
        shapestr = '-'.join(map(str, shape))
        # normalize, e.g. remove repeated slashes, make all forward slash etc
        self.prefix = (pathlib.PurePath(prefix) / shapestr).as_posix()
        self.ext = 'f32'

        # map line-numbers to the fragment ID
        self.key1s = {k: i // shape[0] for i, k in enumerate(key1s)}
        self.key2s = {k: i // shape[1] for i, k in enumerate(key2s)}

        # map line-numbers to its index/offset/position in the fragment
        self.off1s = {k: i % shape[0] for i, k in enumerate(key1s)}
        self.off2s = {k: i % shape[1] for i, k in enumerate(key2s)}

        # samples/depth is implicitly indexed when read, so store how many
        # fragments there are in height
        self.zsections = math.ceil(len(key3s) / shape[2])
        self.shape = shape
        self.traceno = 0
        self.limits = {}

    def manifest_entry(self):
        """Return manifest key and entry

        Return the key and entry to insert into the manifest. The fileset knows
        how to issue a good entry. This should be implemented in derived
        classes, as the entry is surely different.
        """
        return 'data', {
            'file-extension': self.ext,
            'filters': [],
            'shapes': [list(self.shape)],
            'prefix': self.prefix,
            'resolution': 'source',
        }

    def commit(self, key1):
        """
        Commit a "lane", yielding all completed. This function assumes that the
        last trace in the lane has been given to put(). The lane are all files
        with the same i (i,j,k) fragment ID as the one computed from key1.

        Commit can and should be called once for every put().

        Parameters
        ----------
        key1 : int
            line-number of any dimension 0 line in the lane to commit

        Returns
        -------
        files : generator of (guid, np.array)

        Notes
        -----
        Commit will prune files whenever it can, to keep resource usage low,
        but can in pathological cases end up storing the original volume in
        memory. Calling commit() on the same lane twice will result in the
        second call giving all zero-padded files.
        """
        index1 = self.key1s[key1]
        if self.limits[index1] == self.traceno:
            # todo: cache.
            js = set(self.key2s.values())
            ks = set(range(self.zsections))
            i = index1

            # since self.files is a defaultdict, which means that even though an
            # (i,j,k) has never been written to before, it is generated and yielded
            # by the generator. This ensures that fragments or even holes [1] with
            # no data still get explicit representation when uploaded to
            # oneseismic.
            # [1] this happens quite often, especially at the edges of surveys
            for j in js:
                for k in ks:
                    ident = (i, j, k)
                    yield ident, self.files[ident]
                    del self.files[ident]

        self.traceno += 1

    def extract(self, trace):
        """Extract interesting data from a trace

        This function should be overriden or specialised if you need something
        other than the trace data (samples). For example, when using this for
        extracting attributes, it should read (and scale) the appropriate
        header word.

        extract() returns an array_like, i.e. scalar values must be wrapped in
        a list.

        extracted : array_like
        """
        return trace['samples']

    def put(self, key1, key2, trace):
        """Put a trace

        Put a trace read from the input into the fileset. This function assumes
        the trace is parsed to native floats. This function should be called
        once for every trace in the input.

        Parameters
        ----------
        key1 : int
            line-no in dimension 0
        key2 : int
            line-no in dimension 1
        trace : np.array of float
        """
        # determine what file this trace goes into
        # this is the (i, j, _) fragment ID
        index1 = self.key1s[key1]
        index2 = self.key2s[key2]

        # the offset in a specific file the trace starts at
        i = self.off1s[key1]
        j = self.off2s[key2]

        values = self.extract(trace)
        for index3, tr in enumerate(splitarray(values, self.shape[2])):
            fragment = self.files[(index1, index2, index3)]
            fragment[i, j, :len(tr)] = tr

    def setlimits(self, last_trace):
        """
        fileset needs to know when the last trace of a lane is read, to safely
        commit (and prune) files. This means figuring out what traceno is that last
        for every lane, which in theory could be any line.

        Returns a dict of all dimension 0 fragment IDs, and the last trace number
        in that line.

        Parameters
        ----------
        last_trace : dict
            A dict of { lineno: traceno }, as read from scan 'key1-last-trace'
        """
        limits = collections.defaultdict(int)
        for k, v in last_trace.items():
            x = self.key1s[int(k)]
            limits[x] = max(int(v), limits[x])

        self.limits.update(limits)


class cdpset(fileset):
    def __init__(self, word, key, *args, **kwargs):
        self.type = f'{key}'
        self.word = word
        super().__init__(*args, **kwargs)

    def extract(self, trace):
        header = segyio.field.Field(buf = trace['header'], kind = 'trace')
        scale = header[segyio.su.scalco]
        # SEG-Y specifies that a scaling of 0 should be interpreted as identity
        if scale == 0:
            scale = 1

        cdp = header[self.word]
        if scale > 0:
            return [cdp * scale]
        else:
            return [cdp / -scale]

    def manifest_entry(self):
        return 'attributes', {
            # the type must match exactly with the prefix in storage
            'type': f'cdp{self.type}',
            'layout': 'tiled',
            'file-extension': self.ext,
            'labels': [f'cdp {self.type}'.upper()],
            'shapes': [self.shape],
            'prefix': self.prefix,
        },

def upload(manifest, shape, src, origfname, filesys):
    """Upload volume to oneseismic

    Parameters
    ----------
    manifest : dict
        The parsed output of the scan program
    shape : tuple of int
        The shape of the fragment, typically (64, 64, 64) (adds up to 1mb)
    src : io.BaseIO
    fname : Path or str
        The original filename of the SEG-Y being ingested
    blob : azure.storage.blob.BlobServiceClient
    """
    word1 = manifest['key-words'][0]
    word2 = manifest['key-words'][1]
    key1s = manifest['dimensions'][0]
    key2s = manifest['dimensions'][1]
    key3s = manifest['dimensions'][2]
    guid  = manifest['guid']

    key1s.sort()
    key2s.sort()
    key3s.sort()

    # Seek past the textual headers and the binary header
    src.seek(int(manifest['byteoffset-first-trace']), io.SEEK_CUR)

    # Make a custom dtype that corresponds to a header and a trace. This
    # assumes all traces are of same length and sampled similarly, which is
    # a safe assumption in practice. This won't be checked though, the check
    # belongs in the scan program.
    #
    # The dtype is quite useful because it means the input can be read into the
    # numpy array as a buffer, and then passed on directly as numpy arrays to
    # put()
    dtype = np.dtype([
        ('header', 'b', 240),
        ('samples', 'f4', len(key3s)),
    ])
    trace = np.array(1, dtype = dtype)
    fmt = manifest['format']

    files = [fileset(key1s, key2s, key3s, shape, prefix = 'src')]
    files.extend([
        cdpset(
            segyio.su.cdpx,
            'x',
            key1s,
            key2s,
            {1},
            shape = (512, 512, 1),
            prefix = 'attributes/cdpx',
        ),
        cdpset(
            segyio.su.cdpy,
            'y',
            key1s,
            key2s,
            {1},
            shape = (512, 512, 1),
            prefix = 'attributes/cdpy',
        ),
    ])

    for fset in files:
        fset.setlimits(manifest['key1-last-trace'])

    filesys.mkdir(guid)
    filesys.cd(guid)

    while True:
        n = src.readinto(trace)
        if n == 0:
            break

        header = segyio.field.Field(buf = trace['header'], kind = 'trace')
        data = native(data = trace['samples'], format = fmt)

        key1 = header[word1]
        key2 = header[word2]
        for fset in files:
            fset.put(key1, key2, trace)

        for fset in files:
            for ident, block in fset.commit(key1):
                ident = '-'.join(map(str, ident))
                name = f'{fset.prefix}/{ident}.{fset.ext}'
                print('uploading', name)

                with filesys.open(name, mode = 'wb') as f:
                    f.write(block)

    sampleinterval = key3s[1] - key3s[0]
    if sampleinterval >= 500:
        zdomain = 'time'
    else:
        zdomain = 'depth'
    print(f'sample-interval is {sampleinterval}, guessing {zdomain} domain')

    fname = pathlib.Path(origfname).name
    manifest = {
        'format-version': 1,
        'upload-filename': fname,
        'guid': guid,
        'data': [],
        'attributes': [],
        'line-numbers': [key1s, key2s, key3s],
        'line-labels': ['inline', 'crossline', zdomain],
    }

    for fset in files:
        key, entry = fset.manifest_entry()
        manifest[key].append(entry)

    with filesys.open('manifest.json', mode = 'wb') as f:
        f.write(json.dumps(manifest).encode())
