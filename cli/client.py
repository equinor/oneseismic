import http.client
import subprocess
import os
import argparse


def getAccessToken() -> str:
    res = subprocess.run(["oauth2local", "token"], stdout=subprocess.PIPE)
    return res.stdout.decode("utf-8").strip()


def main():
    parser = argparse.ArgumentParser(
        description='Send surface to seismic cloud.')
    parser.add_argument('-m', dest='manID',
                        help='manifest id for cube')
    parser.add_argument('-i',
                        help='in surface file path')
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

    at = getAccessToken()

    conn = http.client.HTTPSConnection(args.scURL)
    conn.request("POST", "/stitch/"+args.manID,
                 headers={"Authorization": "Bearer " + at},
                 body=open(args.i, mode='rb'))
    r1 = conn.getresponse()
    if r1.status == 200:
        if args.o is None:
            print(r1.read())
        else:
            try:
                b = open(args.o, mode="wb")
                b.write(r1.read())
            except Exception as e:
                print("Error:" + str(e))

    else:
        print("Error:" + str(r1.status))
        print(r1.read())


if __name__ == '__main__':
    main()
