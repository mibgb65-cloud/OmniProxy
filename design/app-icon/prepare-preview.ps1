$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$designRoot = $PSScriptRoot
$spec = Get-Content -LiteralPath (Join-Path $designRoot 'design-spec.json') -Raw -Encoding UTF8 | ConvertFrom-Json
$sizesRoot = Join-Path $designRoot 'sizes'
[System.IO.Directory]::CreateDirectory($sizesRoot) | Out-Null
$iconSizes = @(16, 20, 24, 32, 40, 48, 64, 96, 128, 256)
$canvas = [System.Drawing.Bitmap]::new(1440, 1010)
$graphics = [System.Drawing.Graphics]::FromImage($canvas)
$graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$graphics.Clear([System.Drawing.ColorTranslator]::FromHtml('#F4F5F7'))
$validation = @()

function Draw-Text {
    param([string]$Text, [float]$X, [float]$Y, [float]$Size, [string]$Color = '#171A20')
    $font = [System.Drawing.Font]::new('Microsoft YaHei UI', $Size, [System.Drawing.FontStyle]::Regular, [System.Drawing.GraphicsUnit]::Pixel)
    $brush = [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml($Color))
    try { $graphics.DrawString($Text, $font, $brush, $X, $Y) }
    finally { $font.Dispose(); $brush.Dispose() }
}

function Fill-Panel {
    param([int]$X, [int]$Y, [int]$Width, [int]$Height, [string]$Color)
    $brush = [System.Drawing.SolidBrush]::new([System.Drawing.ColorTranslator]::FromHtml($Color))
    try { $graphics.FillRectangle($brush, $X, $Y, $Width, $Height) }
    finally { $brush.Dispose() }
}

Draw-Text 'OmniProxy' 48 34 52
Draw-Text '桌面与任务栏图标 · 三个原创方向' 51 112 23 '#626A76'
$descriptions = @('双向通路 · 青绿双色 · 透明外底', '统一入口 · 石墨黑白 · 青绿切面', '多路汇聚 · 钴蓝色系 · 透明外底')

try {
    for ($index = 0; $index -lt $spec.directions.Count; $index++) {
        $direction = $spec.directions[$index]
        $sourcePath = Join-Path $designRoot ($direction.id + '.png')
        $source = [System.Drawing.Bitmap]::new($sourcePath)
        $smallImages = @{}
        try {
            if ($source.Width -ne $source.Height) { throw "图标画布必须为正方形：$($direction.id)" }
            $corners = @($source.GetPixel(0, 0).A, $source.GetPixel($source.Width - 1, 0).A,
                $source.GetPixel(0, $source.Height - 1).A, $source.GetPixel($source.Width - 1, $source.Height - 1).A)
            if (@($corners | Where-Object { $_ -ne 0 }).Count -gt 0) { throw "图标外底不是透明背景：$($direction.id)" }
            foreach ($size in $iconSizes) {
                $scaled = [System.Drawing.Bitmap]::new($size, $size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
                $scaledGraphics = [System.Drawing.Graphics]::FromImage($scaled)
                try {
                    # 透明边缘采用高质量缩放；保留原始生成稿，缩略图只用于尺寸验证和 ICO 导出。
                    $scaledGraphics.CompositingMode = [System.Drawing.Drawing2D.CompositingMode]::SourceCopy
                    $scaledGraphics.CompositingQuality = [System.Drawing.Drawing2D.CompositingQuality]::HighQuality
                    $scaledGraphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
                    $scaledGraphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
                    $imageAttributes = [System.Drawing.Imaging.ImageAttributes]::new()
                    try {
                        $imageAttributes.SetWrapMode([System.Drawing.Drawing2D.WrapMode]::TileFlipXY)
                        $scaledGraphics.DrawImage($source, [System.Drawing.Rectangle]::new(0, 0, $size, $size), 0, 0,
                            $source.Width, $source.Height, [System.Drawing.GraphicsUnit]::Pixel, $imageAttributes)
                    } finally { $imageAttributes.Dispose() }
                    $scaled.Save((Join-Path $sizesRoot "$($direction.id)-$size.png"), [System.Drawing.Imaging.ImageFormat]::Png)
                } finally { $scaledGraphics.Dispose() }
                $smallImages[$size] = $scaled
            }
            $x = 48 + $index * 456
            Fill-Panel $x 186 432 720 '#FFFFFF'
            $label = [char](65 + $index)
            Draw-Text "$label  $($direction.name)" ($x + 25) 212 30
            Draw-Text $descriptions[$index] ($x + 27) 264 17 '#66717D'
            $graphics.DrawImage($source, [System.Drawing.Rectangle]::new($x + 76, 317, 280, 280))
            Draw-Text '桌面标识放大预览' ($x + 135) 602 15 '#77818D'
            foreach ($theme in @(@{y = 655; color = '#F5F6F8'; text = '#424B56'; name = '浅色任务栏'},
                    @{y = 771; color = '#171A20'; text = '#C5CBD4'; name = '深色任务栏'})) {
                Fill-Panel ($x + 24) $theme.y 384 98 $theme.color
                Draw-Text $theme.name ($x + 39) ($theme.y + 15) 14 $theme.text
                for ($sizeIndex = 0; $sizeIndex -lt 3; $sizeIndex++) {
                    $size = @(32, 24, 16)[$sizeIndex]
                    $centerX = $x + 178 + $sizeIndex * 82
                    $graphics.DrawImageUnscaled($smallImages[$size], [int]($centerX - $size / 2), [int]($theme.y + 40 - $size / 2))
                    Draw-Text "$size px" ($centerX - 19) ($theme.y + 66) 13 $theme.text
                }
            }
            $validation += [ordered]@{ id = $direction.id; width = $source.Width; height = $source.Height;
                corner_alpha = $corners; exported_sizes = $iconSizes; status = '尺寸和透明外底检查通过' }
        } finally {
            foreach ($scaled in $smallImages.Values) { $scaled.Dispose() }
            $source.Dispose()
        }
    }
    Draw-Text '大图观察造型，小图检查辨识度。任务栏样本按实际像素绘制；设计提案尚未接入应用。' 51 943 19 '#626A76'
    $canvas.Save((Join-Path $designRoot 'comparison.png'), [System.Drawing.Imaging.ImageFormat]::Png)
    $json = $validation | ConvertTo-Json -Depth 5
    [System.IO.File]::WriteAllText((Join-Path $designRoot 'validation.json'), $json + "`n", [System.Text.UTF8Encoding]::new($false))
} finally { $graphics.Dispose(); $canvas.Dispose() }
