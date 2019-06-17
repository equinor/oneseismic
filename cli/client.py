#!/usr/bin/env python3
import http.client
import subprocess
import os
import argparse
from urllib.parse import urlparse
import json


def getAccessToken() -> str:
    res = subprocess.run(["oauth2local", "token"], stdout=subprocess.PIPE)
    if res.returncode != 0:
        raise "No token in ouath2local store " + \
            res.stdout.decode("utf-8").strip()
    return res.stdout.decode("utf-8").strip()


def clientConnector(scUrl):
    o = urlparse(scUrl)
    op = o.netloc + o.path
    if o.scheme == "http":
        return http.client.HTTPConnection(op)
    else:
        return http.client.HTTPSConnection(op)


def sendSurface(scUrl, manID, surface, outFile, at):
    conn = clientConnector(scUrl)
    conn.request("POST", "/stitch/"+manID,
                 headers={"Authorization": "Bearer " + at},
                 body=open(surface, mode='rb'))
    r1 = conn.getresponse()
    if r1.status == 200:
        if outFile is None:
            print(r1.read())
        else:
            try:
                b = open(outFile, mode="wb")
                b.write(r1.read())
                b.close()
            except Exception as e:
                print("Error:" + str(e))

    else:
        print("Error:" + str(r1.status))
        print(r1.read())


def verify(binFile, cube, surface):
    return True


def sendSurfaces(configFile, at):
    config = json.loads(open(configFile))
    outFile = "work.i32"
    for apiUrl in config.apis:
        for bench in config.benchs:
            for surface in bench.surface:
                open(outFile, 'w').close()
                sendSurface(apiUrl, bench.cube, surface, outFile, at)
                if not verify(outFile, bench.cube, surface):
                    print("Error: Response is not verifiable",
                          outFile, bench.cube, surface)


def main():
    parser = argparse.ArgumentParser(
        description='Send surface to seismic cloud.')
    parser.add_argument('-c', dest='configFile',
                        help='Config file with benchmarks')
    parser.add_argument('-m', dest='manID',
                        help='manifest id for cube')
    parser.add_argument('-i',
                        help='in surface file path')
    parser.add_argument('-t',
                        help='access token for user')
    parser.add_argument('-o',
                        help='out surface file path')
    parser.add_argument('--sc-url', dest='scURL',
                        help='url for seismic cloud api')
    args = parser.parse_args()
    if args.scURL is None:
        parser.print_usage()
        return
    if args.manID is None:
        parser.print_usage()
        return
    if args.i is None:
        parser.print_usage()
        return

    if len(args.t) == 0:
        at = getAccessToken()
    else:
        at = args.t

    if args.configFile is None:
        sendSurface(args.scUrl, args.manID, args.surface, args.outFile, at)
    else:
        sendSurfaces(args.configFile, at)


if __name__ == '__main__':
    main()
