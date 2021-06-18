import collections
import functools
import numpy as np
import requests
import msgpack
import time
import xarray

class assembler:
    """Base for the assembler
    """
    kind = 'untyped'

    def __init__(self, src):
        self.sourcecube = src

    def __repr__(self):
        return self.kind

    def numpy(self, unpacked):
        """Assemble numpy array

        Assemble a numpy array from a parsed response.

        Parameters
        ----------
        unpacked
            The result of msgpack.unpackb(slice.get())
        Returns
        -------
        a : numpy.array
            The result as a numpy array
        """
        raise NotImplementedError

    def xarray(self, unpacked):
        """Assemble xarray

        Assemble an xarray from a parsed response.

        Parameters
        ----------
        unpacked
            The result of msgpack.unpackb(slice.get())

        Returns
        -------
        xa : xarray.DataArray
            The result as an xarray
        """
        raise NotImplementedError

class assembler_slice(assembler):
    kind = 'slice'

    def __init__(self, sourcecube, dimlabels, name):
        super().__init__(sourcecube)
        self.dims = dimlabels
        self.name = name

    def numpy(self, unpacked):
        index = unpacked[0]['index']
        dims0 = len(index[0])
        dims1 = len(index[1])

        result = np.zeros((dims0 * dims1), dtype = np.single)
        for bundle in unpacked[1]:
            for tile in bundle['tiles']:
                layout = tile
                dst = layout['initial-skip']
                chunk_size = layout['chunk-size']
                src = 0
                v = tile['v']
                for _ in range(layout['iterations']):
                    result[dst : dst + chunk_size] = v[src : src + chunk_size]
                    src += layout['substride']
                    dst += layout['superstride']

        return result.reshape((dims0, dims1))

    def xarray(self, unpacked):
        index = unpacked[0]['index']
        a = self.numpy(unpacked)
        # TODO: add units for time/depth
        return xarray.DataArray(
            data   = a,
            dims   = self.dims,
            name   = self.name,
            coords = index,
        )

class assembler_curtain(assembler):
    kind = 'curtain'

    def numpy(self, unpacked):
        # This function is very rough and does suggest that the message from the
        # server should be richer, to more easily allocate and construct a curtain
        # object
        header = unpacked[0]
        shape = header['shape']
        index = header['index']
        dims0 = len(index[0])
        dimsz = len(index[2])

        # pre-compute where to put traces based on the dim0/dim1 coordinates
        # note that the index is made up of zero-indexed coordinates in the volume,
        # not the actual line numbers
        xyindex = { (x, y): i for i, (x, y) in enumerate(zip(index[0], index[1])) }

        # allocate the result. The shape can be slightly larger than dims0 * dimsz
        # since the traces can be padded at the end. By allocating space for the
        # padded traces we can just put floats directly into the array
        xs = np.zeros(shape = shape, dtype = np.single)

        for bundle in unpacked[1]:
            for part in bundle['traces']:
                x, y, z = part['coordinates']
                v = part['v']
                xs[xyindex[(x, y)], z:z+len(v)] = v[:]

        return xs[:dims0, :dimsz]

    def xarray(self, unpacked):
        index = unpacked[0]['index']
        a = self.numpy(unpacked)
        ijk = self.sourcecube.ijk

        xs = [ijk[0][x] for x in index[0]]
        ys = [ijk[1][x] for x in index[1]]
        # TODO: address this inconsistency - zs is in 'real' sample offsets,
        # while xs/ys are cube indexed
        zs = index[2]
        da = xarray.DataArray(
            data = a,
            name = 'curtain',
            # TODO: derive labels from query, header, or manifest
            dims = ['xy', 'z'],
            coords = {
                'x': ('xy', xs),
                'y': ('xy', ys),
                'z': zs,
            }
        )

        return da

class cube:
    """ Cube handle

    Constructing a cube object does not trigger any http calls as all properties
    are fetched lazily.
    """
    def __init__(self, guid, session):
        self.session = session
        self.guid = guid
        self._shape = None
        self._ijk = None

    @property
    def shape(self):
        """ Shape of the cube

        N-element int-tuple.

        Notes
        -----
        The shape is immutable and the result may be cached.
        """
        if self._shape is not None:
            return self._shape

        self._shape = tuple(len(dim) for dim in self.ijk)
        return self._shape

    @property
    def ijk(self):
        """
        Notes
        -----
        The ijk is immutable and the result may be cached.

        The ijk name is temporary and will change without notice
        """
        if self._ijk is not None:
            return self._ijk

        resource = f'query/{self.guid}'
        r = self.session.get(resource)
        self._ijk = [
            [x for x in dim['keys']] for dim in r.json()['dimensions']
        ]
        return self._ijk

    def slice(self, dim, lineno):
        """ Fetch a slice

        Parameters
        ----------

        dim : int
            The dimension along which to slice
        lineno : int
            The line number we would like to fetch. This corresponds to the
            axis labels given in the dim<n> members. In order to fetch the nth
            surface allong the mth dimension use lineno = dim<m>[n].

        Returns
        -------

        slice : numpy.ndarray
        """
        resource = f"query/{self.guid}/slice/{dim}/{lineno}"
        # TODO: derive labels from query, header, or manifest
        labels = ['inline', 'crossline', 'time']
        name = f'{labels.pop(dim)} {lineno}'
        proc = schedule(
            session = self.session,
            resource = resource,
        )
        proc.assembler = assembler_slice(self, dimlabels = labels, name = name)
        return proc

    def curtain(self, intersections):
        """Fetch a curtain

        Parameters
        ----------

        Returns
        -------
        curtain : numpy.ndarray
        """

        resource = f'query/{self.guid}/curtain'
        body = {
            'intersections': intersections
        }
        import json
        proc = schedule(
            session = self.session,
            resource = resource,
            data = json.dumps(body),
        )

        proc.assembler = assembler_curtain(self)
        return proc

class process:
    """

    Maps conceptually to an observer of a process server-side. Comes with
    methods for querying status, completedness, and the final result.

    Parameters
    ----------
    host : str
        Hostname.
    session : request.Session
        A requests.Session-like with a get() method. Authorization headers
        should be set.
    pid : str
        The process id
    status_url : str
        Relative path to the status endpoint.
    result_url : str
        Relative path to the result endpoint.

    Notes
    -----
    Constructing a process manually is reserved for the implementation.

    See also
    --------
    schedule
    """
    def __init__(self, session, pid, status_url, result_url):
        self.session = session
        self.pid = pid
        self.assembler = None
        self.status_url = status_url
        self.result_url = result_url
        self.done = False

    def __repr__(self):
        return '\n\t'.join([
            'oneseismic.process',
                f'pid: {self.pid}',
                f'assembler: {repr(self.assembler)}'
        ])

    def status(self):
        """ Processs status

        Retuns
        ------
        status : str
            Returns one of { 'working', 'finished' }

        Notes
        -----
        This function simply returns what the server responds with, so code
        inspecting the status should always have a fall-through case, in case
        the server is updated and returns something new.
        """
        r = self.session.get(self.status_url)
        response = r.json()

        if r.status_code == 200:
            self.done = True
            return response['status']

        if r.status_code == 202:
            return response['status']

        raise AssertionError(f'Unhandled status code f{r.status_code}')

    def get_raw(self):
        """Get the unparsed response
        Get the raw response for the result. This function will block until the
        result is ready, and will start downloading data as soon as any is
        available.

        Returns
        -------
        reponse : bytes
            The (possibly cached) response
        """
        try:
            return self._cached_raw
        except AttributeError:
            stream = f'{self.result_url}/stream'
            r = self.session.get(stream)
            self._cached_raw = r.content
            return self._cached_raw

    def get(self):
        """Get the parsed response
        """
        return msgpack.unpackb(self.get_raw())

    def numpy(self):
        try:
            return self._cached_numpy
        except AttributeError:
            raw = self.get()
            self._cached_numpy = self.assembler.numpy(raw)
            return self._cached_numpy

    def xarray(self):
        try:
            return self._cached_xarray
        except AttributeError:
            raw = self.get()
            self._cached_xarray = self.assembler.xarray(raw)
            return self._cached_xarray

    def withcompression(self, kind = 'gz'):
        """Get response compressed if available

        Request that the response be sent compressed, if available.  Compressed
        responses are typically half the size of uncompressed responses, which
        can be faster if there is limited bandwidth to oneseismic. Compressed
        responses are typically not faster inside the data centre.

        If kind is None, compression will be disabled.

        Compression defaults to 'gz'.

        Parameters
        ----------
        kind : { 'gz', None }, optional
            Compression algorithm. Defaults to gz.

        Returns
        -------
        self : process

        Examples
        --------
        Read a compressed slice:
        >>> proc = cube.slice(dim = 0, lineno = 5024)
        >>> proc.withcompression(kind = 'gz')
        >>> s = proc.numpy()
        >>> proc = cube.slice(dim = 0, lineno = 5).withcompression(kind = 'gz')
        >>> s = proc.numpy()
        """
        self.session.withcompression(kind)
        return self

    def withgz(self):
        """process.withcompression(kind = 'gz')
        """
        return self.withcompression(kind = 'gz')

def schedule(session, resource, data = None):
    """Start a server-side process.

    This function centralises setting up a HTTP session and building the
    process object, whereas end-users should use methods on the outermost cube
    class.

    Parameters
    ----------
    session : requests.Session
        Session object with a get() for making http requests
    resource : str
        Resource to schedule, e.g. 'query/<id>/slice'

    Returns
    -------
    proc : process
        Process handle for monitoring status and getting the result

    Notes
    -----
    Scheduling a process manually is reserved for the implementation.
    """
    r = session.get(resource, data = data)

    body = r.json()
    auth = 'Bearer {}'.format(body['authorization'])
    s = http_session(session.base_url)
    s.headers.update({'Authorization': auth})

    pid = body['location'].split('/')[-1]
    return process(
        session = s,
        pid = pid,
        status_url = body['status'],
        result_url = body['location'],
    )

class http_session(requests.Session):
    """
    http_session provides some automation on top of the requests.Session type,
    to simplify http requests in more seismic-specific interfaces and logic.
    Methods also raise non-200 http status codes as exceptions.

    The http_session methods do not take absolute URLs, but relative URLs e.g.
    req.get(url = 'result/<pid>/status').

    Parameters
    ----------
    base_url : str
        The base url, schema + host, for the oneseismic service
    auth :
        Object to request up-to-date authorization headers from

    Notes
    -----
    This class is meant for internal use, to provide a clean boundary for
    low-level network-oriented code.
    """
    def __init__(self, base_url, tokens = None, params = {}, *args, **kwargs):
        self.base_url = base_url
        self.tokens = tokens
        super().__init__(*args, **kwargs)
        self.params.update(params)

    def merge_auth_headers(self, kwargs):
        if self.tokens is None:
            return kwargs

        headers = self.tokens.headers()
        if 'headers' in kwargs:
            # unpack-and-set rather than just assigning the dictionary, in case
            # headers() starts returning more than just the Authorization
            # headers. This puts the power of definition where it belongs, and
            # keeps http_session oblivious to oneseismic specific header
            # expectations.
            #
            # If users at call-time explicitly set any of these headers,
            # respect them
            for k, v in headers.items():
                kwargs['headers'].setdefault(k, v)
        else:
            kwargs['headers'] = headers

        return kwargs

    def get(self, url, *args, **kwargs):
        """HTTP GET

        requests.Session.get, but raises exception for non-200 HTTP status
        codes. Authorization headers will be added to the request if
        http_session.tokens is available.

        This function will respect call-level custom headers, and only use
        http_session.tokens.headers() if not specified, similar to the requests
        API [1]_.

        Parameters
        ----------
        url : str
            Relative url to the resource, e.g. 'result/<pid>/status'

        Returns
        -------
        r : request.Response

        See also
        --------
        requests.get

        References
        ----------
        .. [1] https://requests.readthedocs.io/en/master/user/advanced/#session-objects

        Examples
        --------
        Defaulted and custom authorization:
        >>> session = http_session(url, tokens = tokens)
        >>> session.get('/needs-auth')
        >>> session.get('/needs-auth', headers = { 'Accept': 'text/html' })
        >>> session.get('/no-auth', headers = { 'Authorization': None })
        """
        kwargs = self.merge_auth_headers(kwargs)
        r = super().get(
            f'{self.base_url}/{url}',
            *args, **kwargs
        )
        r.raise_for_status()
        return r

    def withcompression(self, kind):
        """Get response compressed if available

        Request that the response be sent compressed, if available.  Compressed
        responses are typically half the size of uncompressed responses, which
        can be faster if there is limited bandwidth to oneseismic. Compressed
        responses are typically not faster inside the data centre.

        If kind is None, compression will be disabled.

        Parameters
        ----------
        kind : { 'gz', None }
            Compression algorithm

        Returns
        -------
        self : http_session

        Notes
        -----
        This function does not accept defaults, and the http_session does not
        have withgz() or similar methods, since it is a lower-level class and
        not built for end-users.
        """
        if kind is None:
            self.params.pop('compression', None)
            return self

        kinds = ['gz']
        if kind not in kinds:
            msg = f'compression {kind} not one of {",".join(kinds)}'
            raise ValueError(msg)
        self.params['compression'] = kind
        return self

    @staticmethod
    def fromconfig(cache_dir = None):
        """Create a new session from on-disk config

        Create a new http_sesssion with parameters and auth read from disk.
        This is a convenient constructor for most programs and uses outside of
        testing.

        Parameters
        ----------
        cache_dir : path or str, optional
            Configuration cache directory

        Returns
        -------
        session : http_session
            A ready-to-use http_session with authorization headers set
        """
        from ..login.login import config, tokens
        cfg = config(cache_dir = cache_dir).load()
        auth = tokens(cache_dir = cache_dir).load(cfg)
        return http_session(base_url = cfg['url'], tokens = auth)

    @staticmethod
    def from_sas_token(sas_token, base_url):
        """Create a new session from a SAS token

        Parameters
        ----------
        sas_token : str
        base_url : str

        Returns
        -------
        session : http_session
            A ready-to-use http_session with authorization headers set
        """
        from urllib.parse import parse_qs
        sas = parse_qs(sas_token)

        return http_session(base_url=base_url, params=sas)

def ls(session):
    """List available cubes

    List the cubes stored in oneseismic. The ids returned should all be valid
    arguments for the oneseismic.client.cube class.

    Parameters
    ----------
    session : oneseismic.http_session
        Session with authorization headers set

    Returns
    -------
    guids : iterable of str
        Cube GUIDs

    See also
    --------
    oneseismic.client.cube
    """
    return session.get('query').json()['links'].keys()

class cubes(collections.abc.Mapping):
    """Dict-like interface to cubes in the oneseismic subscription

    Parameters
    ----------
    session : http_session
    """
    def __init__(self, session):
        self.session = session
        self.cache = None

    def __getitem__(self, guid):
        if guid not in self.guids:
            raise KeyError(guid)
        return cube(guid, self.session)

    def __iter__(self):
        yield from self.guids

    def __len__(self):
        return len(self.guids)

    def sync(self):
        """Synchronize the set of guids in the subscription.

        It is generally only necessary to call this function once, but it can
        be called manually to get new IDs that have been added to the
        subscription since the client was created. For programs, it is
        Generally a better idea to create a new client.

        This is intended for internal use.
        """
        self.cache = ls(self.session)

    @property
    def guids(self):
        """Guids of cubes in subscription

        This is for internal use.

        All other functions should use this property to interact with guids, as
        it manages the cache.
        """
        if self.cache is None:
            self.sync()
        return self.cache

class cli:
    """User friendly access to oneseismic

    Access oneseismic services in a user-friendly manner with the cli class,
    suitable for programs, REPLs, and notebooks.

    Parameters
    ----------
    session : http_session
    """
    def __init__(self, session):
        self.session = session

    @property
    def cubes(self):
        return cubes(self.session)
