# Always-on-top banner on the *current* session desktop (no LegalNotice, no Win+L).
# RDP in → copy script → run → banner shows immediately on that session's screen.
#
#   .\show-remote-banner.ps1
#   .\show-remote-banner.ps1 -Title "正在被远程" -Text "操作员远程桌面连接中，请勿本地操作。"
#   .\show-remote-banner.ps1 -Close
#
# Closes with the on-banner button or Esc. Physical console only sees it if it is
# the same Windows session as the one running this script.

param(
  [string]$Title = '正在被远程',
  [string]$Text = '本机正在被远程桌面连接，请勿本地操作。',
  [int]$Height = 88,
  [string]$BackgroundColor = 'DarkOrange',
  [string]$ForegroundColor = 'White',
  [switch]$Close,
  [switch]$Wait
)

$ErrorActionPreference = 'Stop'
$marker = 'SemaphoreRdpRemoteBanner'

if ($Close) {
  $n = 0
  Get-CimInstance Win32_Process -Filter "Name = 'powershell.exe'" -ErrorAction SilentlyContinue | ForEach-Object {
    if ($_.CommandLine -and ($_.CommandLine -like "*$marker*")) {
      Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue
      Write-Output "BANNER_CLOSED|pid=$($_.ProcessId)"
      $n++
    }
  }
  Write-Output "BANNER_CLOSE_DONE|count=$n"
  exit 0
}

# Parent: spawn UI child (hidden console) so the RDP shell returns immediately.
if ($env:SEMAPHORE_RDP_BANNER_UI -ne '1') {
  $self = $MyInvocation.MyCommand.Path
  $ps = (Get-Process -Id $PID).Path
  # Embed marker in command line for -Close discovery.
  $cmd = @"
`$env:SEMAPHORE_RDP_BANNER_UI='1'
# $marker
& '$($self.Replace("'", "''"))' -Title '$($Title.Replace("'", "''"))' -Text '$($Text.Replace("'", "''"))' -Height $Height -BackgroundColor '$($BackgroundColor.Replace("'", "''"))' -ForegroundColor '$($ForegroundColor.Replace("'", "''"))' -Wait
"@
  $p = Start-Process -FilePath $ps -ArgumentList @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-Command', $cmd) -PassThru -WindowStyle Hidden
  Write-Output "BANNER_STARTED|pid=$($p.Id)|title=$Title"
  if ($Wait) { Wait-Process -Id $p.Id }
  exit 0
}

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$screen = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
$form = New-Object System.Windows.Forms.Form
$form.Text = $marker
$form.FormBorderStyle = 'None'
$form.TopMost = $true
$form.ShowInTaskbar = $true
$form.StartPosition = 'Manual'
$form.Bounds = New-Object System.Drawing.Rectangle($screen.Left, $screen.Top, $screen.Width, [Math]::Max(48, $Height))
try {
  $c = [System.Drawing.Color]::FromName($BackgroundColor)
  if ($c.A -eq 0) { throw 'bad' }
  $form.BackColor = $c
} catch {
  $form.BackColor = [System.Drawing.Color]::DarkOrange
}
$form.Opacity = 0.96

$fg = [System.Drawing.Color]::White
try {
  $fc = [System.Drawing.Color]::FromName($ForegroundColor)
  if ($fc.A -ne 0) { $fg = $fc }
} catch { }

$titleLabel = New-Object System.Windows.Forms.Label
$titleLabel.Text = $Title
$titleLabel.Dock = 'Top'
$titleLabel.Height = 36
$titleLabel.TextAlign = 'MiddleCenter'
$titleLabel.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 16, [System.Drawing.FontStyle]::Bold)
$titleLabel.ForeColor = $fg
$titleLabel.BackColor = [System.Drawing.Color]::Transparent

$bodyLabel = New-Object System.Windows.Forms.Label
$bodyLabel.Text = $Text
$bodyLabel.Dock = 'Fill'
$bodyLabel.TextAlign = 'MiddleCenter'
$bodyLabel.Font = New-Object System.Drawing.Font('Microsoft YaHei UI', 12, [System.Drawing.FontStyle]::Regular)
$bodyLabel.ForeColor = $fg
$bodyLabel.BackColor = [System.Drawing.Color]::Transparent
$bodyLabel.Padding = New-Object System.Windows.Forms.Padding(16, 0, 16, 10)

$closeBtn = New-Object System.Windows.Forms.Button
$closeBtn.Text = '关闭 (Esc)'
$closeBtn.Width = 110
$closeBtn.Height = 28
$closeBtn.FlatStyle = 'Flat'
$closeBtn.BackColor = [System.Drawing.Color]::FromArgb(40, 0, 0, 0)
$closeBtn.ForeColor = $fg
$closeBtn.Add_Click({ $form.Close() })
$placeClose = {
  $closeBtn.Location = New-Object System.Drawing.Point(($form.ClientSize.Width - 122), 8)
}
$form.Add_Resize($placeClose)

$form.Controls.Add($bodyLabel)
$form.Controls.Add($titleLabel)
$form.Controls.Add($closeBtn)
& $placeClose
$form.KeyPreview = $true
$form.Add_KeyDown({
    param($sender, $e)
    if ($e.KeyCode -eq 'Escape') { $form.Close() }
  })
$form.Add_Shown({ $form.Activate() })

[System.Windows.Forms.Application]::EnableVisualStyles()
[System.Windows.Forms.Application]::Run($form)
