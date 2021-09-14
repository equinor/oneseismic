from .process import procs_from_promises

def test_single_promise():
    response = {
        'cube': {
            'sliceByIndex': {
                'url': '<path>/<pid>',
                'key': '<key>',
            }
        }
    }
    procs = procs_from_promises(response)
    p = procs['cube']['sliceByIndex']
    assert p.path == '<path>/<pid>'
    assert p.pid  == '<pid>'
    assert p.key  == '<key>'

def test_multiple_promises():
    response = {
        'cube': {
            'sliceByIndex': {
                'url': '<ipath>/<ipid>',
                'key': '<ikey>',
            },
            'sliceByLineno': {
                'url': '<lpath>/<lpid>',
                'key': '<lkey>',
            }
        }
    }
    procs = procs_from_promises(response)
    p = procs['cube']['sliceByIndex']
    assert p.path == '<ipath>/<ipid>'
    assert p.pid  == '<ipid>'
    assert p.key  == '<ikey>'

    p = procs['cube']['sliceByLineno']
    assert p.path == '<lpath>/<lpid>'
    assert p.pid  == '<lpid>'
    assert p.key  == '<lkey>'

def test_with_list_item():
    response = {
        'cube': {
            'sliceByIndex': {
                'url': '<path>/<pid>',
                'key': '<key>',
            },
            'linenumbers': [[1,2], [3,4], [5,6]],
        }
    }
    procs = procs_from_promises(response)
    p = procs['cube']['sliceByIndex']
    assert p.path == '<path>/<pid>'
    assert p.pid  == '<pid>'
    assert p.key  == '<key>'

    assert procs['cube']['linenumbers'] == [[1,2], [3,4], [5,6]]

def test_with_string_item():
    response = {
        'cube': {
            'sliceByIndex': {
                'url': '<path>/<pid>',
                'key': '<key>',
            },
            'guid': '<guid>',
        }
    }
    procs = procs_from_promises(response)
    p = procs['cube']['sliceByIndex']
    assert p.path == '<path>/<pid>'
    assert p.pid  == '<pid>'
    assert p.key  == '<key>'

    assert procs['cube']['guid'] == '<guid>'

