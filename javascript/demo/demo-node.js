const one = require("oneseismic");
const https = require("https");

/**
 * To run this demo:
 *   1. Build the oneseismic library in module mode (aka ONESEISMIC_MODULARIZE=ON)
 *   2. Let node environment know where to find it (export NODE_PATH=path_to_dir)
 *   3. Update server, guid and sas_token.
 *      For demo purpose container must be small and result is expected immediately
 *      Exchange https with http in "require" if necessary
 */

one().then((mod) => {
  server = "<oneseismic-endpoint>";
  guid = "<guid>";
  sas_token = "<sas_token>";

  url = `${server}/graphql?${sas_token}`;
  const query = `{ cube (id: "${guid}") { sliceByIndex(dim: 0, index: 0) } }`;

  const req = https.request(
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
        https.get(
          `${server}/${promise.url}/stream`,
          {
            headers: {
              Authorization: `Bearer ${promise.key}`,
            },
          },
          (res) => {
            const { statusCode } = res;
            if (statusCode !== 200) {
              console.error("Request Failed.\n" + `Status Code: ${statusCode}`);
              res.resume();
              return;
            }
            dec = mod.stream_decoder();
            res.on("data", function (chunk) {
              console.log("Got a chunk!");
              var d = dec(chunk);
              d = dec(null);
              let data = d[1];
              let vals = data["data"]["data"];
              let sum = vals.reduce((acc, v) => acc + v);
              console.log(sum);
            });
            res.on("end", function () {
              console.log("End of data");
            });
          }
        );
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
