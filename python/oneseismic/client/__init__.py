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


def new_from_sas(sas_token, base_url):
    """Create a new client from a SAS token

    Parameters
    ----------
    sas_token : str
        SAS token granting access to the backing storage. Note that the
        permissions should be set to grant access to all operations that will be
        triggered towards the backing storage. This might include listing
        containers in the account as well as read access to the containers
        containing your data.
    base_url : str
        The base url, schema + host, for the oneseismic service

    Returns
    -------
    cli : oneseismic.client.cli

    Examples
    --------
    Get a list of cubes:
    >>> cli = oneseismic.client.new_from_sas(sas_token, base_url)
    >>> [x for x in cli.cubes]
    ['038855eeb243a56b8c39046df4c2b894abcdabb8',
     '0d235a7138104e00c421e63f5e3261bf2dc3254b',
     '10e6302b435da5aa6adf538fe0f99ad2c9ba109d']
    """
    session = http_session.from_sas_token(
        sas_token=sas_token,
        base_url=base_url
    )

    return cli(session=session)
