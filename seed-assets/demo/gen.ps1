Add-Type -AssemblyName System.Drawing
$specs = @(
  @{ n = 'sunrise'; c = [System.Drawing.Color]::Orange },
  @{ n = 'ridge';   c = [System.Drawing.Color]::DarkSeaGreen },
  @{ n = 'crater';  c = [System.Drawing.Color]::Sienna }
)
foreach ($s in $specs) {
  $bmp = New-Object System.Drawing.Bitmap 640, 400
  $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.Clear($s.c)
  $g.FillEllipse([System.Drawing.Brushes]::White, 240, 140, 160, 120)
  $g.Dispose()
  $out = Join-Path $PSScriptRoot ("{0}.jpg" -f $s.n)
  $bmp.Save($out, [System.Drawing.Imaging.ImageFormat]::Jpeg)
  $bmp.Dispose()
  Write-Output "wrote $out"
}
