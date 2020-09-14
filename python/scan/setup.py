import setuptools

with open('readme') as f:
    long_description = f.read()

setuptools.setup(
    name = 'oneseismic-scan',
    description = 'oneseismic - Understand seismic cube identity and layout',
    long_description = long_description,

    url = 'https://github.com/equinor/oneseismic',
    license = 'AGPL-3.0',
    version = '0.1.2',

    packages = [
        'scan',
    ],

    install_requires = [
        'segyio',
    ],

    setup_requires = [
        'pytest-runner',
    ],

    tests_require = [
        'pytest',
        'hypothesis',
    ],
)
