import setuptools

setuptools.setup(
    name = 'upload',

    packages = [
        'upload',
    ],

    install_requires = [
        'segyio',
        'numpy',
        'azure-storage',
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
