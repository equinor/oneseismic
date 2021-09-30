import contextlib
import urllib.parse

import gql
from gql.transport.requests import RequestsHTTPTransport
import requests

import oneseismic.internal as internal
import oneseismic.decoding as decoding

def add_url_path(url, path):
    split = urllib.parse.urlsplit(url)
    return urllib.parse.urlunsplit((
        split.scheme,
        split.netloc,
        '/'.join([split.path, path]),
        split.query,
        split.fragment,
    ))

def remove_graphql_path(url):
    split = urllib.parse.urlsplit(url)
    return urllib.parse.urlunsplit((
        split.scheme,
        split.netloc,
        '/'.join(split.path.split('/')[:-1]),
        split.query,
        split.fragment,
    ))

class simple_result:
    def __init__(self, process, url):
        self.process = process
        self.url = remove_graphql_path(url)

    def __repr__(self):
        pid = self.process.pid
        query = self.process.query
        name = type(self).__name__
        return f'{name}(pid = {pid}, query = {query})'

    def decoded(self):
        try:
            return self.cached_decoded
        except AttributeError:
            r = requests.get(
                self.process.stream(self.url),
                headers = self.process.headers(),
                stream = True,
            )
            r.raise_for_status()
            self.cached_decoded = decoding.decode_stream(r.iter_content(None))
            return self.cached_decoded

    def numpy(self):
        return decoding.numpy(self.decoded())

    def xarray(self):
        return decoding.xarray(self.decoded())

@contextlib.contextmanager
def transport_opts(gclient, params, headers):
    if params is None and headers is None:
        yield gclient
    else:
        # This is (potentially) buggy if the user has modified the transport
        # for whatever reason, since it does not even try to preserve settings.
        #
        # I'm not sure if it's ever useful to try and handle that case, because
        # if there are significant user modifications of the transport
        # mechanism in the simple client then there's a need for a less
        # assuming interface anyway.
        ot = gclient.transport
        if headers is not None:
            headers.update(ot.headers)
        else:
            headers = ot.headers
        gclient.transport = RequestsHTTPTransport(
            url     = ot.url,
            headers = headers,
            params  = params,
        )
        yield gclient
        gclient.transport = ot

def prepared_query(client, gq, variables, *args, **kwargs):
    def fn(sas = None, *args, **kwargs):
        if sas is not None:
            sas = urllib.parse.parse_qs(sas)
            if 'params' not in kwargs:
                kwargs['params'] = {}
            kwargs['params'].update(sas)

        params  = kwargs.pop('params',  None)
        headers = kwargs.pop('headers', None)
        with transport_opts(client, params, headers) as gc:
            r = gc.execute(gq, variable_values = variables, *args, **kwargs)
            p = internal.procs_from_promises(r)
            try:
                proc = next(internal.filter_procs(p))
                return simple_result(proc, client.transport.url)
            except StopIteration:
                return p

    return fn

def check_curtain(curtain):
    for i, coord in enumerate(curtain):
        if len(coord) != 2:
            msg = f'expected pair [x, y], got {coord!r} at index {i}'
            raise ValueError(msg)
    return curtain

class simple_client:
    """Simple and limited queries

    This class provides a quick and easy way to perform some simple queries.
    The class makes some trade offs to be easy to use, in particular few
    options and very limited queries (e.g. mostly structures, little metadata).
    It aims to be *useful* for small programs and testing, rather than powerful.

    To separate query parameters from transport parameters, the query functions
    return a prepared_query callable, which can be used to just-in-time set
    transport options such as shared access signatures.

    Parameters
    ----------
    url : str
        The oneseismic server, e.g. 'https://oneseismic.equinor.com'

    Examples
    --------
    >>> sas = generate_sas(guid)
    >>> sc = simple_client(url)
    >>> np1 = sc.sliceByIndex(guid, dim = 0, index = 150)().numpy()
    >>> np2 = sc.sliceByIndex(guid, dim = 0, index = 150)(sas = sas).numpy()
    >>> (np1 == np1).all()
    True
    """
    def __init__(self, url):
        self.url = url
        self.client = gql.Client(
            transport = RequestsHTTPTransport(
                url = add_url_path(url, 'graphql'),
            ),
            fetch_schema_from_transport = True,
        )

    def get_config(self):
        url = add_url_path(self.url, 'config')
        return requests.get(url).json()

    def metadata(self, guid):
        query = gql.gql('''
            query metadata($id: ID!) {
                cube(id: $id) {
                    linenumbers
                    filenameOnUpload
                    sampleValueMin
                    sampleValueMax
                }
            }
        ''')
        variables = {
            'id': guid,
        }
        return prepared_query(self.client, query, variables)

    def sliceByIndex(self, guid, dim, index, attributes = None):
        """
        Examples
        --------
        >>> sc = simple_client(url)
        >>> proc = sc.sliceByIndex(guid, dim = 0, index = 150)()
        >>> proc.numpy()
        [[0.7283162   0.00633962  0.02923059 ...  1.3414259   0.39454985]
        [ 2.471489    0.5148768  -0.28820574 ...  3.3378954   1.1355114 ]
        [ 1.9840803  -0.05085047 -0.47386587 ...  4.450244    2.9178839 ]
        ...
        [ 0.71043783  2.047329    3.1183748  ... -0.6594674  -1.0297937 ]
        [ 1.0213394   1.283536    0.8089452  ...  0.78478104 -0.07442832]
        [ 0.7409897   0.24140906 -0.5452634  ...  0.37146062  0.03715111]]
        """
        query = gql.gql('''
            query sliceByIndex($id: ID!, $dim: Int!, $idx: Int!, $opts: Opts) {
                cube(id: $id) {
                    sliceByIndex(dim: $dim, index: $idx, opts: $opts)
                }
            }
        ''')
        variables = {
            'id':   guid,
            'dim':  dim,
            'idx':  index,
        }

        if attributes is not None:
            variables['opts'] = { 'attributes': attributes }

        return prepared_query(self.client, query, variables)

    def sliceByLineno(self, guid, dim, lineno, attributes = None):
        """
        Examples
        --------
        >>> sc = simple_client(url)
        >>> proc = sc.sliceByLineno(guid, dim = 1, lineno = 150)()
        >>> proc.numpy()
        [[0.7283162   0.00633962  0.02923059 ...  1.3414259   0.39454985]
        [ 2.471489    0.5148768  -0.28820574 ...  3.3378954   1.1355114 ]
        [ 1.9840803  -0.05085047 -0.47386587 ...  4.450244    2.9178839 ]
        ...
        [ 0.71043783  2.047329    3.1183748  ... -0.6594674  -1.0297937 ]
        [ 1.0213394   1.283536    0.8089452  ...  0.78478104 -0.07442832]
        [ 0.7409897   0.24140906 -0.5452634  ...  0.37146062  0.03715111]]
        """
        query = gql.gql('''
            query sliceByLineno($id: ID!, $dim: Int!, $lno: Int!, $opts: Opts) {
                cube(id: $id) {
                    sliceByLineno(dim: $dim, lineno: $lno, opts: $opts)
                }
            }
        ''')
        variables = {
            'id':   guid,
            'dim':  dim,
            'lno':  lineno,
        }

        if attributes is not None:
            variables['opts'] = { 'attributes': attributes }

        return prepared_query(self.client, query, variables)

    def curtainByIndex(self, guid, curtain, attributes = None):
        """
        Examples
        --------
        >>> sc = simple_client(url)
        >>> proc = sc.curtainByIndex(guid, [[0,0], [0,1], [1,1]])
        >>> proc.numpy()
        [[0.7283162   0.00633962  0.02923059 ...  1.3414259   0.39454985]
        [ 2.471489    0.5148768  -0.28820574 ...  3.3378954   1.1355114 ]
        [ 1.9840803  -0.05085047 -0.47386587 ...  4.450244    2.9178839 ]]
        """
        query = gql.gql('''
            query curtainByIndex($id: ID!, $coords: [[Int!]!]!, $opts: Opts) {
                cube(id: $id) {
                    curtainByIndex(coords: $coords, opts: $Opts)
                }
            }
        ''')

        variables = {
            'id': guid,
            'coords': check_curtain(curtain),
        }

        if attributes is not None:
            variables['opts'] = { 'attributes': attributes }

        return prepared_query(self.client, query, variables)

    def curtainByLineno(self, guid, curtain, attributes = None):
        """
        Examples
        --------
        >>> sc = simple_client(url)
        >>> proc = sc.curtainByLineno(guid, [[0,0], [0,1], [1,1]])
        >>> proc.numpy()
        [[0.7283162   0.00633962  0.02923059 ...  1.3414259   0.39454985]
        [ 2.471489    0.5148768  -0.28820574 ...  3.3378954   1.1355114 ]
        [ 1.9840803  -0.05085047 -0.47386587 ...  4.450244    2.9178839 ]]
        """
        query = gql.gql('''
            query curtainByLineno($id: ID!, $coords: [[Int!]!]!, $opts: Opts) {
                cube(id: $id) {
                    curtainByLineno(coords: $coords, opts: $opts)
                }
            }
        ''')

        variables = {
            'id': guid,
            'coords': check_curtain(curtain),
        }

        if attributes is not None:
            variables['opts'] = { 'attributes': attributes }

        return prepared_query(self.client, query, variables)
