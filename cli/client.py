#!/usr/bin/env python3
from http.client import HTTPConnection, HTTPSConnection
import subprocess
import os
import argparse
from urllib.parse import urlparse
import json
import time
from typing import Union


def getAccessToken() -> str:
    res = subprocess.run(["oauth2local", "token"], stdout=subprocess.PIPE)
    if res.returncode != 0:
        raise Exception("No token in ouath2local store " + \
            res.stdout.decode("utf-8").strip())
    return res.stdout.decode("utf-8").strip()


def connectionProvider(scUrl) -> Union[HTTPConnection, HTTPSConnection]:
    o = urlparse(scUrl)
    op = o.netloc
    print("address is",op)
    if o.scheme == "http":
        return HTTPConnection(op)
    else:
        print("Connection is secure")
        return HTTPSConnection(op)


def sendSurface(scUrl, manID, surface, outFile, at):
    print("Sending surface to", scUrl,manID,surface,outFile)
    start_time = time.time()
    success = False
    conn = connectionProvider(scUrl)
    
    o = urlparse(scUrl)
    print(o.path+"/stitch/"+manID)
    conn.request("POST", o.path+"/stitch/"+manID,
                 headers={"Authorization": "Bearer " + at},
                 body=open(surface, mode='rb'))
    r1 = conn.getresponse()
    procDuration = 0
    if r1.status == 200:
        procDuration = r1.getheader("Duration", 0)
        if outFile is None:
            print(r1.read())
            success = True
        else:
            try:
                b = open(outFile, mode="wb")
                b.write(r1.read())
                b.close()
                success = True
            except Exception as e:
                print("Error:" + str(e))

    else:
        print("Error:" + str(r1.status))
        print(r1.read())
    elapsed_time = time.time() - start_time
    print("Time elapsed", elapsed_time, "Processing time", procDuration)
    return success


def verify(verifier,cubeDir, workFile, cube, surface) -> bool:
    res = subprocess.run(
        [verifier, '-i',cubeDir, cube+".manifest", surface],stdin=open(workFile), stdout=subprocess.PIPE)
    return res.returncode == 0


def sendSurfaces(configFile, at):
    config = json.loads(open(configFile).read())
    outFile = "work.i32"
    for apiUrl in config['apis']:
        for bench in config['benchs']:
            for surface in bench['surfaces']:
                open(outFile, 'w').close()
                succ = sendSurface(apiUrl, bench['cube'], surface, outFile, at)
                if not succ:
                    print("Error: Api failed to process",
                          outFile, bench['cube'], surface)
                    return False

                if not verify(config['verifier'], config['cube_dir'], outFile, bench['cube'], surface):
                    print("Error: Response is not verifiable",
                          outFile, bench.cube, surface)
                    return False


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
    if args.configFile is None:
        if args.scURL is None:
            parser.print_usage()
            return
        if args.manID is None:
            parser.print_usage()
            return
        if args.i is None:
            parser.print_usage()
            return

        if args.t is None:
            at = getAccessToken()
        else:
            at = args.t
        sendSurface(args.scURL, args.manID, args.i, args.o, at)
    else:
        if args.t is None or len(args.t) == 0:
            at = getAccessToken()
        else:
            at = args.t
        sendSurfaces(args.configFile, at)


if __name__ == '__main__':
    main()
