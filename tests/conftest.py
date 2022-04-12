import sys
import pytest
from packaging import version

VERSION_FEATURES = set("data_version client_version server_version".split())


def pytest_addoption(parser):
    for feature in VERSION_FEATURES:
        parser.addoption(
            "--{}".format(feature),
            action="store",
            metavar=format(feature.upper()),
            help="only run tests where {} fits under test [min, max).".format(
                feature),
        )


def pytest_configure(config):
    for feature in VERSION_FEATURES:
        config.addinivalue_line(
            "markers", "{}(min, max): mark test to run only on versions [min, max)".format(
                feature)
        )


def run_versiontest_setup(item):
    def filter_tests(feature):
        feature_markers = [
            mark for mark in item.iter_markers(name=feature)]
        markers_length = len(feature_markers)
        if markers_length == 0:
            # other version markers are specified
            pass
        elif markers_length == 1:
            args = feature_markers[0].args
            args_length = len(args)
            if args_length == 0:
                # no restrictions
                pass
            elif args_length > 2:
                raise ValueError(
                    "Version features expect max 2 args, was {}".format(args_length))
            else:
                supplied_version = item.config.getoption(
                    "--{}".format(feature))
                if supplied_version:
                    supplied_version = version.parse(supplied_version)
                test_min_version = args[0]
                if test_min_version:
                    test_min_version = version.parse(test_min_version)
                if test_min_version and supplied_version and supplied_version < test_min_version:
                    pytest.skip("Test {} supplied version lesser than min ".format(item))
                if args_length == 2:
                    test_max_version = args[1]
                    if test_max_version and not supplied_version:
                        # not supplied version means latest dev release
                        pytest.skip("Test {} has max version and we assume current dev release is bigger ".format(item))
                    if test_max_version:
                        test_max_version = version.parse(test_max_version)
                    if test_min_version and test_max_version and test_min_version > test_max_version:
                        raise ValueError("Test min version {} > test max version {} for feature {}".format(
                            test_min_version, test_max_version, feature))
                    if test_max_version and supplied_version >= test_max_version:
                        pytest.skip("Test {} supplied version bigger than max ".format(item))
        else:
            raise ValueError("Unhandled situation: {} markers of type {}".format(
                markers_length, feature))

    supplied_version_markers = [
        feature for feature in VERSION_FEATURES if item.config.getoption("--{}".format(feature))]
    if len(supplied_version_markers) != 0:
        # there wer supplied version markers - aka we are running version tests
        test_version_markers = [
            mark for feature in VERSION_FEATURES for mark in item.iter_markers(name=feature)]
        if len(test_version_markers) != 0:
            # current test contains at least one version marker so should be checked
            for feature in VERSION_FEATURES:
                filter_tests(feature)
        else:
            # current test has no version markers, so skip it from version runs
            pytest.skip("Item {} is not a version test".format(item))



def pytest_runtest_setup(item):
    run_versiontest_setup(item)
