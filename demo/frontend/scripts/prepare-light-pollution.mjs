import { closeSync, fsyncSync, mkdirSync, openSync, renameSync, rmSync, writeSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { endianness } from 'node:os'
import { fromFile } from 'geotiff'

const MAGIC = Buffer.from('AURVNL1\n', 'ascii')
const NO_DATA = 65535

function optionsFrom(argv) {
  if (argv[0] === '--') argv = argv.slice(1)
  const options = {
    west: 72,
    south: 18,
    east: 136,
    north: 54,
    year: 2025,
    resolutionMeters: 500,
    scale: 0.1,
    chunkRows: 64,
    source: 'NASA Black Marble VJ146A4 2025 AllAngle_Composite_Snow_Free',
  }
  for (let index = 0; index < argv.length; index += 2) {
    const key = argv[index]?.replace(/^--/, '')
    const value = argv[index + 1]
    if (!key || value == null) throw new Error(`参数缺少值：${argv[index] ?? ''}`)
    if (['input', 'output', 'source'].includes(key)) options[key] = value
    else if (['west', 'south', 'east', 'north', 'scale'].includes(key)) options[key] = Number(value)
    else if (['year', 'resolutionMeters', 'chunkRows'].includes(key)) options[key] = Number.parseInt(value, 10)
    else throw new Error(`未知参数：--${key}`)
  }
  if (!options.input || !options.output) throw new Error('必须提供 --input 和 --output')
  if (![options.west, options.south, options.east, options.north, options.scale].every(Number.isFinite)) throw new Error('范围和 scale 必须是数字')
  if (options.west >= options.east || options.south >= options.north || options.scale <= 0) throw new Error('范围或 scale 无效')
  return options
}

function writeAll(fd, buffer) {
  let offset = 0
  while (offset < buffer.length) offset += writeSync(fd, buffer, offset)
}

function encodeRadiance(values, scale) {
  const encoded = new Uint16Array(values.length)
  for (let index = 0; index < values.length; index += 1) {
    const value = values[index]
    encoded[index] = Number.isFinite(value) && value >= 0
      ? Math.min(NO_DATA - 1, Math.round(value / scale))
      : NO_DATA
  }
  if (endianness() === 'LE') return Buffer.from(encoded.buffer, encoded.byteOffset, encoded.byteLength)
  const output = Buffer.allocUnsafe(encoded.byteLength)
  for (let index = 0; index < encoded.length; index += 1) output.writeUInt16LE(encoded[index], index * 2)
  return output
}

async function main() {
  const options = optionsFrom(process.argv.slice(2))
  const input = resolve(options.input)
  const output = resolve(options.output)
  const temporary = `${output}.tmp`
  mkdirSync(dirname(output), { recursive: true })

  const tiff = await fromFile(input)
  let fd
  try {
    const image = await tiff.getImage()
    const [sourceWest, sourceSouth, sourceEast, sourceNorth] = image.getBoundingBox()
    const sourceWidth = image.getWidth()
    const sourceHeight = image.getHeight()
    const pixelWidth = (sourceEast - sourceWest) / sourceWidth
    const pixelHeight = (sourceNorth - sourceSouth) / sourceHeight

    const requestedWest = Math.max(options.west, sourceWest)
    const requestedSouth = Math.max(options.south, sourceSouth)
    const requestedEast = Math.min(options.east, sourceEast)
    const requestedNorth = Math.min(options.north, sourceNorth)
    if (requestedWest >= requestedEast || requestedSouth >= requestedNorth) throw new Error('请求范围与输入 GeoTIFF 没有交集')

    const x0 = Math.max(0, Math.floor((requestedWest - sourceWest) / pixelWidth))
    const x1 = Math.min(sourceWidth, Math.ceil((requestedEast - sourceWest) / pixelWidth))
    const y0 = Math.max(0, Math.floor((sourceNorth - requestedNorth) / pixelHeight))
    const y1 = Math.min(sourceHeight, Math.ceil((sourceNorth - requestedSouth) / pixelHeight))
    const width = x1 - x0
    const height = y1 - y0
    const header = {
      version: 1,
      source: options.source,
      dataYear: options.year,
      resolutionMeters: options.resolutionMeters,
      west: sourceWest + x0 * pixelWidth,
      south: sourceNorth - y1 * pixelHeight,
      east: sourceWest + x1 * pixelWidth,
      north: sourceNorth - y0 * pixelHeight,
      width,
      height,
      scale: options.scale,
      noData: NO_DATA,
      encoding: 'uint16-le',
    }
    const headerBytes = Buffer.from(JSON.stringify(header), 'utf8')
    const headerLength = Buffer.allocUnsafe(4)
    headerLength.writeUInt32LE(headerBytes.length)

    rmSync(temporary, { force: true })
    fd = openSync(temporary, 'wx')
    writeAll(fd, MAGIC)
    writeAll(fd, headerLength)
    writeAll(fd, headerBytes)

    for (let row = y0; row < y1; row += options.chunkRows) {
      const rowEnd = Math.min(y1, row + options.chunkRows)
      const values = await image.readRasters({ window: [x0, row, x1, rowEnd], samples: [0], interleave: true })
      writeAll(fd, encodeRadiance(values, options.scale))
      const percent = ((rowEnd - y0) / height * 100).toFixed(1)
      process.stdout.write(`\r裁切并编码 ${percent}%`)
    }
    process.stdout.write('\n')
    fsyncSync(fd)
    closeSync(fd)
    fd = undefined
    renameSync(temporary, output)
    console.log(`已生成 ${output}`)
    console.log(JSON.stringify(header, null, 2))
  } catch (error) {
    if (fd != null) closeSync(fd)
    rmSync(temporary, { force: true })
    throw error
  } finally {
    if (typeof tiff.close === 'function') await tiff.close()
  }
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : error)
  process.exitCode = 1
})
