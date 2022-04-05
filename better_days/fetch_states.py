import argparse
import json

import requests

def main(outfile):
    data = requests.get('https://emapa.fra1.cdn.digitaloceanspaces.com/statuses.json').json()
    mapping = {name: list(state['districts']) for name, state in data['states'].items()}
    print(mapping)
    with open(outfile, 'w') as fobj:
        fobj.write(json.dumps(mapping, indent=4))

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('outfile')
    args = parser.parse_args()
    main(**vars(args))
