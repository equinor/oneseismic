import setuptools

with open('readme') as f:
    long_description = f.read()

setuptools.setup(
    name = 'oneseismic-upload',
    description = 'oneseismic - upload cubes to storage',
    long_description = long_description,

    url = 'https://github.com/equinor/oneseismic',
    license = 'AGPL-3.0',
    version = '0.1.0',

    packages = [
        'upload',
    ],

    install_requires = [
        'segyio',
        'numpy',
        'azure-storage-blob',
        'tqdm',
    ],

    setup_requires = [
        'pytest-runner',
    ],

    tests_require = [
        'pytest',
        'hypothesis',
        'segyio',
    ],
)
