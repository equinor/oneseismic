const oneseismic = require("oneseismic");
const fs = require("fs");

describe("decoding module", function () {
  var one;
  beforeAll(async function () {
    one = await oneseismic();
  });

  describe("when reading the result data", function () {
    var chunk;
    beforeEach(async function () {
      let read = fs.promises.readFile("spec/data/curtain-1.bin", null);
      chunk = await read;
    });

    function verifyResult(res){
      const head = res[0];
      const body = res[1];
      const data = body["data"];
      expect(head.ndims).toEqual(3);
      expect(data["shape"]).toEqual([5, 850]);
      expect(data["data"].reduce((acc, v) => acc + v)).toBeCloseTo(-1.589, 3);
    }

    it("decodes single buffer correctly", function () {
      const decode = one.stream_decoder();
      decode(chunk);
      const res = decode(null);
      verifyResult(res)
    });

    it("decodes small chunks correctly", function () {
      const decode = one.stream_decoder();
      var res;
      const len = chunk.byteLength;
      for (let i = 0; i < len; i += 5) {
          const end = Math.min(i + 5, len);
          res = decode(chunk.slice(i, end));
      }
      verifyResult(res)
    });
  });
});
