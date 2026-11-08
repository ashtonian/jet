"""Export portable text with original hashes and deduplicated input manifests."""

import hashlib
import json
from pathlib import Path
import sys


REPO = Path(__file__).resolve().parents[4]


def digest(data):
    return hashlib.sha256(data).hexdigest()


def main(work, destination):
    destination.mkdir(parents=True, exist_ok=False)
    manifests = {}
    originals = {}
    substitutions = [(str(REPO), '$JET_CHECKOUT'),
                     ('/opt/homebrew/Cellar/go/1.27.3/libexec', '$GOROOT'),
                     (str(Path.home()), '$HOME')]

    def portable_text(value):
        for old, new in substitutions:
            value = value.replace(old, new)
        return value

    def compact(value):
        if isinstance(value, dict):
            result = {}
            for key, child in value.items():
                if key in ('inputs', 'source_inputs', 'frozen_inputs_before_queue', 'frozen_inputs_after_run') and isinstance(child, dict):
                    identity = digest(json.dumps(child, sort_keys=True).encode())
                    manifests[identity] = child
                    result[key + '_manifest'] = identity
                else:
                    result[key] = compact(child)
            return result
        if isinstance(value, list):
            return [compact(child) for child in value]
        return value

    def export(source, target):
        raw = source.read_bytes()
        originals[str(source.relative_to(REPO))] = digest(raw)
        text = raw.decode()
        if source.suffix == '.json':
            text = json.dumps(compact(json.loads(text)), indent=2)
        else:
            text = '\n'.join(line.rstrip() for line in text.splitlines()
                             if not line.startswith('gate-lock:')) + '\n'
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(portable_text(text))

    export(work / 'sources.json', destination / 'sources.json')
    results = work / 'results'
    for source in sorted(results.rglob('*')):
        if source.is_file() and source.suffix in ('.json', '.txt', '.log'):
            export(source, destination / 'results' / source.relative_to(results))
    labels = ['release-build', 'release-core', 'release-profiles', 'release-profiles-resume',
              'release-diagnostics', 'release-profile-analysis', 'release-profile-analysis-resume',
              'core-analysis-final', 'callback-cancellation-samples', 'driver-coverage-completion',
              'cancellation-analysis-counts', 'cancellation-analysis-boundary',
              'driver-controls-report-final', 'core-analysis-report-final']
    for label in labels:
        directory = work.parent / label
        if not directory.exists():
            raise RuntimeError('missing stage: ' + label)
        for source in sorted(directory.iterdir()):
            if source.is_file() and source.suffix in ('.json', '.log'):
                export(source, destination / 'stages' / label / source.name)
    for name in ['ci-54b5614.json', 'signatures-54b5614.json']:
        export(work.parent / name, destination / name)
    for source in sorted(results.rglob('*')):
        if source.is_file() and (source.suffix in ('.pprof', '.test')):
            originals[str(source.relative_to(REPO))] = digest(source.read_bytes())
    (destination / 'input-manifests.json').write_text(portable_text(json.dumps(manifests, indent=2)))
    (destination / 'raw-artifact-sha256.json').write_text(json.dumps(originals, indent=2))
    description = {
        'source': str(work.relative_to(REPO)),
        'adaptations': ['Replace checkout, home and Go installation paths with placeholders.',
                        'Omit host-gate queue/holder notices; receipts preserve this stage timing and admission.',
                        'Normalize trailing whitespace in text logs.',
                        'Intern repeated input maps by the SHA-256 of their sorted JSON representation.'],
        'raw_binary_profiles': 'Retained locally at the original source paths; hashes are included. No binary or pprof file is copied.',
        'acceptance': 'Core target failed. Missing final-program groups remain explicit; packaging is blocked.'}
    (destination / 'export.json').write_text(json.dumps(description, indent=2))
    exported = {str(path.relative_to(destination)): digest(path.read_bytes())
                for path in sorted(destination.rglob('*')) if path.is_file()}
    (destination / 'portable-artifact-sha256.json').write_text(json.dumps(exported, indent=2))
    print(json.dumps({'files': len(exported), 'input_manifests': len(manifests)}, indent=2))


if __name__ == '__main__':
    main(Path(sys.argv[1]).resolve(), Path(sys.argv[2]).resolve())
