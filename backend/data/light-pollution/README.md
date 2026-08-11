# 本地 VIIRS 光污染数据

AURORA 使用 NASA Black Marble VJ146A4 年度合成的真实卫星辐射值，并把 SQM / Bortle 明确标为估算。大体积栅格不进入 Git。

## 准备 2025 中国范围数据

原始数据由 Light Pollution Map 基于 NASA Black Marble 2.0 的 `AllAngle_Composite_Snow_Free` 子集公开，约 928MB：

```bash
curl -fL -o /tmp/viirs_2025_raw.zip https://www2.lightpollutionmap.info/data/v2/viirs_2025_raw.zip
unzip /tmp/viirs_2025_raw.zip -d /tmp/viirs_2025_raw
```

找到解压后的 `.tif`，从前端目录运行一次预处理：

```bash
cd frontend
pnpm data:light-pollution -- \
  --input /tmp/viirs_2025_raw/viirs_2025_raw.tif \
  --output ../backend/data/light-pollution/viirs-2025-cn.avnl
```

默认裁切 `72°E–136°E、18°N–54°N`，保留约 500 米原始网格，辐射量化精度为 `0.1 nW/cm²/sr`。可用 `--west --south --east --north` 改变覆盖范围。

后端 `.env` 配置：

```env
LIGHT_POLLUTION_DATA_PATH=data/light-pollution/viirs-2025-cn.avnl
```

`.avnl` 是 AURORA 的随机访问格式：启动时仅读取元数据，每次查询只读取经纬度对应的 2 字节像素，不会把整个中国栅格载入内存。

数据来源：NASA Black Marble VJ146A4；Light Pollution Map 公开的年度原始 GeoTIFF。卫星辐射是年度实测合成，SQM/Bortle 是 AURORA 的启发式估算，不应当作地面 SQM 仪器实测。
