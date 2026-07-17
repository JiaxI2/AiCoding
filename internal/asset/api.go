package asset

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Adapter interface {
	Type() Type
	Validate(context.Context, Manifest, string) error
	AfterInstall(context.Context, Manifest, string, map[string]any) error
	BeforeUninstall(context.Context, LockEntry, string) error
	Verify(context.Context, Manifest, string) error
}

type Source interface { Resolve(context.Context, string) (string, error) }
type Executor interface { Run(context.Context, string, string) error }

type Manager struct {
	Root string
	Adapters map[Type]Adapter
	Now func() time.Time
}

func NewManager(root string, adapters ...Adapter) *Manager {
	m := &Manager{Root: root, Adapters: map[Type]Adapter{}, Now: time.Now}
	for _, a := range adapters { m.Adapters[a.Type()] = a }
	return m
}

func LoadManifest(path string) (Manifest, error) {
	b, err := os.ReadFile(path); if err != nil { return Manifest{}, err }
	var m Manifest; if err = json.Unmarshal(b, &m); err != nil { return m, err }
	return m, ValidateManifest(m)
}

func ValidateManifest(m Manifest) error {
	if m.SchemaVersion != 1 { return fmt.Errorf("unsupported schemaVersion %d", m.SchemaVersion) }
	if m.ID == "" || strings.ContainsAny(m.ID, "\\/:*?\"<>|") { return errors.New("invalid asset id") }
	if m.Version == "" { return errors.New("version is required") }
	switch m.Type { case TypeKit, TypeSkill, TypeMCP, TypeTemplate, TypeRuleset, TypeProfile: default: return fmt.Errorf("unsupported asset type %q", m.Type) }
	if m.Paths.Payload == "" || filepath.IsAbs(m.Paths.Payload) || strings.Contains(filepath.Clean(m.Paths.Payload), "..") { return errors.New("payload must be a safe relative path") }
	return nil
}

func MergeConfig(l ConfigLayers) map[string]any {
	out := map[string]any{}
	for _, layer := range []map[string]any{l.Defaults,l.Repository,l.User,l.Local,l.CLI} { deepMerge(out, layer) }
	return out
}
func deepMerge(dst, src map[string]any) {
	for k,v := range src {
		if vm,ok := v.(map[string]any); ok { dm,_ := dst[k].(map[string]any); if dm==nil { dm=map[string]any{} }; deepMerge(dm,vm); dst[k]=dm; continue }
		dst[k]=v
	}
}
func LoadJSONMap(path string) (map[string]any,error) {
	if path=="" { return map[string]any{},nil }; b,err:=os.ReadFile(path); if errors.Is(err,os.ErrNotExist){return map[string]any{},nil}; if err!=nil{return nil,err}; var v map[string]any; err=json.Unmarshal(b,&v); return v,err
}
func SetConfig(path, dotted, raw string) error {
	v:=any(raw); if err:=json.Unmarshal([]byte(raw),&v); err!=nil { v=raw }
	m,err:=LoadJSONMap(path); if err!=nil{return err}; cur:=m; parts:=strings.Split(dotted,"."); for _,p:=range parts[:len(parts)-1]{ n,_:=cur[p].(map[string]any); if n==nil{n=map[string]any{};cur[p]=n};cur=n };cur[parts[len(parts)-1]]=v
	return writeJSONAtomic(path,m)
}

func (m *Manager) Pack(ctx context.Context, dir, output string) (Result,error) {
	manifest,err:=LoadManifest(filepath.Join(dir,"asset.json")); if err!=nil{return Result{},err}; if a:=m.Adapters[manifest.Type];a!=nil{if err=a.Validate(ctx,manifest,dir);err!=nil{return Result{},err}}
	if output==""{output=manifest.ID+"-"+manifest.Version+".aicoding.zip"}; if err=zipDir(dir,output);err!=nil{return Result{},err}; d,err:=fileDigest(output); if err!=nil{return Result{},err}
	return Result{OK:true,Action:"pack",AssetID:manifest.ID,Changed:[]string{output},Data:map[string]any{"digest":d}},nil
}

func (m *Manager) Install(ctx context.Context, pkg string, mode InstallMode) (Result,error) {
	if mode!=ModeManaged && mode!=ModeEditable{return Result{},errors.New("mode must be managed or editable")}
	stage,err:=os.MkdirTemp(filepath.Join(m.Root,".aicoding"),"asset-stage-");if err!=nil{return Result{},err};defer os.RemoveAll(stage)
	if err=unzipSafe(pkg,stage);err!=nil{return Result{},err};manifest,err:=LoadManifest(filepath.Join(stage,"asset.json"));if err!=nil{return Result{},err};if a:=m.Adapters[manifest.Type];a!=nil{if err=a.Validate(ctx,manifest,stage);err!=nil{return Result{},err}}
	if err=m.checkDependencies(manifest);err!=nil{return Result{},err};target:=filepath.Join(m.Root,".aicoding","assets",manifest.ID);backup:=target+".rollback";_ = os.RemoveAll(backup);if _,e:=os.Stat(target);e==nil{if err=os.Rename(target,backup);err!=nil{return Result{},err}}
	payload:=filepath.Join(stage,manifest.Paths.Payload);if err=copyTree(payload,target);err!=nil{_ = os.Rename(backup,target);return Result{},err};digest,err:=treeDigest(target);if err!=nil{return Result{},err};files,_:=listFiles(target)
	lock,err:=m.loadLock();if err!=nil{return Result{},err};lock.Assets[manifest.ID]=LockEntry{ID:manifest.ID,Type:manifest.Type,Version:manifest.Version,Mode:mode,Source:pkg,Digest:digest,InstalledAt:m.Now().UTC(),Files:files};if err=m.saveLock(lock);err!=nil{return Result{},err}
	cfg,err:=m.EffectiveConfig(manifest);if err!=nil{return Result{},err};if a:=m.Adapters[manifest.Type];a!=nil{if err=a.AfterInstall(ctx,manifest,target,cfg);err!=nil{return Result{},err}}
	return Result{OK:true,Action:"install",AssetID:manifest.ID,Changed:files},nil
}

func (m *Manager) Update(ctx context.Context,pkg string)(Result,error){r,err:=m.Install(ctx,pkg,ModeManaged);r.Action="update";return r,err}
func (m *Manager) Uninstall(ctx context.Context,id string,purge bool)(Result,error){lock,err:=m.loadLock();if err!=nil{return Result{},err};e,ok:=lock.Assets[id];if !ok{return Result{},fmt.Errorf("asset %s is not installed",id)};target:=filepath.Join(m.Root,".aicoding","assets",id);if a:=m.Adapters[e.Type];a!=nil{if err=a.BeforeUninstall(ctx,e,target);err!=nil{return Result{},err}};if err=os.RemoveAll(target);err!=nil{return Result{},err};delete(lock.Assets,id);if err=m.saveLock(lock);err!=nil{return Result{},err};if purge{_ = os.Remove(filepath.Join(m.Root,"UserCfg","assets",id+".json"))};return Result{OK:true,Action:"uninstall",AssetID:id,Changed:e.Files},nil}
func (m *Manager) Rollback(id string)(Result,error){target:=filepath.Join(m.Root,".aicoding","assets",id);backup:=target+".rollback";if _,err:=os.Stat(backup);err!=nil{return Result{},fmt.Errorf("no rollback snapshot for %s",id)};failed:=target+".failed";_ = os.RemoveAll(failed);_ = os.Rename(target,failed);if err:=os.Rename(backup,target);err!=nil{_ = os.Rename(failed,target);return Result{},err};return Result{OK:true,Action:"rollback",AssetID:id},nil}
func (m *Manager) List()(Lockfile,error){return m.loadLock()}
func (m *Manager) Verify(ctx context.Context,id string)error{lock,err:=m.loadLock();if err!=nil{return err};e,ok:=lock.Assets[id];if !ok{return fmt.Errorf("asset %s is not installed",id)};target:=filepath.Join(m.Root,".aicoding","assets",id);d,err:=treeDigest(target);if err!=nil{return err};if d!=e.Digest{return errors.New("installed asset checksum mismatch")};if a:=m.Adapters[e.Type];a!=nil{return a.Verify(ctx,Manifest{ID:e.ID,Type:e.Type,Version:e.Version,Paths:Paths{Payload:"."}},target)};return nil}
func (m *Manager) EffectiveConfig(man Manifest)(map[string]any,error){d,_:=LoadJSONMap(filepath.Join(m.Root,".aicoding","assets",man.ID,"config.default.json"));r,_:=LoadJSONMap(filepath.Join(m.Root,"config","assets",man.ID+".json"));u,_:=LoadJSONMap(filepath.Join(m.Root,"UserCfg","assets",man.ID+".json"));l,_:=LoadJSONMap(filepath.Join(m.Root,".aicoding","local","assets",man.ID+".json"));return MergeConfig(ConfigLayers{Defaults:d,Repository:r,User:u,Local:l}),nil}
func (m *Manager) checkDependencies(man Manifest)error{lock,err:=m.loadLock();if err!=nil{return err};var missing []string;for _,d:=range man.Dependencies{if _,ok:=lock.Assets[d.ID];!ok&&!d.Optional{missing=append(missing,d.ID)}};if len(missing)>0{return fmt.Errorf("missing required assets: %s",strings.Join(missing,", "))};return nil}
func (m *Manager) loadLock()(Lockfile,error){p:=filepath.Join(m.Root,".aicoding","assets.lock.json");b,err:=os.ReadFile(p);if errors.Is(err,os.ErrNotExist){return Lockfile{SchemaVersion:1,Assets:map[string]LockEntry{}},nil};if err!=nil{return Lockfile{},err};var l Lockfile;if err=json.Unmarshal(b,&l);err!=nil{return l,err};if l.Assets==nil{l.Assets=map[string]LockEntry{}};return l,nil}
func (m *Manager) saveLock(l Lockfile)error{return writeJSONAtomic(filepath.Join(m.Root,".aicoding","assets.lock.json"),l)}
func writeJSONAtomic(path string,v any)error{if err:=os.MkdirAll(filepath.Dir(path),0755);err!=nil{return err};b,err:=json.MarshalIndent(v,"","  ");if err!=nil{return err};b=append(b,'\n');tmp:=path+".tmp";if err=os.WriteFile(tmp,b,0644);err!=nil{return err};return os.Rename(tmp,path)}
func zipDir(root,out string)error{f,err:=os.Create(out);if err!=nil{return err};defer f.Close();zw:=zip.NewWriter(f);defer zw.Close();return filepath.WalkDir(root,func(p string,d fs.DirEntry,e error)error{if e!=nil{return e};if d.IsDir(){return nil};rel,_:=filepath.Rel(root,p);w,e:=zw.Create(filepath.ToSlash(rel));if e!=nil{return e};r,e:=os.Open(p);if e!=nil{return e};defer r.Close();_,e=io.Copy(w,r);return e})}
func unzipSafe(pkg,dst string)error{r,err:=zip.OpenReader(pkg);if err!=nil{return err};defer r.Close();for _,f:=range r.File{p:=filepath.Join(dst,filepath.FromSlash(f.Name));if !strings.HasPrefix(filepath.Clean(p),filepath.Clean(dst)+string(os.PathSeparator)){return errors.New("unsafe archive path")};if f.FileInfo().IsDir(){continue};if err=os.MkdirAll(filepath.Dir(p),0755);err!=nil{return err};in,e:=f.Open();if e!=nil{return e};out,e:=os.Create(p);if e!=nil{in.Close();return e};_,e=io.Copy(out,in);in.Close();out.Close();if e!=nil{return e}};return nil}
func copyTree(src,dst string)error{return filepath.WalkDir(src,func(p string,d fs.DirEntry,e error)error{if e!=nil{return e};rel,_:=filepath.Rel(src,p);q:=filepath.Join(dst,rel);if d.IsDir(){return os.MkdirAll(q,0755)};in,e:=os.Open(p);if e!=nil{return e};defer in.Close();out,e:=os.Create(q);if e!=nil{return e};defer out.Close();_,e=io.Copy(out,in);return e})}
func listFiles(root string)([]string,error){var a []string;err:=filepath.WalkDir(root,func(p string,d fs.DirEntry,e error)error{if e!=nil{return e};if !d.IsDir(){r,_:=filepath.Rel(root,p);a=append(a,filepath.ToSlash(r))};return nil});sort.Strings(a);return a,err}
func treeDigest(root string)(string,error){files,err:=listFiles(root);if err!=nil{return "",err};h:=sha256.New();for _,f:=range files{io.WriteString(h,f+"\n");b,e:=os.ReadFile(filepath.Join(root,filepath.FromSlash(f)));if e!=nil{return "",e};h.Write(b)};return hex.EncodeToString(h.Sum(nil)),nil}
func fileDigest(path string)(string,error){b,err:=os.ReadFile(path);if err!=nil{return "",err};s:=sha256.Sum256(b);return hex.EncodeToString(s[:]),nil}
