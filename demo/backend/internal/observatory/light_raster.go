package observatory

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"time"
)

const (
	rasterMagic          = "AURVNL1\n"
	rasterHeaderMaxBytes = 64 << 10
	rasterBytesPerPixel  = 2
)

// ErrNoCoverage 表示观测点不在当前本地栅格覆盖范围内，或对应像素没有有效观测。
var ErrNoCoverage = errors.New("light pollution raster has no coverage for this coordinate")

// rasterHeader 描述 AURORA 的轻量随机访问栅格。像素从西北角开始，按行存储，
// 每个像素为 little-endian uint16；真实辐射值 = 像素值 * Scale。
type rasterHeader struct {
	Version          int     `json:"version"`
	Source           string  `json:"source"`
	DataYear         int     `json:"dataYear"`
	ResolutionMeters int     `json:"resolutionMeters"`
	West             float64 `json:"west"`
	South            float64 `json:"south"`
	East             float64 `json:"east"`
	North            float64 `json:"north"`
	Width            int     `json:"width"`
	Height           int     `json:"height"`
	Scale            float64 `json:"scale"`
	NoData           uint16  `json:"noData"`
	Encoding         string  `json:"encoding"`
}

// RasterLightProvider 从预处理后的 VIIRS 年度栅格中按经纬度随机读取一个像素。
// 文件在启动时只解析一次，查询使用 ReadAt，不会把数百 MB 数据整体载入内存。
type RasterLightProvider struct {
	file       *os.File
	header     rasterHeader
	dataOffset int64
	now        func() time.Time
}

// OpenRasterLightProvider 打开由 frontend/scripts/prepare-light-pollution.mjs
// 生成的 .avnl 数据文件。
func OpenRasterLightProvider(path string) (*RasterLightProvider, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open light pollution raster: %w", err)
	}
	closeOnError := func(err error) (*RasterLightProvider, error) {
		_ = file.Close()
		return nil, err
	}

	magic := make([]byte, len(rasterMagic))
	if _, err := io.ReadFull(file, magic); err != nil {
		return closeOnError(fmt.Errorf("read light pollution raster magic: %w", err))
	}
	if string(magic) != rasterMagic {
		return closeOnError(errors.New("invalid light pollution raster magic"))
	}

	var headerLength uint32
	if err := binary.Read(file, binary.LittleEndian, &headerLength); err != nil {
		return closeOnError(fmt.Errorf("read light pollution raster header length: %w", err))
	}
	if headerLength == 0 || headerLength > rasterHeaderMaxBytes {
		return closeOnError(errors.New("invalid light pollution raster header length"))
	}
	headerBytes := make([]byte, headerLength)
	if _, err := io.ReadFull(file, headerBytes); err != nil {
		return closeOnError(fmt.Errorf("read light pollution raster header: %w", err))
	}

	var header rasterHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return closeOnError(fmt.Errorf("decode light pollution raster header: %w", err))
	}
	if err := validateRasterHeader(header); err != nil {
		return closeOnError(err)
	}

	dataOffset := int64(len(rasterMagic) + 4 + int(headerLength))
	info, err := file.Stat()
	if err != nil {
		return closeOnError(fmt.Errorf("stat light pollution raster: %w", err))
	}
	expectedSize := dataOffset + int64(header.Width)*int64(header.Height)*rasterBytesPerPixel
	if info.Size() != expectedSize {
		return closeOnError(fmt.Errorf("invalid light pollution raster size: got %d bytes, want %d", info.Size(), expectedSize))
	}

	return &RasterLightProvider{file: file, header: header, dataOffset: dataOffset, now: time.Now}, nil
}

func validateRasterHeader(header rasterHeader) error {
	if header.Version != 1 || header.Encoding != "uint16-le" {
		return errors.New("unsupported light pollution raster format")
	}
	if header.Width <= 0 || header.Height <= 0 || header.Scale <= 0 {
		return errors.New("invalid light pollution raster dimensions or scale")
	}
	if !isFinite(header.West) || !isFinite(header.South) || !isFinite(header.East) || !isFinite(header.North) ||
		header.West >= header.East || header.South >= header.North {
		return errors.New("invalid light pollution raster bounds")
	}
	if header.DataYear <= 0 || header.ResolutionMeters <= 0 || header.Source == "" {
		return errors.New("incomplete light pollution raster provenance")
	}
	return nil
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// Light 返回经纬度所在的原始 VIIRS 像素和估算的 SQM/Bortle。
func (provider *RasterLightProvider) Light(ctx context.Context, latitude, longitude float64) (LightPollution, error) {
	if err := ctx.Err(); err != nil {
		return LightPollution{}, err
	}
	if provider == nil || provider.file == nil {
		return LightPollution{}, ErrNotConfigured
	}
	if !isFinite(latitude) || !isFinite(longitude) || longitude < provider.header.West || longitude >= provider.header.East ||
		latitude <= provider.header.South || latitude > provider.header.North {
		return LightPollution{}, ErrNoCoverage
	}

	column := int(math.Floor((longitude - provider.header.West) / (provider.header.East - provider.header.West) * float64(provider.header.Width)))
	row := int(math.Floor((provider.header.North - latitude) / (provider.header.North - provider.header.South) * float64(provider.header.Height)))
	if column < 0 || column >= provider.header.Width || row < 0 || row >= provider.header.Height {
		return LightPollution{}, ErrNoCoverage
	}

	offset := provider.dataOffset + (int64(row)*int64(provider.header.Width)+int64(column))*rasterBytesPerPixel
	var raw [rasterBytesPerPixel]byte
	if _, err := provider.file.ReadAt(raw[:], offset); err != nil {
		return LightPollution{}, fmt.Errorf("read light pollution raster pixel: %w", err)
	}
	encoded := binary.LittleEndian.Uint16(raw[:])
	if encoded == provider.header.NoData {
		return LightPollution{}, ErrNoCoverage
	}

	radiance := float64(encoded) * provider.header.Scale
	return estimatedLightPollution(
		radiance,
		provider.header.Source,
		provider.header.DataYear,
		provider.header.ResolutionMeters,
		provider.now(),
	), nil
}

func (provider *RasterLightProvider) Close() error {
	if provider == nil || provider.file == nil {
		return nil
	}
	return provider.file.Close()
}
