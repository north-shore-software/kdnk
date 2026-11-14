import { execFileSync } from 'node:child_process';
import { mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const gomarkdoc = 'github.com/princjef/gomarkdoc/cmd/gomarkdoc@v1.1.0';
const packages = {
  agentsmith: 'agentsmith',
  providers: 'providers',
};

const root = fileURLToPath(new URL('../../..', import.meta.url));
const out = join(root, 'docs', 'reference');

rmSync(out, { recursive: true, force: true });
mkdirSync(out, { recursive: true });

for (const [pkg, dir] of Object.entries(packages)) {
  execFileSync('go', ['run', gomarkdoc, '--output', `docs/reference/${pkg}.md`, `./${dir}`], {
    cwd: root,
    stdio: 'inherit',
  });
}

for (const pkg of Object.keys(packages)) {
  const file = join(out, `${pkg}.md`);
  const body = readFileSync(file, 'utf8').replace(/^# .*\n/m, '');
  writeFileSync(file, `---\ntitle: ${pkg}\n---\n\n${body}`);
}
