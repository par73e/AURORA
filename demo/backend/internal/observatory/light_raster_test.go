package observatory

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestLightRaster(t *testing.T, header rasterHeader, pixels []uint16) string {
	t.Helper()
	headerBytes, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	data.WriteString(rasterMagic)
	if err := binary.Write(&data, binary.LittleEndian, uint32(len(headerBytes))); err != nil {
		t.Fatal(err)
	}
	data.Write(headerBytes)
	if err := binary.Write(&data, binary.LittleEndian, pixels); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "test.avnl")
	if err := os.WriteFile(path, data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func testRasterHeader() rasterHeader {
	return rasterHeader{
		Version:          1,
		Source:           "NASA Black Marble VJ146A4 test",
		DataYear:         2025,
		ResolutionMeters: 500,
		West:             100,
		South:            20,
		East:             102,
		North:            22,
		Width:            2,
		Height:           2,
		Scale:            0.1,
		NoData:           65535,
		Encoding:         "uint16-le",
	}
}

func TestRasterLightProviderReadsCoordinate(t *testing.T) {
	path := writeTestLightRaster(t, testRasterHeader(), []uint16{10, 20, 30, 65535})
	provider, err := OpenRasterLightProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()
	provider.now = func() time.Time { return time.Unix(1_700_000_000, 0) }

	result, err := provider.Light(context.Background(), 21.5, 100.5)
	if err != nil {
		t.Fatal(err)
	}
	if result.Radiance != 1 || result.RadianceUnit != "nW/cm²/sr" {
		t.Fatalf("radiance = %v %s, want 1 nW/cm²/sr", result.Radiance, result.RadianceUnit)
	}
	if result.DataYear != 2025 || result.ResolutionMeters != 500 || !result.Estimated {
		t.Fatalf("unexpected provenance: %+v", result)
	}
	if result.RetrievedAt != 1_700_000_000 {
		t.Fatalf("retrievedAt = %d", result.RetrievedAt)
	}
}

func TestRasterLightProviderNoCoverage(t *testing.T) {
	path := writeTestLightRaster(t, testRasterHeader(), []uint16{10, 20, 30, 65535})
	provider, err := OpenRasterLightProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()

	for _, coordinate := range [][2]float64{{20.5, 101.5}, {30, 101}, {21, 110}} {
		if _, err := provider.Light(context.Background(), coordinate[0], coordinate[1]); !errors.Is(err, ErrNoCoverage) {
			t.Errorf("coordinate %v should return ErrNoCoverage, got %v", coordinate, err)
		}
	}
}

func TestOpenRasterLightProviderRejectsTruncatedFile(t *testing.T) {
	header := testRasterHeader()
	path := writeTestLightRaster(t, header, []uint16{10})
	if _, err := OpenRasterLightProvider(path); err == nil {
		t.Fatal("truncated raster should fail validation")
	}
}

func TestRasterLightProviderRealDatasetSanity(t *testing.T) {
	path := os.Getenv("AURORA_TEST_VIIRS_PATH")
	if path == "" {
		t.Skip("set AURORA_TEST_VIIRS_PATH to validate a prepared VIIRS raster")
	}
	provider, err := OpenRasterLightProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()

	locations := map[string][2]float64{
		"上海市中心": {31.2304, 121.4737},
		"北京市中心": {39.9042, 116.4074},
		"青海荒漠":  {35.5, 93.0},
	}
	results := make(map[string]LightPollution, len(locations))
	for name, coordinate := range locations {
		value, err := provider.Light(context.Background(), coordinate[0], coordinate[1])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		results[name] = value
		t.Logf("%s: radiance=%.1f %s, SQM≈%.1f, Bortle≈%.0f", name, value.Radiance, value.RadianceUnit, value.SQM, value.Bortle)
	}
	if results["上海市中心"].Radiance <= results["青海荒漠"].Radiance || results["北京市中心"].Radiance <= results["青海荒漠"].Radiance {
		t.Fatalf("城市辐射应高于偏远荒漠：%+v", results)
	}
}
