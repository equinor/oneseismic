from pathlib import Path
from .argparse import localfs_from_args

def test_localfs_root_from_args():
    assert localfs_from_args(None).root   == Path('.')
    assert localfs_from_args('/usr').root == Path('/usr')
    assert localfs_from_args('rel').root  == Path('.') / Path('rel')

def test_localfs_file_arg_gets_dir(tmp_path):
    (tmp_path / 'some-file').touch()
    fs = localfs_from_args(tmp_path / 'some-file')
    assert fs.root == tmp_path
    with fs.open('some-file', 'rb'):
        pass
