import setuptools

setuptools.setup(
    name = 'scan',

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
