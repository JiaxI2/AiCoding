package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JiaxI2/AiCoding/internal/asset"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args)==0 { usage(); return 2 }
	root, _ := os.Getwd(); m := asset.NewManager(root, asset.DefaultAdapters()...); ctx:=context.Background(); var r asset.Result; var err error
	switch args[0] {
	case "validate": if len(args)!=2{return usageErr()}; _,err=asset.LoadManifest(filepath.Join(args[1],"asset.json")); r=asset.Result{OK:err==nil,Action:"validate"}
	case "pack": fs:=flag.NewFlagSet("pack",flag.ContinueOnError); out:=fs.String("out","","output archive");_ = fs.Parse(args[1:]);if fs.NArg()!=1{return usageErr()};r,err=m.Pack(ctx,fs.Arg(0),*out)
	case "install": fs:=flag.NewFlagSet("install",flag.ContinueOnError); mode:=fs.String("mode","managed","managed|editable");_ = fs.Parse(args[1:]);if fs.NArg()!=1{return usageErr()};r,err=m.Install(ctx,fs.Arg(0),asset.InstallMode(*mode))
	case "update": if len(args)!=2{return usageErr()};r,err=m.Update(ctx,args[1])
	case "uninstall": fs:=flag.NewFlagSet("uninstall",flag.ContinueOnError);purge:=fs.Bool("purge",false,"remove user override");_ = fs.Parse(args[1:]);if fs.NArg()!=1{return usageErr()};r,err=m.Uninstall(ctx,fs.Arg(0),*purge)
	case "rollback": if len(args)!=2{return usageErr()};r,err=m.Rollback(args[1])
	case "verify": if len(args)!=2{return usageErr()};err=m.Verify(ctx,args[1]);r=asset.Result{OK:err==nil,Action:"verify",AssetID:args[1]}
	case "list": var l asset.Lockfile;l,err=m.List();r=asset.Result{OK:err==nil,Action:"list",Data:map[string]any{"lock":l}}
	case "config-set": if len(args)!=4{return usageErr()};p:=filepath.Join(root,"UserCfg","assets",args[1]+".json");err=asset.SetConfig(p,args[2],args[3]);r=asset.Result{OK:err==nil,Action:"config-set",AssetID:args[1]}
	default: return usageErr()
	}
	if err!=nil { r.OK=false;r.Message=err.Error() }
	_ = json.NewEncoder(os.Stdout).Encode(r);if err!=nil{return 1};return 0
}
func usageErr() int { usage(); return 2 }
func usage(){fmt.Fprintln(os.Stderr,"assetkit validate DIR | pack DIR [--out FILE] | install PACKAGE [--mode managed|editable] | update PACKAGE | uninstall ID [--purge] | rollback ID | verify ID | list | config-set ID KEY JSON")}
