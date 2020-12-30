from .client import http_session
from .client import cube
from .client import cubes
from .client import cli

def new(cache_dir = None):
    """Create a new client

    Create a new client instance with reasonable defaults. When writing
    programs, you should generally use new() to create accessors to oneseismic.

    This function only creates a cli object with reasonable defaults. For more
    fine-grained control, sessions and cli objects can be created manually.

    Parameters
    ----------
    cache_dir : path or str, optional
        Configuration cache directory

    Returns
    -------
    cli : oneseismic.client.cli

    Examples
    --------
    Get a list of cubes:
    >>> cli = oneseismic.client.new()
    >>> [x for x in cli.cubes]
    ['038855eeb243a56b8c39046df4c2b894abcdabb8',
     '0d235a7138104e00c421e63f5e3261bf2dc3254b',
     '10e6302b435da5aa6adf538fe0f99ad2c9ba109d']
    """
    session = http_session.fromconfig(cache_dir = cache_dir)
    return cli(session = session)
