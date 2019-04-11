#!/usr/bin/env python

import argparse
import re
import numpy as np

frgsize_pettern = r'Fragment size: x: (\d*), y: (\d*), z: (\d*)'
ttl_time_pattern = r'Total elapsed time: (\d*)ms'

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('source_file')

    args = parser.parse_args()

    f = open(args.source_file)
    results = {}

    while True:
        line = f.readline()

        if line == '': break

        fragsize = re.search(frgsize_pettern, line).groups()
        f.readline()
        f.readline()
        f.readline()
        ttl_time = int(re.search(ttl_time_pattern, f.readline()).group(1))
        f.readline()

        if fragsize in results:
            results[fragsize].append(ttl_time)
        else:
            results[fragsize] = [ttl_time]

    for r in results:
        a = np.array(results[r])
        report = '{}: avg: {}ms, std: {}ms, min: {}ms, max: {}ms' \
                 .format(r, int(a.mean()), int(a.std()), min(a), max(a))
        print(report)
