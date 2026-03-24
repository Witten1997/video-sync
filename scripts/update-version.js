const fs = require('fs');
const path = require('path');

const versionFile = path.join(__dirname, '..', 'internal', 'version', 'version.go');

function readCurrentVersion() {
  const content = fs.readFileSync(versionFile, 'utf-8');
  const match = content.match(/Version\s*=\s*"([^"]+)"/);
  return match ? match[1] : '0.0.0';
}

function writeVersion(newVersion) {
  let content = fs.readFileSync(versionFile, 'utf-8');
  content = content.replace(/Version\s*=\s*"[^"]+"/, `Version   = "${newVersion}"`);
  fs.writeFileSync(versionFile, content, 'utf-8');
  console.log(`Version updated: ${readCurrentVersion()} -> ${newVersion}`);
}

function bumpVersion(current, type) {
  const parts = current.split('.').map(Number);
  switch (type) {
    case 'major': return `${parts[0] + 1}.0.0`;
    case 'minor': return `${parts[0]}.${parts[1] + 1}.0`;
    case 'patch': return `${parts[0]}.${parts[1]}.${parts[2] + 1}`;
    default: return current;
  }
}

const arg = process.argv[2];
if (!arg) {
  console.log(`Usage: node update-version.js <version|patch|minor|major>`);
  console.log(`Current: ${readCurrentVersion()}`);
  process.exit(1);
}

const current = readCurrentVersion();
const newVersion = ['patch', 'minor', 'major'].includes(arg)
  ? bumpVersion(current, arg)
  : arg;

writeVersion(newVersion);
