param(
 [Parameter(Mandatory=$true)][string]$Application,
 [Parameter(Mandatory=$true)][string]$Installer,
 [Parameter(Mandatory=$true)][string]$Version
)
$ErrorActionPreference='Stop'
if(-not('OmniProxy.ReleaseMetadata' -as [type])){
Add-Type -TypeDefinition @'
using System;using System.Collections.Generic;using System.Runtime.InteropServices;
namespace OmniProxy {
 public static class ReleaseMetadata {
  [DllImport("version.dll",CharSet=CharSet.Unicode,ExactSpelling=true)]static extern uint GetFileVersionInfoSizeW(string f,out uint h);
  [DllImport("version.dll",CharSet=CharSet.Unicode,ExactSpelling=true)]static extern bool GetFileVersionInfoW(string f,uint h,uint n,IntPtr b);
  [DllImport("version.dll",CharSet=CharSet.Unicode,ExactSpelling=true)]static extern bool VerQueryValueW(IntPtr b,string key,out IntPtr value,out uint n);
  static IntPtr Query(IntPtr b,string key){IntPtr v;uint n;if(!VerQueryValueW(b,key,out v,out n)||n==0)throw new Exception("缺少版本资源: "+key);return v;}
  static string Number(IntPtr p,int offset){uint ms=unchecked((uint)Marshal.ReadInt32(p,offset));uint ls=unchecked((uint)Marshal.ReadInt32(p,offset+4));return (ms>>16)+"."+(ms&65535)+"."+(ls>>16)+"."+(ls&65535);}
  public static Dictionary<string,string> Read(string f){uint h;uint n=GetFileVersionInfoSizeW(f,out h);if(n==0)throw new Exception("没有版本资源");IntPtr b=Marshal.AllocHGlobal((int)n);try{
   if(!GetFileVersionInfoW(f,0,n,b))throw new Exception("读取版本资源失败");
   var r=new Dictionary<string,string>();IntPtr fixedInfo=Query(b,"\\");r["FileVersionNumber"]=Number(fixedInfo,8);r["ProductVersionNumber"]=Number(fixedInfo,16);
   IntPtr translation=Query(b,"\\VarFileInfo\\Translation");string prefix="\\StringFileInfo\\"+unchecked((ushort)Marshal.ReadInt16(translation)).ToString("x4")+unchecked((ushort)Marshal.ReadInt16(translation,2)).ToString("x4")+"\\";
   foreach(string key in new[]{"ProductVersion","ProductName","CompanyName"})r[key]=Marshal.PtrToStringUni(Query(b,prefix+key));return r;
  }finally{Marshal.FreeHGlobal(b);}}
 }
}
'@
}
$full=$Version.TrimStart('v');$numeric=($full -split '-',2)[0]
foreach($item in @(@{Path=$Application;Expected=$full},@{Path=$Installer;Expected=$numeric})){
 $resolved=(Resolve-Path -LiteralPath $item.Path).Path
 $metadata=[OmniProxy.ReleaseMetadata]::Read($resolved)
 if($metadata['ProductVersion'] -ne $item.Expected -or $metadata['FileVersionNumber'] -ne "$numeric.0" -or $metadata['ProductVersionNumber'] -ne "$numeric.0"){throw "版本资源不匹配：$resolved"}
 if($metadata['ProductName'] -ne 'OmniProxy' -or $metadata['CompanyName'] -ne 'OmniProxy'){throw "产品身份不匹配：$resolved"}
}
Write-Output 'Windows 应用和安装器版本资源校验通过'
