import setuptools

setuptools.setup(
    name="blobio",
    description="simple streams for azure blob store",
    packages=["blobio"],
    platforms="any",
    install_requires=["azure-storage-blob"],
)
