import skbuild
import pybind11

class get_pybind_include(object):
    def __init__(self, user=False):
        self.user = user

    def __str__(self):
        # postpone importing pybind11 until building actually happens
        import pybind11
        return pybind11.get_include(self.user)


pybind_includes = [
    str(get_pybind_include()),
    str(get_pybind_include(user = True))
]


skbuild.setup(
    name = 'seismic cloud',
    author = 'Equinor ASA',
    author_email = 'ssru@equinor.com',
    setup_requires = [
        'setuptools',
        'pytest-runner',
        'pybind11',
    ],
    tests_require = ['pytest'],
    install_requires = ['grpcio'],
    packages=['server', 'server/proto'],
    cmake_args=['-DPYBIND11_INCLUDE_DIRS=' + ';'.join(pybind_includes)],
    entry_points = {
        'console_scripts': ['core-server=server.server:main'],
    }
)
