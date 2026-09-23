// Optional local smoke runner: Node + Python/PyYAML. Executes the collection's
// HTTP requests and its current assertion subset, not the Postman application.
const { execFileSync } = require('node:child_process');
const assert = require('node:assert/strict');
const path = require('node:path');
const vm = require('node:vm');

const root = path.resolve(__dirname, '../docs/postman/collections/Hackathon API');
const python = process.env.PYTHON || 'python';
const parsed = JSON.parse(execFileSync(python, ['-c',
  'import json,pathlib,sys,yaml; root=pathlib.Path(sys.argv[1]); print(json.dumps({"definition":yaml.safe_load((root/".resources/definition.yaml").read_text(encoding="utf-8")),"requests":[dict(yaml.safe_load(p.read_text(encoding="utf-8")),file=str(p.relative_to(root))) for p in root.rglob("*.request.yaml")]}))', root], { encoding: 'utf8' }));
const variables = new Map(Object.entries(parsed.definition.variables));
if (process.env.BASE_URL) variables.set('base_url', process.env.BASE_URL);
const base = new URL(variables.get('base_url'));
assert(['localhost', '127.0.0.1', '[::1]'].includes(base.hostname), 'Smoke tests must target a local server');
const replace = text => text.replace(/\{\{([^}]+)\}\}/g, (_, key) => {
  assert(variables.has(key), `Unknown collection variable ${key}`);
  return String(variables.get(key));
});
let checks = 0;
function expect(value, negate = false) {
  const check = fn => {
    if (!negate) fn();
    else assert.throws(fn, assert.AssertionError);
  };
  const chain = {
    eql: expected => check(() => assert.deepEqual(value, expected)),
    include: expected => check(() => assert(value.includes(expected))),
    property: name => check(() => assert(Object.hasOwn(value, name))),
    an: type => check(() => assert.equal(Array.isArray(value) ? 'array' : typeof value, type)),
    least: min => check(() => assert(value >= min)),
  };
  for (const key of ['to', 'have', 'be', 'at', 'and']) chain[key] = chain;
  Object.defineProperty(chain, 'not', { get: () => expect(value, !negate) });
  return chain;
}
async function main() {
  const requests = parsed.requests.sort((a, b) => a.order - b.order);
  assert.equal(requests.length, 24, 'Unexpected request count');
  for (const request of requests) {
    assert.equal(request.$kind, 'http-request');
    const headers = Object.fromEntries(Object.entries(request.headers).map(([key, value]) => [key, replace(value)]));
    const body = request.body ? replace(request.body.content) : undefined;
    if (body) JSON.parse(body);
    const response = await fetch(replace(request.url), { method: request.method, headers, body, signal: AbortSignal.timeout(30000) });
    const json = await response.json();
    const pm = {
      test: (name, fn) => { try { fn(); checks++; } catch (error) { throw new Error(`${request.file}: ${name}: ${error.message}`); } },
      expect,
      collectionVariables: { set: (key, value) => variables.set(key, value), get: key => variables.get(key) },
      response: { code: response.status, json: () => json, headers: response.headers,
        to: { have: { status: expected => assert.equal(response.status, expected, JSON.stringify(json)) } } },
    };
    for (const script of request.scripts || []) {
      assert.equal(script.type, 'afterResponse');
      vm.runInNewContext(script.code, { pm }, { timeout: 1000 });
    }
    console.log(`${response.status} ${request.file}`);
  }
  console.log(`PASS: ${requests.length} requests, ${checks} collection assertions; task=${variables.get('task_id')}`);
}
main().catch(error => { console.error(error.message); process.exitCode = 1; });
