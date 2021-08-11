import collections
import functools
import numpy as np
import requests
import msgpack
import time
import xarray
import gql

from gql.transport.requests import RequestsHTTPTransport

from .. import decoder

def splitindex(ndims, index):
    shape = index[:ndims]
    index = index[ndims:]
    for k in shape:
        yield index[:k]
        index = index[k:]

class assembler:
    def __init__(self):
        self.decoder = decoder.decoder()

        self.numpys = {
            1: self.npslice,
            2: self.npcurtain,
        }

        self.xarrays = {
            1: self.xaslice,
            2: self.xacurtain,
        }

    def decnumpy(self, stream):
        self.decoder.reset()
        self.decoder.buffer_and_process(stream)

        head = self.decoder.header()
        ndims = head.ndims
        index = head.index
        function = head.function

        if function == decoder.functionid.slice:
            a = np.zeros(shape = index[:ndims], dtype = 'f4')
        elif function == decoder.functionid.curtain:
            a = np.zeros(shape = index[1:ndims], dtype = 'f4')
        else:
            raise RuntimeError('Bad message; unknown function')

        self.decoder.register_writer('data', a)
        self.decoder.process()
        return a

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
        header = unpacked[0]
        function = header['function']

        try:
            fn = self.numpys[function]
        except KeyError:
            raise RuntimeError(f'bad message; unknown function {function}')

        return fn(unpacked)

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
        header = unpacked[0]
        function = header['function']

        try:
            fn = self.xarrays[function]
        except KeyError:
            raise RuntimeError(f'bad message; unknown function {function}')

        return fn(unpacked)

    def npslice(self, unpacked):
        header = unpacked[0]
        ndims = header['ndims']
        shape = header['index'][:ndims]

        result = np.zeros(shape, dtype = np.single).ravel()
        for bundle in unpacked[1]:
            attribute = bundle[0]
            tiles = bundle[1]

            if attribute != 'data':
                continue

            for tile in tiles:
                iterations   = tile[0]
                chunk_size   = tile[1]
                initial_skip = tile[2]
                superstride  = tile[3]
                substride    = tile[4]
                v            = tile[5]
                v = np.frombuffer(v, dtype = 'f4')
                src = 0
                dst = initial_skip
                for _ in range(iterations):
                    result[dst : dst + chunk_size] = v[src : src + chunk_size]
                    dst += superstride
                    src += substride

        return result.reshape(shape)

    def xaslice(self, unpacked):
        a = self.numpy(unpacked)

        header = unpacked[0]
        ndims = header['ndims']
        index = header['index']
        shape = index[:ndims]

        attrs = {}
        for attr in unpacked[0]['attributes']:
            dtype = 'f4'
            if self.name.startswith('time'):
                attrlabels = self.dims
                attrshape = shape[:len(attrlabels)]
            else:
                attrlabels = self.dims[0]
                attrshape = shape[0]

            attrs[attr] = np.zeros(np.prod(attrshape), dtype = dtype).ravel()

        for bundle in unpacked[1]:
            attribute = bundle[0]
            tiles = bundle[1]

            if attribute not in attrs:
                continue

            result = attrs[attribute]
            for tile in tiles:
                iterations   = tile[0]
                chunk_size   = tile[1]
                initial_skip = tile[2]
                superstride  = tile[3]
                substride    = tile[4]
                v            = tile[5]
                v = np.frombuffer(v, dtype = 'f4')
                src = 0
                dst = initial_skip
                for _ in range(iterations):
                    result[dst : dst + chunk_size] = v[src : src + chunk_size]
                    dst += superstride
                    src += substride

        # TODO: add units for time/depth
        coords = {
            self.dims[0]: index[ndims:][:shape[0]],
            self.dims[1]: index[ndims:][shape[0]:]
        }

        for k, v in attrs.items():
            coords[k] = (attrlabels, v.reshape(attrshape))

        return xarray.DataArray(
            data   = a,
            dims   = self.dims,
            name   = self.name,
            coords = coords,
        )

    def npcurtain(self, unpacked):
        # This function is very rough and does suggest that the message from the
        # server should be richer, to more easily allocate and construct a curtain
        # object
        header = unpacked[0]
        ndims = header['ndims']
        index = header['index']
        shape = index[:ndims]
        index = [dim for dim in splitindex(ndims, index)]

        xs = np.zeros(shape = shape[1:], dtype = np.single)
        for bundle in unpacked[1]:
            attribute = bundle[0]
            if attribute != 'data':
                continue

            ntraces = bundle[1]
            major   = bundle[2]
            minor   = bundle[3]
            values  = bundle[4]
            values  = np.frombuffer(values, dtype = 'f4')
            for i in range(ntraces):
                ifst = major[i*2]
                ilst = major[i*2 + 1]
                zfst = minor[i*2]
                zlst = minor[i*2 + 1]

                dst = xs[ifst:ilst, zfst:zlst]
                v = values[:dst.size].reshape(dst.shape)
                values = values[v.size:]
                dst[:] = v[:]

        return xs

    def xacurtain(self, unpacked):
        a = self.numpy(unpacked)

        header = unpacked[0]
        index = header['index']
        ndims = header['ndims']
        index = [dim for dim in splitindex(ndims, index)]

        xs = index[0]
        ys = index[1]
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
    def __init__(self, guid, session, gclient):
        self.session = session
        self.guid = guid
        self._shape = None
        self._ijk = None
        self.gclient = gclient

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

        query = f'''
        {{
            cube(id: "{self.guid}") {{
                linenumbers
            }}
        }}
        '''
        q = gql.gql(query)
        res = self.gclient.execute(q)
        # TODO: should this be copied out of the gql structure?
        self._ijk = res['cube']['linenumbers']
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

        query = f'''
        query {{
            cube(id: "{self.guid}") {{
                sliceByLineno(dim: {dim}, lineno: {lineno}) {{
                    url
                    key
                }}
            }}
        }}
        '''

        proc = gschedule(
            self.gclient,
            self.session.base_url,
            query,
        )
        # TODO: derive labels from query or result
        labels = ['inline', 'crossline', 'time']
        name = f'{labels.pop(dim)} {lineno}'
        proc.decoder.dims = labels
        proc.decoder.name = name
        return proc

    def curtain(self, intersections):
        """Fetch a curtain

        Parameters
        ----------

        Returns
        -------
        curtain : numpy.ndarray
        """

        # Rendering intersections in the fstring works because the python list
        # *happens* to be formatted the same way as the graphql
        # list-of-list-of-ints. Simple enough for this demo, but this should be
        # significantly different with a new gql (the python library) version
        # and more a more sophisticated schema.
        query = f'''
        query {{
            cube(id: "{self.guid}") {{
                curtain(coords: {intersections}) {{
                    url
                    key
                }}
            }}
        }}
        '''
        proc = gschedule(
            self.gclient,
            self.session.base_url,
            query,
        )
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
        self.decoder = assembler()
        self.status_url = status_url
        self.result_url = result_url
        self.done = False

    def __repr__(self):
        return '\n\t'.join([
            'oneseismic.process',
                f'pid: {self.pid}',
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
            self._cached_numpy = self.decoder.decnumpy(self.get_raw())
            return self._cached_numpy

    def xarray(self):
        try:
            return self._cached_xarray
        except AttributeError:
            raw = self.get()
            self._cached_xarray = self.decoder.xarray(raw)
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

def gschedule(client, base_url, query):
    """Schedule a job with GraphQL

    This is the graphql version of schedule(), which eventually will become
    schedule().
    """
    q = gql.gql(query)
    res = client.execute(q)

    for promise in res['cube'].values():
        url = promise['url']
        key = promise['key']

    auth = f'Bearer {key}'
    pid = url.split('/')[-1]
    session = http_session(base_url)
    session.headers.update({'Authorization': auth})

    return process(
        session = session,
        pid = pid,
        status_url = f'{url}/status',
        result_url = url,
    )

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
    print('QUERY: ', resource)
    print('BODY: ', body)

    try:
        body = body['data']['cube']['slice']
    except KeyError:
        body = body['data']['cube']['curtain']

    auth = 'Bearer {}'.format(body['key'])
    s = http_session(session.base_url)
    s.headers.update({'Authorization': auth})

    pid = body['url'].split('/')[-1]
    return process(
        session = s,
        pid = pid,
        status_url = body['url'] + '/status',
        result_url = body['url'],
    )

class graphclient(gql.Client):
    def __init__(self, tokens = None, *args, **kwargs):
        self.tokens = tokens
        super().__init__(*args, **kwargs)

    def execute(self, query, *args, **kwargs):
        if self.tokens is not None:
            if self.transport.headers is None:
                self.transport.headers = self.tokens.headers()
            else:
                self.transport.headers.update(self.tokens.headers())
        return super().execute(query, *args, **kwargs)

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
    def __init__(self, base_url, tokens = None, *args, **kwargs):
        self.base_url = base_url
        self.tokens = tokens
        super().__init__(*args, **kwargs)

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
        r = super().get(f'{self.base_url}/{url}', *args, **kwargs)
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
    query = '''{
        cubes
    }
    '''
    q = gql.gql(query)
    res = session.execute(q)
    return res['cubes']

class cubes(collections.abc.Mapping):
    """Dict-like interface to cubes in the oneseismic subscription

    Parameters
    ----------
    session : http_session
    """
    def __init__(self, session):
        self.session = session
        transport = RequestsHTTPTransport(
            url = self.session.base_url + '/graphql',
        )

        tokens = None
        try:
            tokens = self.session.tokens
        except AttributeError:
            pass

        self.gclient = graphclient(
            tokens = tokens,
            transport = transport,
            fetch_schema_from_transport = True,
        )
        self.cache = None

    def __getitem__(self, guid):
        if guid not in self.guids:
            raise KeyError(guid)
        return cube(guid, self.session, self.gclient)

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
        self.cache = ls(self.gclient)

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
