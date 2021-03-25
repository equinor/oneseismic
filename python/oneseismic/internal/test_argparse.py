from pathlib import Path
from .argparse import localfs_from_args

def test_localfs_root_from_args():
    assert localfs_from_args(None).root   == Path('.')
    assert localfs_from_args('/usr').root == Path('/usr')
    assert localfs_from_args('rel').root  == Path('.') / Path('rel')
