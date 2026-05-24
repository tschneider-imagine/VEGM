# Pull all publicly discovered old G2S / IGT XSD files from TestScriptRunner.
# Run from this folder in PowerShell on a machine with internet access.
$ErrorActionPreference = "Stop"
$Base = "https://raw.githubusercontent.com/anthony-folen-igt/TestScriptRunner/47f7acdc7f321c3fd1b1d733272864b583e422c6/WindowsApplication1/schemas"
$Files = @(
  "g2s_IncludeClasses.xsd",
  "g2sADR.xsd",
  "g2sPrinter.xsd",
  "g2sCabinet.xsd",
  "g2sNoteAcceptor.xsd",
  "g2sIncludeClasses_.xsd",
  "g2sEventHandler.xsd",
  "g2sGamePlayExtA.xsd",
  "g2sCoinAcceptor.xsd",
  "g2sIncludeConfig.xsd",
  "igtBonus-MJT-FreeSpin.xsd",
  "igtBonus-WM.xsd",
  "igtCommunications-HostVerif.xsd",
  "g2sWAT.xsd",
  "g2sIncludeClasses.xsd",
  "g2sImportExtA.xsd",
  "g2sGAT.xsd",
  "g2sIncludeGlobalExt1.xsd",
  "gtkStorage.xsd",
  "g2sMessage.xsd",
  "igtBonus-EBG.xsd",
  "igtCommConfig-smdExt.xsd",
  "g2sCommunications-withMWspecs.xsd",
  "igtMediaDisplay.xsd",
  "igtCBG.xsd",
  "igtIdReader-idNumberFilter.xsd",
  "igtBonus-ext2.xsd",
  "g2sOptionConfigExtA.xsd",
  "igtMediaDisplay-AwExt.xsd",
  "igtLicensing.xsd",
  "g2sHardware.xsd",
  "gtkCashout.xsd",
  "igtPlayerContext.xsd",
  "igtPlayer-limits.xsd",
  "g2sCommConfig.xsd",
  "gtkCabinet-oper hours.xsd",
  "g2sCentral.xsd",
  "g2sBonus.xsd",
  "igtLicensingSecurityData.xsd",
  "igtGamePlay.xsd",
  "g2sNoteDispenser.xsd",
  "igtPrinter-ext1.xsd",
  "g2sDownload.xsd",
  "g2sCommunications.xsd",
  "g2sPlayer.xsd",
  "g2sCommConfigExtA.xsd",
  "g2sMeters.xsd",
  "g2sIdReader.xsd",
  "g2sDeviceConfig.xsd",
  "igtTournament.xsd",
  "g2sHandpay.xsd",
  "igtPlayer-WM.xsd",
  "g2sVoucher.xsd",
  "g2sOptionConfig.xsd",
  "g2sHopper.xsd",
  "g2sGamePlay.xsd",
  "igtBonus-limits-full.xsd"
)
foreach ($File in $Files) {
  $UrlName = $File -replace ' ', '%20'
  $Url = "$Base/$UrlName"
  Write-Host "Downloading $File"
  Invoke-WebRequest -Uri $Url -OutFile $File
}
Write-Host "Done. Validate with: Get-ChildItem *.xsd | Measure-Object"
