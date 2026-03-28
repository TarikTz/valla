#!/usr/bin/env node
'use strict'

const { execFileSync } = require('child_process')
const { createGunzip } = require('zlib')
const tar = require('tar')
const https = require('https')
const fs = require('fs')
const os = require('os')
const path = require('path')

const VERSION = require('./package.json').version
const ORG = 'tariktz'
const REPO = 'valla-cli'

function getPlatform() {
  const platform = process.platform
  if (platform === 'darwin') return 'Darwin'
  if (platform === 'linux') return 'Linux'
  if (platform === 'win32') return 'Windows'
  throw new Error(`Unsupported platform: ${platform}`)
}

function getArch() {
  const arch = process.arch
  if (arch === 'x64') return 'x86_64'
  if (arch === 'arm64') return 'arm64'
  throw new Error(`Unsupported arch: ${arch}`)
}

function getCachePath() {
  const base = process.platform === 'win32'
    ? path.join(process.env.LOCALAPPDATA || os.homedir(), 'valla-cli')
    : path.join(os.homedir(), '.cache', 'valla-cli')
  const binaryName = process.platform === 'win32' ? 'valla-cli.exe' : 'valla-cli'
  return path.join(base, VERSION, binaryName)
}

function downloadBinary(url, dest) {
  return new Promise((resolve, reject) => {
    fs.mkdirSync(path.dirname(dest), { recursive: true })
    https.get(url, (res) => {
      if (res.statusCode === 301 || res.statusCode === 302) {
        return downloadBinary(res.headers.location, dest).then(resolve).catch(reject)
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`HTTP ${res.statusCode} for ${url}`))
      }
      if (url.endsWith('.zip')) {
        const tmpZip = dest + '.zip'
        const file = fs.createWriteStream(tmpZip)
        res.pipe(file)
        file.on('finish', () => {
          file.close()
          execFileSync('powershell', [
            '-Command',
            `Expand-Archive -Path "${tmpZip}" -DestinationPath "${path.dirname(dest)}" -Force`
          ])
          fs.unlinkSync(tmpZip)
          resolve()
        })
        file.on('error', reject)
      } else {
        const gunzip = createGunzip()
        const extract = tar.extract({ cwd: path.dirname(dest), strip: 0 })
        res.pipe(gunzip).pipe(extract)
        extract.on('finish', () => {
          fs.chmodSync(dest, 0o755)
          resolve()
        })
        extract.on('error', reject)
        gunzip.on('error', reject)
      }
    }).on('error', reject)
  })
}

async function main() {
  const cachePath = getCachePath()
  if (!fs.existsSync(cachePath)) {
    const platform = getPlatform()
    const arch = getArch()
    const ext = process.platform === 'win32' ? '.zip' : '.tar.gz'
    const filename = `valla-cli_${VERSION}_${platform}_${arch}${ext}`
    const url = `https://github.com/${ORG}/${REPO}/releases/download/v${VERSION}/${filename}`
    console.error(`Downloading valla-cli v${VERSION}...`)
    await downloadBinary(url, cachePath)
  }
  execFileSync(cachePath, process.argv.slice(2), { stdio: 'inherit' })
}

main().catch((err) => {
  console.error(err.message)
  process.exit(1)
})