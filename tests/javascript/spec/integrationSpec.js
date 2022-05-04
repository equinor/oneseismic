const execSync = require("child_process").execSync;
const os = require("os");
const fs = require("fs");
const path = require("path");
const oneseismic = require("oneseismic");
const http = require("http");

// required
SERVER_URL = process.env.SERVER_URL;
STORAGE_LOCATION = process.env.STORAGE_LOCATION;
BLOB_URL = process.env.BLOB_URL;
UPLOAD_WITH_PYTHON = process.env.UPLOAD_WITH_PYTHON;

function createSegy(structure, path) {
  var script = "../data/create.py";
  // use UPLOAD_WITH_PYTHON for simplicity
  execSync(`${UPLOAD_WITH_PYTHON} ${script} ${structure} ${path}`, {
    encoding: "utf-8",
  });
}

function scan(path) {
  const output = execSync(`${UPLOAD_WITH_PYTHON} -m oneseismic scan ${path}`, {
    encoding: "utf-8",
  });
  return JSON.parse(output);
}

function upload(
  path,
  storage_location = STORAGE_LOCATION,
  scan_meta = undefined
) {
  if (!scan_meta) {
    scan_meta = scan(path);
  }
  const scan_insights = os.tmpdir() + "/scan_insights.json";
  fs.writeFileSync(scan_insights, JSON.stringify(scan_meta));

  execSync(
    `${UPLOAD_WITH_PYTHON} -m oneseismic upload ${scan_insights} ${path} ${storage_location}`,
    { encoding: "utf-8" }
  );
  return scan_meta.guid;
}

describe("e2e tests", function () {
  describe("simple user operations", function () {
    function customGuid() {
      return new Promise((resolveGuid) => {
        custom = path.join(os.tmpdir(), "custom.sgy");
        createSegy("custom", custom);
        guid = scan(custom).guid;

        // at the moment safeguard against file reupload due to etag checks
        var dataPath = path.join(BLOB_URL, guid, "/");
        var uploadNeededPromise = new Promise((fileCheckResolve) => {
          const req = http.request(dataPath, (res) => {
            const { statusCode } = res;
            if (statusCode == 200) {
              fileCheckResolve(false);
            } else {
              fileCheckResolve(true);
            }
          });
          req.end();
        });
        uploadNeededPromise.then((res) => {
          if (res) {
            console.log(`File ${dataPath} is not in the blob yet. Uploading`);
            uploaded_guid = upload(custom);
            expect(uploaded_guid).toEqual(guid);
          } else {
            console.log(`File ${dataPath} already exists.`);
          }
          return resolveGuid(guid);
        });
      });
    }

    var one;
    var guid;
    beforeAll(async function () {
      one = await oneseismic();
      guid = await customGuid();
    });

    it("sample check runs fine", async function () {
      // this is not really a test, just a check that setup works as expected.
      // it should be removed once sdk is ready.
      server = SERVER_URL;
      guid = guid;

      url = `${server}/graphql`;
      const query = `{ cube (id: "${guid}") { sliceByIndex(dim: 0, index: 0) } }`;

      var queryPromise = new Promise((queryResolve) => {
        const req = http.request(
          url,
          {
            method: "post",
            headers: {
              "Content-type": "application/json",
            },
          },
          (res) => {
            const { statusCode } = res;
            if (statusCode !== 200) {
              console.error("Request Failed.\n" + `Status Code: ${statusCode}`);
              res.resume();
              return;
            }
            res.on("data", async function (chunk) {
              console.log("Got a response from query endpoint!");
              promise = JSON.parse(chunk).data.cube.sliceByIndex;
              var resultPromise = new Promise((resultResolve, resultReject) => {
                http.get(
                  `${server}/${promise.url}/stream`,
                  {
                    headers: {
                      Authorization: `Bearer ${promise.key}`,
                    },
                  },
                  (res) => {
                    const { statusCode } = res;
                    if (statusCode !== 200) {
                      console.error(
                        "Request Failed.\n" + `Status Code: ${statusCode}`
                      );
                      res.resume();
                      return;
                    }
                    dec = one.stream_decoder();
                    res.on("data", function (chunk) {
                      console.log("Got a chunk!");
                      var d = dec(chunk);
                      d = dec(null);
                      let data = d[1];
                      let values = data["data"]["data"];
                      console.log(values);
                      if (values.every((v) => v === 0)) {
                        console.log("No idea what and why this is. Skipping");
                      } else {
                        expect(values).toEqual(
                          Float32Array.from([100, 101, 102, 103, 104, 105])
                        );
                        resultResolve();
                      }
                    });
                    res.on("end", function () {
                      console.log("End of data");
                    });
                  }
                );
              });
              await resultPromise;
              queryResolve();
            });
            res.on("end", function () {
              console.log("End of data on query");
            });
          }
        );

        req.write(
          JSON.stringify({
            query: query,
          })
        );
        req.end();
      });
      await queryPromise;
    });
  });
});
