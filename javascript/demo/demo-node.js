const one = require('oneseismic')
const http = require('http')

one().then(mod => {
    http.get("http://host:port/result/<pid>/stream", {
            headers: {
                'Authorization': 'Bearer <token>'
            }
        }, (res) => {
            dec = mod.decode_stream()
            res.on('data', dec)
            res.on('end', () => dec(null));
        }).then(d => {
            let data = d[1]
            let vals = data['data']['data']
            let sum = vals.reduce((acc, v) => acc + v);
            console.log(sum);
        })
    })
;
